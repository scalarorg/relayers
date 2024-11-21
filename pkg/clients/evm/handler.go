package evm

import (
	"context"
	"encoding/hex"
	"fmt"

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
	err = ec.dbAdapter.CreateSingleValue(relayData)
	if err != nil {
		return fmt.Errorf("failed to create evm contract call: %w", err)
	}
	//2. Send to the bus
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: ec.evmConfig.GetId(),
		TxHashs:   []string{relayData.CallContract.TxHash},
	}

	ec.eventBus.BroadcastEvent(&events.EventEnvelope{
		EventType:        events.EVENT_EVM_CONTRACT_CALL,
		DestinationChain: scalar.SCALAR_NETWORK_NAME,
		Data:             confirmTxs,
	})
	return nil
}
func (ec *EvmClient) preprocessContractCall(event *contracts.IAxelarGatewayContractCall) error {
	log.Info().Msgf("Start handle Contract call: %v", event)
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) handleContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) error {
	//0. Preprocess the event
	ec.preprocessContractCallApproved(event)
	//1. Convert into a RelayData instance then store to the db
	contractCallApproved, err := ec.ContractCallApprovedEvent2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallApprovedEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(contractCallApproved)
	if err != nil {
		return fmt.Errorf("failed to create contract call approved: %w", err)
	}
	// Find relayData from the db by combination (contractAddress, sourceAddress, payloadHash)
	// This contract call (initiated by the user call to the source chain) is approved by EVM network
	// So anyone can execute it on the EVM by broadcast the corresponding payload to protocol's smart contract on the destination chain
	contractCall := models.CallContract{
		ContractAddress: event.ContractAddress.Hex(),
		SourceAddress:   event.SourceAddress,
		PayloadHash:     hex.EncodeToString(event.PayloadHash[:]),
	}
	relayDatas, err := ec.dbAdapter.FindRelayDataByContractCall(&contractCall)
	if err != nil {
		log.Error().Msgf("[EvmClient] [handleContractCallApproved] find relay data with error: %v", err)
		return err
	}
	log.Info().Msgf("[EvmClient] [handleContractCallApproved] found relay data: %v", relayDatas)
	//3. Execute payload in the found relaydatas
	executeResults, err := ec.executeDestinationCall(event, relayDatas)
	if err != nil {
		return fmt.Errorf("failed to execute relay datas: %w", err)
	}
	log.Info().Msgf("[EvmClient] [handleContractCallApproved] execute results: %v", executeResults)
	//Done; Don't need to send to the bus
	return nil
}
func (ec *EvmClient) executeDestinationCall(event *contracts.IAxelarGatewayContractCallApproved, relayDatas []models.RelayData) ([]db.RelaydataExecuteResult, error) {
	executeResults := []db.RelaydataExecuteResult{}
	executed, err := ec.isContractCallExecuted(event)
	if err != nil {
		return nil, fmt.Errorf("[EvmClient] [executeDestinationCall] failed to check if contract call is approved: %w", err)
	}
	if executed {
		//Update the relay data status to executed
		for _, relayData := range relayDatas {
			executeResults = append(executeResults, db.RelaydataExecuteResult{
				Status:      db.SUCCESS,
				RelayDataId: relayData.ID,
			})
		}
		return executeResults, fmt.Errorf("[EvmClient] [executeDestinationCall] destination contract call is already executed")
	}
	if len(relayDatas) > 0 {
		for _, relayData := range relayDatas {
			if len(relayData.CallContract.Payload) == 0 {
				continue
			}
			log.Info().Msgf("[EvmClient] [executeDestinationCall] execute payload: %v", relayData.CallContract.Payload)
			receipt, err := ec.ExecuteDestinationCall(event.ContractAddress, event.CommandId, event.SourceChain, event.SourceAddress, relayData.CallContract.Payload)
			if err != nil {
				return nil, fmt.Errorf("[EvmClient] [executeDestinationCall] failed to execute payload: %w", err)
			}
			log.Info().Msgf("[EvmClient] [executeDestinationCall] execute receipt: %v", receipt)
		}
	}
	return executeResults, nil
}

// Check if ContractCall is already executed
func (ec *EvmClient) isContractCallExecuted(event *contracts.IAxelarGatewayContractCallApproved) (bool, error) {
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
	log.Info().Msgf("[EvmClient] [handleContractCallApproved] Start handle Contract call approved: %v", event)
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) handleCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	//0. Preprocess the event
	ec.preprocessCommandExecuted(event)
	//1. Convert into a RelayData instance then store to the db
	cmdExecuted, err := ec.CommandExecutedEvent2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert EVMExecutedEvent to RelayData: %w", err)
	}
	err = ec.dbAdapter.CreateSingleValue(cmdExecuted)
	if err != nil {
		return fmt.Errorf("failed to create evm executed: %w", err)
	}
	//Done; Don't need to send to the bus
	return nil
}

func (ec *EvmClient) preprocessCommandExecuted(event *contracts.IAxelarGatewayExecuted) error {
	log.Info().Msgf("Start handle EVM Command executed: %v", event)
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
		return nil, fmt.Errorf("failed to create executable contract: %w", err)
	}
	//Return signed transaction
	signedTx, err := executable.Execute(ec.auth, commandId, sourceChain, sourceAddress, payload)
	if err != nil {
		log.Error().Msgf("[EvmClient] [ExecuteDestinationCall] Failed to execute destination contract: %v", err)
		return nil, err
	}

	return signedTx, nil
}

// func (ec *EvmClient) SubmitTx(tx *ethtypes.Transaction, retryAttempt int) (*ethtypes.Receipt, error) {
// 	if retryAttempt >= ec.evmConfig.MaxRetry {
// 		return nil, fmt.Errorf("max retry exceeded")
// 	}

// 	// Create a new context with timeout
// 	ctx, cancel := context.WithTimeout(context.Background(), ec.evmConfig.TxTimeout)
// 	defer cancel()

// 	// Log transaction details
// 	log.Debug().
// 		Interface("tx", tx).
// 		Msg("Submitting transaction")

// 	// Send the transaction using the new context
// 	err := ec.Client.SendTransaction(ctx, tx)
// 	if err != nil {
// 		log.Error().
// 			Err(err).
// 			Str("rpcUrl", ec.evmConfig.RPCUrl).
// 			Str("walletAddress", ec.auth.From.String()).
// 			Str("to", tx.To().String()).
// 			Str("data", hex.EncodeToString(tx.Data())).
// 			Msg("[EvmClient.SubmitTx] Failed to submit transaction")

// 		// Sleep before retry
// 		time.Sleep(ec.evmConfig.RetryDelay)

// 		log.Debug().
// 			Int("attempt", retryAttempt+1).
// 			Msg("Retrying transaction")

// 		return ec.SubmitTx(tx, retryAttempt+1)
// 	}

// 	// Wait for transaction receipt using the new context
// 	receipt, err := bind.WaitMined(ctx, ec.Client, tx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to wait for transaction receipt: %w", err)
// 	}

// 	log.Debug().
// 		Interface("receipt", receipt).
// 		Msg("Transaction receipt received")

// 	return receipt, nil
// }

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
