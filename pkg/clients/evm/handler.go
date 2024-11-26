package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
)

func (ec *EvmClient) handleContractCall(event *contracts.IAxelarGatewayContractCall) error {
	//0. Preprocess the event
	ec.preprocessContractCall(event)
	//1. Convert into a RelayData instance then store to the db
	relayData, err := ec.ContractCallEvent2RelayData(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallEvent to RelayData: %w", err)
	}
	//2. update last checkpoint
	lastCheckpoint, err := ec.dbAdapter.GetLastEventCheckPoint(ec.evmConfig.GetId(), events.EVENT_EVM_CONTRACT_CALL)
	if err != nil {
		log.Debug().Str("chainId", ec.evmConfig.GetId()).
			Str("eventName", events.EVENT_EVM_CONTRACT_CALL).
			Msg("[EvmClient] [handleContractCall] Get event from begining")
	}
	if event.Raw.BlockNumber > lastCheckpoint.BlockNumber ||
		(event.Raw.BlockNumber == lastCheckpoint.BlockNumber && event.Raw.TxIndex > lastCheckpoint.LogIndex) {
		lastCheckpoint.BlockNumber = event.Raw.BlockNumber
		lastCheckpoint.TxHash = event.Raw.TxHash.String()
		lastCheckpoint.LogIndex = event.Raw.Index
		lastCheckpoint.EventKey = fmt.Sprintf("%s-%d-%d", event.Raw.TxHash.String(), event.Raw.BlockNumber, event.Raw.TxIndex)
	}
	//3. store relay data to the db, update last checkpoint
	err = ec.dbAdapter.CreateRelayDatas([]models.RelayData{relayData}, lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to create evm contract call: %w", err)
	}
	//2. Send to the bus
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: ec.evmConfig.GetId(),
		TxHashs:   map[string]string{relayData.CallContract.TxHash: relayData.To},
	}
	if ec.eventBus != nil {
		ec.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_CONTRACT_CALL,
			DestinationChain: scalar.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[EvmClient] [handleContractCall] event bus is undefined")
	}
	return nil
}
func (ec *EvmClient) preprocessContractCall(event *contracts.IAxelarGatewayContractCall) error {
	log.Info().
		Str("sender", event.Sender.Hex()).
		Str("destinationChain", event.DestinationChain).
		Str("destinationContractAddress", event.DestinationContractAddress).
		Str("payloadHash", hex.EncodeToString(event.PayloadHash[:])).
		Msg("[EvmClient] [preprocessContractCall] Start handle Contract call")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) HandleContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) error {
	//0. Preprocess the event
	err := ec.preprocessContractCallApproved(event)
	if err != nil {
		return fmt.Errorf("failed to preprocess contract call approved: %w", err)
	}
	//1. Convert into a RelayData instance then store to the db
	contractCallApproved, err := ec.ContractCallApprovedEvent2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallApprovedEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(&contractCallApproved)
	if err != nil {
		return fmt.Errorf("failed to create contract call approved: %w", err)
	}
	// Find relayData from the db by combination (contractAddress, sourceAddress, payloadHash)
	// This contract call (initiated by the user call to the source chain) is approved by EVM network
	// So anyone can execute it on the EVM by broadcast the corresponding payload to protocol's smart contract on the destination chain
	contractCall := models.CallContract{
		ContractAddress: strings.TrimLeft(event.ContractAddress.Hex(), "0x"),
		SourceAddress:   strings.TrimLeft(event.SourceAddress, "0x"),
		PayloadHash:     strings.TrimLeft(hex.EncodeToString(event.PayloadHash[:]), "0x"),
	}
	relayDatas, err := ec.dbAdapter.FindRelayDataByContractCall(&contractCall)
	if err != nil {
		log.Error().Err(err).Msg("[EvmClient] [handleContractCallApproved] find relay data")
		return err
	}
	log.Debug().Str("contractAddress", event.ContractAddress.String()).
		Str("sourceAddress", event.SourceAddress).
		Str("payloadHash", hex.EncodeToString(event.PayloadHash[:])).
		Any("relayDatas", relayDatas).
		Msg("[EvmClient] [handleContractCallApproved] query relaydata by ContractCall")
	//3. Execute payload in the found relaydatas
	executeResults, err := ec.executeDestinationCall(event, relayDatas)
	if err != nil {
		log.Warn().Err(err).Any("executeResults", executeResults).Msg("[EvmClient] [handleContractCallApproved] execute destination call")
	}
	// Done; Don't need to send to the bus
	// TODO: Do we need to update relay data atomically?
	err = ec.dbAdapter.UpdateBatchRelayDataStatus(executeResults, len(executeResults))
	if err != nil {
		return fmt.Errorf("failed to update relay data status to executed: %w", err)
	}
	return nil
}
func (ec *EvmClient) executeDestinationCall(event *contracts.IAxelarGatewayContractCallApproved, relayDatas []models.RelayData) ([]db.RelaydataExecuteResult, error) {
	executeResults := []db.RelaydataExecuteResult{}
	executed, err := ec.isContractCallExecuted(event)
	if err != nil {
		return executeResults, fmt.Errorf("[EvmClient] [executeDestinationCall] failed to check if contract call is approved: %w", err)
	}
	if executed {
		//Update the relay data status to executed
		for _, relayData := range relayDatas {
			executeResults = append(executeResults, db.RelaydataExecuteResult{
				Status:      db.SUCCESS,
				RelayDataId: relayData.ID,
			})
		}
		return executeResults, fmt.Errorf("destination contract call is already executed")
	}
	if len(relayDatas) > 0 {
		for _, relayData := range relayDatas {
			if len(relayData.CallContract.Payload) == 0 {
				continue
			}
			log.Info().Str("payload", hex.EncodeToString(relayData.CallContract.Payload)).
				Msg("[EvmClient] [executeDestinationCall]")
			receipt, err := ec.ExecuteDestinationCall(event.ContractAddress, event.CommandId, event.SourceChain, event.SourceAddress, relayData.CallContract.Payload)
			if err != nil {
				return executeResults, fmt.Errorf("execute destination call with error: %w", err)
			}

			log.Info().Any("txReceipt", receipt).Msg("[EvmClient] [executeDestinationCall]")

			if receipt.Hash() != (common.Hash{}) {
				executeResults = append(executeResults, db.RelaydataExecuteResult{
					Status:      db.SUCCESS,
					RelayDataId: relayData.ID,
				})
			} else {
				executeResults = append(executeResults, db.RelaydataExecuteResult{
					Status:      db.FAILED,
					RelayDataId: relayData.ID,
				})
			}
		}
	}
	return executeResults, nil
}

// Check if ContractCall is already executed
func (ec *EvmClient) isContractCallExecuted(event *contracts.IAxelarGatewayContractCallApproved) (bool, error) {
	if ec.auth == nil {
		log.Error().
			Str("commandId", hex.EncodeToString(event.CommandId[:])).
			Str("sourceChain", event.SourceChain).
			Str("sourceAddress", event.SourceAddress).
			Str("contractAddress", event.ContractAddress.String()).
			Msg("[EvmClient] [isContractCallExecuted] auth is nil")
		return false, fmt.Errorf("auth is nil")
	}
	callOpt := &bind.CallOpts{
		From:    ec.auth.From,
		Context: context.Background(),
	}
	approved, err := ec.Gateway.IsContractCallApproved(callOpt, event.CommandId, event.SourceChain, event.SourceAddress, event.ContractAddress, event.PayloadHash)
	if err != nil {
		return false, fmt.Errorf("failed to check if contract call is approved: %w", err)
	}
	return !approved, nil
}

func (ec *EvmClient) preprocessContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) error {
	log.Info().Any("event", event).Msgf("[EvmClient] [handleContractCallApproved]")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) HandleCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	//0. Preprocess the event
	ec.preprocessCommandExecuted(event)
	//1. Convert into a RelayData instance then store to the db
	cmdExecuted := ec.CommandExecutedEvent2Model(event)
	//Find the ContractCall by sourceTxHash and sourceEventIndex
	// contractCall, err := ec.dbAdapter.FindContractCallByCommnadId(cmdExecuted.CommandId)
	// if err != nil {
	// 	return fmt.Errorf("failed to find contract call by sourceTxHash and sourceEventIndex: %w", err)
	// }
	// cmdExecuted.CallContract = contractCall
	// cmdExecuted.ReferenceTxHash = &contractCall.TxHash
	err := ec.dbAdapter.CreateSingleValue(&cmdExecuted)
	if err != nil {
		return fmt.Errorf("failed to create evm executed: %w", err)
	}
	//Done; Don't need to send to the bus
	return nil
}

func (ec *EvmClient) preprocessCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	log.Info().Any("event", event).Msg("[EvmClient] [ExecutedHandler] Start processing evm command executed")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) ExecuteDestinationCall(
	contractAddress common.Address,
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	payload []byte,
) (*ethtypes.Transaction, error) {
	executable, err := contracts.NewIAxelarExecutable(contractAddress, ec.Client)
	if err != nil {
		log.Error().Err(err).Any("contractAddress", contractAddress).Msg("[EvmClient] [ExecuteDestinationCall] create executable contract")
		return nil, err
	}
	if ec.auth == nil {
		return nil, fmt.Errorf("auth is nil")
	}
	//Return signed transaction
	signedTx, err := executable.Execute(ec.auth, commandId, sourceChain, sourceAddress, payload)
	if err != nil {
		log.Error().Err(err).
			Str("commandId", hex.EncodeToString(commandId[:])).
			Str("sourceChain", sourceChain).
			Str("sourceAddress", sourceAddress).
			Str("contractAddress", contractAddress.String()).
			Msg("[EvmClient] [ExecuteDestinationCall]")
		//Retry
		return nil, err
	}
	//Remove pending tx
	ec.pendingTxs.RemoveTx(signedTx.Hash().Hex())
	//Resubmit the transaction to the network
	// receipt, err := ec.SubmitTx(signedTx, 0)
	// if err != nil {
	// 	return nil, err
	// }

	// return receipt, nil
	return signedTx, nil
}

func (ec *EvmClient) SubmitTx(signedTx *ethtypes.Transaction, retryAttempt int) (*ethtypes.Receipt, error) {
	if retryAttempt >= ec.evmConfig.MaxRetry {
		return nil, fmt.Errorf("max retry exceeded")
	}

	// Create a new context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ec.evmConfig.TxTimeout)
	defer cancel()

	// Log transaction details
	log.Debug().
		Interface("tx", signedTx).
		Msg("Submitting transaction")

	// Send the transaction using the new context
	err := ec.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Error().
			Err(err).
			Str("rpcUrl", ec.evmConfig.RPCUrl).
			Str("walletAddress", ec.auth.From.String()).
			Str("to", signedTx.To().String()).
			Str("data", hex.EncodeToString(signedTx.Data())).
			Msg("[EvmClient.SubmitTx] Failed to submit transaction")

		// Sleep before retry
		time.Sleep(ec.evmConfig.RetryDelay)

		log.Debug().
			Int("attempt", retryAttempt+1).
			Msg("Retrying transaction")

		return ec.SubmitTx(signedTx, retryAttempt+1)
	}

	// Wait for transaction receipt using the new context
	receipt, err := bind.WaitMined(ctx, ec.Client, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction receipt: %w", err)
	}

	log.Debug().
		Interface("receipt", receipt).
		Msg("Transaction receipt received")

	return receipt, nil
}

func (ec *EvmClient) WaitForTransaction(hash string) (*ethtypes.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ec.evmConfig.TxTimeout)
	defer cancel()

	txHash := common.HexToHash(hash)
	tx, _, err := ec.Client.TransactionByHash(ctx, txHash)
	if err != nil {
		return nil, err
	}
	return bind.WaitMined(ctx, ec.Client, tx)
}
