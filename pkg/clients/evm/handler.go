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
	"github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

func (ec *EvmClient) handleContractCall(event *contracts.IScalarGatewayContractCall) error {
	//0. Preprocess the event
	ec.preprocessContractCall(event)
	//1. Convert into a RelayData instance then store to the db
	contractCall, err := ec.ContractCallEvent2Model(event)
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
		lastCheckpoint.EventKey = fmt.Sprintf("%s-%d-%d", event.Raw.TxHash.String(), event.Raw.BlockNumber, event.Raw.Index)
	}
	//3. store relay data to the db, update last checkpoint
	err = ec.dbAdapter.CreateContractCall(contractCall, lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to create evm contract call: %w", err)
	}
	//2. Send to the bus
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: ec.evmConfig.GetId(),
		TxHashs:   map[string]string{contractCall.TxHash: contractCall.DestinationChain},
	}
	if ec.eventBus != nil {
		ec.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_CONTRACT_CALL,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[EvmClient] [handleContractCall] event bus is undefined")
	}
	return nil
}
func (ec *EvmClient) preprocessContractCall(event *contracts.IScalarGatewayContractCall) error {
	log.Info().
		Str("sender", event.Sender.Hex()).
		Str("destinationChain", event.DestinationChain).
		Str("destinationContractAddress", event.DestinationContractAddress).
		Str("payloadHash", hex.EncodeToString(event.PayloadHash[:])).
		Str("txHash", event.Raw.TxHash.String()).
		Uint("logIndex", event.Raw.Index).
		Uint("txIndex", event.Raw.TxIndex).
		Str("logData", hex.EncodeToString(event.Raw.Data)).
		Msg("[EvmClient] [preprocessContractCall] Start handle Contract call")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) handleContractCallWithToken(event *contracts.IScalarGatewayContractCallWithToken) error {
	//0. Preprocess the event
	ec.preprocessContractCallWithToken(event)
	//1. Convert into a RelayData instance then store to the db
	contractCallWithToken, err := ec.ContractCallWithToken2Model(event)
	if err != nil {
		return fmt.Errorf("failed to convert ContractCallEvent to RelayData: %w", err)
	}
	//2. update last checkpoint
	lastCheckpoint, err := ec.dbAdapter.GetLastEventCheckPoint(ec.evmConfig.GetId(), events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN)
	if err != nil {
		log.Debug().Str("chainId", ec.evmConfig.GetId()).
			Str("eventName", events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN).
			Msg("[EvmClient] [handleContractCallWithToken] Get event from begining")
	}
	if event.Raw.BlockNumber > lastCheckpoint.BlockNumber ||
		(event.Raw.BlockNumber == lastCheckpoint.BlockNumber && event.Raw.TxIndex > lastCheckpoint.LogIndex) {
		lastCheckpoint.BlockNumber = event.Raw.BlockNumber
		lastCheckpoint.TxHash = event.Raw.TxHash.String()
		lastCheckpoint.LogIndex = event.Raw.Index
		lastCheckpoint.EventKey = fmt.Sprintf("%s-%d-%d", event.Raw.TxHash.String(), event.Raw.BlockNumber, event.Raw.Index)
	}
	//3. store relay data to the db, update last checkpoint
	err = ec.dbAdapter.CreateContractCallWithToken(contractCallWithToken, lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to create evm contract call: %w", err)
	}
	//2. Send to the bus
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: ec.evmConfig.GetId(),
		TxHashs:   map[string]string{contractCallWithToken.TxHash: contractCallWithToken.DestinationChain},
	}
	if ec.eventBus != nil {
		ec.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[EvmClient] [handleContractCallWithToken] event bus is undefined")
	}
	return nil
}
func (ec *EvmClient) preprocessContractCallWithToken(event *contracts.IScalarGatewayContractCallWithToken) error {
	log.Info().
		Str("sender", event.Sender.Hex()).
		Str("destinationChain", event.DestinationChain).
		Str("destinationContractAddress", event.DestinationContractAddress).
		Str("payloadHash", hex.EncodeToString(event.PayloadHash[:])).
		Str("Symbol", event.Symbol).
		Uint64("Amount", event.Amount.Uint64()).
		Str("txHash", event.Raw.TxHash.String()).
		Uint("logIndex", event.Raw.Index).
		Uint("txIndex", event.Raw.TxIndex).
		Str("logData", hex.EncodeToString(event.Raw.Data)).
		Msg("[EvmClient] [preprocessContractCallWithToken] Start handle Contract call with token")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) HandleTokenSent(event *contracts.IScalarGatewayTokenSent) error {
	//0. Preprocess the event
	ec.preprocessTokenSent(event)
	//1. Convert into a RelayData instance then store to the db
	tokenSent, err := ec.TokenSentEvent2Model(event)
	if err != nil {
		log.Error().Err(err).Msg("[EvmClient] [HandleTokenSent] failed to convert TokenSentEvent to model data")
		return err
	}
	//For evm, the token sent is verified immediately by the scalarnet
	tokenSent.Status = chains.TokenSentStatusVerifying
	//2. update last checkpoint
	lastCheckpoint, err := ec.dbAdapter.GetLastEventCheckPoint(ec.evmConfig.GetId(), events.EVENT_EVM_TOKEN_SENT)
	if err != nil {
		log.Debug().Str("chainId", ec.evmConfig.GetId()).
			Str("eventName", events.EVENT_EVM_TOKEN_SENT).
			Msg("[EvmClient] [handleTokenSent] Get event from begining")
	}
	if event.Raw.BlockNumber > lastCheckpoint.BlockNumber ||
		(event.Raw.BlockNumber == lastCheckpoint.BlockNumber && event.Raw.TxIndex > lastCheckpoint.LogIndex) {
		lastCheckpoint.BlockNumber = event.Raw.BlockNumber
		lastCheckpoint.TxHash = event.Raw.TxHash.String()
		lastCheckpoint.LogIndex = event.Raw.Index
		lastCheckpoint.EventKey = fmt.Sprintf("%s-%d-%d", event.Raw.TxHash.String(), event.Raw.BlockNumber, event.Raw.Index)
	}
	//3. store relay data to the db, update last checkpoint
	err = ec.dbAdapter.SaveTokenSent(tokenSent, lastCheckpoint)
	if err != nil {
		return fmt.Errorf("failed to create evm token send: %w", err)
	}
	//2. Send to the bus
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: ec.evmConfig.GetId(),
		TxHashs:   map[string]string{tokenSent.TxHash: tokenSent.DestinationChain},
	}
	if ec.eventBus != nil {
		ec.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_TOKEN_SENT,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[EvmClient] [HandleTokenSent] event bus is undefined")
	}
	return nil
}

func (ec *EvmClient) preprocessTokenSent(event *contracts.IScalarGatewayTokenSent) error {
	log.Info().
		Str("sender", event.Sender.Hex()).
		Str("destinationChain", event.DestinationChain).
		Str("destinationAddress", event.DestinationAddress).
		Str("txHash", event.Raw.TxHash.String()).
		Str("symbol", event.Symbol).
		Uint64("amount", event.Amount.Uint64()).
		Uint("logIndex", event.Raw.Index).
		Uint("txIndex", event.Raw.TxIndex).
		Str("logData", hex.EncodeToString(event.Raw.Data)).
		Msg("[EvmClient] [preprocessTokenSent] Start handle TokenSent")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) HandleContractCallApproved(event *contracts.IScalarGatewayContractCallApproved) error {
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
	err = ec.dbAdapter.SaveSingleValue(&contractCallApproved)
	if err != nil {
		return fmt.Errorf("failed to create contract call approved: %w", err)
	}
	// Find relayData from the db by combination (contractAddress, sourceAddress, payloadHash)
	// This contract call (initiated by the user call to the source chain) is approved by EVM network
	// So anyone can execute it on the EVM by broadcast the corresponding payload to protocol's smart contract on the destination chain
	destContractAddress := strings.TrimLeft(event.ContractAddress.Hex(), "0x")
	sourceAddress := strings.TrimLeft(event.SourceAddress, "0x")
	payloadHash := strings.TrimLeft(hex.EncodeToString(event.PayloadHash[:]), "0x")
	relayDatas, err := ec.dbAdapter.FindContractCallByParams(sourceAddress, destContractAddress, payloadHash)
	if err != nil {
		log.Error().Err(err).Msg("[EvmClient] [handleContractCallApproved] find relay data")
		return err
	}
	log.Debug().Str("contractAddress", event.ContractAddress.String()).
		Str("sourceAddress", event.SourceAddress).
		Str("payloadHash", hex.EncodeToString(event.PayloadHash[:])).
		Any("relayDatas count", len(relayDatas)).
		Msg("[EvmClient] [handleContractCallApproved] query relaydata by ContractCall")
	//3. Execute payload in the found relaydatas
	executeResults, err := ec.executeDestinationCall(event, relayDatas)
	if err != nil {
		log.Warn().Err(err).Any("executeResults", executeResults).Msg("[EvmClient] [handleContractCallApproved] execute destination call")
	}
	// Done; Don't need to send to the bus
	// TODO: Do we need to update relay data atomically?
	err = ec.dbAdapter.UpdateBatchContractCallStatus(executeResults, len(executeResults))
	if err != nil {
		return fmt.Errorf("failed to update relay data status to executed: %w", err)
	}
	return nil
}
func (ec *EvmClient) executeDestinationCall(event *contracts.IScalarGatewayContractCallApproved, contractCalls []chains.ContractCall) ([]db.ContractCallExecuteResult, error) {
	executeResults := []db.ContractCallExecuteResult{}
	executed, err := ec.isContractCallExecuted(event)
	if err != nil {
		return executeResults, fmt.Errorf("[EvmClient] [executeDestinationCall] failed to check if contract call is approved: %w", err)
	}
	if executed {
		//Update the relay data status to executed
		for _, contractCall := range contractCalls {
			executeResults = append(executeResults, db.ContractCallExecuteResult{
				Status:  chains.ContractCallStatusSuccess,
				EventId: contractCall.EventID,
			})
		}
		return executeResults, fmt.Errorf("destination contract call is already executed")
	}
	if len(contractCalls) > 0 {
		for _, contractCall := range contractCalls {
			if len(contractCall.Payload) == 0 {
				continue
			}
			log.Info().Str("payload", hex.EncodeToString(contractCall.Payload)).
				Msg("[EvmClient] [executeDestinationCall]")
			receipt, err := ec.ExecuteDestinationCall(event.ContractAddress, event.CommandId, event.SourceChain, event.SourceAddress, contractCall.Payload)
			if err != nil {
				return executeResults, fmt.Errorf("execute destination call with error: %w", err)
			}

			log.Info().Any("txReceipt", receipt).Msg("[EvmClient] [executeDestinationCall]")

			if receipt.Hash() != (common.Hash{}) {
				executeResults = append(executeResults, db.ContractCallExecuteResult{
					Status:  chains.ContractCallStatusSuccess,
					EventId: contractCall.EventID,
				})
			} else {
				executeResults = append(executeResults, db.ContractCallExecuteResult{
					Status:  chains.ContractCallStatusFailed,
					EventId: contractCall.EventID,
				})
			}
		}
	}
	return executeResults, nil
}

// Check if ContractCall is already executed
func (ec *EvmClient) isContractCallExecuted(event *contracts.IScalarGatewayContractCallApproved) (bool, error) {
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

func (ec *EvmClient) preprocessContractCallApproved(event *contracts.IScalarGatewayContractCallApproved) error {
	log.Info().Any("event", event).Msgf("[EvmClient] [handleContractCallApproved]")
	//Todo: validate the event
	return nil
}

func (ec *EvmClient) HandleCommandExecuted(event *contracts.IScalarGatewayExecuted) error {
	//0. Preprocess the event
	ec.preprocessCommandExecuted(event)
	//1. Convert into a RelayData instance then store to the db
	cmdExecuted := ec.CommandExecutedEvent2Model(event)
	//Get commandId from scalarnet
	if ec.ScalarClient != nil {
		command, err := ec.ScalarClient.GetCommand(cmdExecuted.SourceChain, cmdExecuted.CommandId)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [HandleCommandExecuted] failed to get commandId from scalarnet")
		} else if command != nil {
			log.Info().Any("command", command).Msg("[EvmClient] [HandleCommandExecuted] get command from scalarnet")
			//err = ec.dbAdapter.SaveSingleValue(&cmdExecuted)
			err = ec.dbAdapter.SaveCommandExecuted(&cmdExecuted, command.Type, cmdExecuted.CommandId)
			if err != nil {
				log.Error().Err(err).Msg("[EvmClient] [HandleCommandExecuted] failed to save evm executed to the db")
				return fmt.Errorf("failed to create evm executed: %w", err)
			}
		}
	}

	//Done; Don't need to send to the bus
	return nil
}

func (ec *EvmClient) preprocessCommandExecuted(event *contracts.IScalarGatewayExecuted) error {
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
	executable, err := contracts.NewIScalarExecutable(contractAddress, ec.Client)
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
			Str("Sender", ec.auth.From.String()).
			Uint64("GasLimit", ec.auth.GasLimit).
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
