package evm

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
)

func (ec *EvmClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Debug().Str("eventType", event.EventType).
		Str("messageID", event.MessageID).
		Str("destinationChain", event.DestinationChain).
		Msg("[EvmClient] [handleEventBusMessage]")
	switch event.EventType {
	case events.EVENT_SCALAR_BATCHCOMMAND_SIGNED:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarBatchCommandSigned(event.DestinationChain, event.Data.(*chainstypes.BatchedCommandsResponse))
		if err != nil {
			log.Error().
				Err(err).
				Str("eventData", event.Data.(string)).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_TOKEN_SENT:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarTokenSent(event.Data.(string))
		if err != nil {
			log.Error().
				Err(err).
				Any("eventData", event.Data).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_DEST_CALL_APPROVED:
		//Emitted from scalar.handleContractCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarContractCallApproved(event.MessageID, event.Data.(string))
		if err != nil {
			log.Error().
				Err(err).
				Str("messageId", event.MessageID).
				Str("eventData", event.Data.(string)).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	}
	return nil
}

func (ec *EvmClient) handleScalarTokenSent(executeData string) error {
	log.Debug().
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarTokenSent]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.evmConfig.GetId()).
			Msg("[EvmClient] [handleScalarTokenSent] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarTokenSent] auth is nil")
	}
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.evmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarTokenSent]")
		return err
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarTokenSent]")
	txHash := signedTx.Hash().String()
	//2. Add the transaction waiting to be mined
	ec.pendingTxs.AddTx(txHash, time.Now())
	//3. Update status of the event
	// err = ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	// if err != nil {
	// 	log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarContractCallApproved]")
	// 	return err
	// }
	return nil
}

func (ec *EvmClient) handleScalarBatchCommandSigned(chainId string, batchedCmdRes *chainstypes.BatchedCommandsResponse) error {
	log.Debug().
		Str("ChainId", chainId).
		Str("BatchedCommandID", batchedCmdRes.ID).
		Any("CommandIDs", batchedCmdRes.CommandIDs).
		Msg("[EvmClient] [handleScalarBatchCommandSigned]")
	decodedExecuteData, err := DecodeExecuteData(batchedCmdRes.ExecuteData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.evmConfig.GetId()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
	}
	//Todo: check if token is not yet deployed on the chain
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.evmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned]")
		return err
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarBatchCommandSigned]")
	txHash := signedTx.Hash().String()
	//2. Add the transaction waiting to be mined
	ec.pendingTxs.AddTx(txHash, time.Now())
	//3. Todo: Clearify how to update status of the batchcommand
	return nil
}

// Call ScalarGateway's execute method
// executeData is raw transaction data in hex string
// It is ready to be sent directly to the network
// If call to the contract method, then we to unpack the input arguments
// After the execute method is called, the gateway contract will emit 2 events
// 1. ContractCallApproved event -> relayer will handle this event for execute protocol's contract method
// 2. Executed event -> relayer will handle this event for create a record in the db for scanner

func (ec *EvmClient) handleScalarContractCallApproved(messageID string, executeData string) error {
	log.Debug().
		Str("messageID", messageID).
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarContractCallApproved]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarExecuteData(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.evmConfig.GetId()).
			Msg("[EvmClient] [handleScalarContractCallApproved] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarContractCallApproved] auth is nil")
	}
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.evmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarContractCallApproved]")
		return err
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx()
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
	log.Info().Str("signed TxHash", signedTx.Hash().String()).
		Uint64("nonce", signedTx.Nonce()).
		Int64("chainId", signedTx.ChainId().Int64()).
		Str("signer", signedTx.To().Hex()).
		Msg("[EvmClient] [handleScalarContractCallApproved]")
	txHash := signedTx.Hash().String()
	//2. Add the transaction waiting to be mined
	ec.pendingTxs.AddTx(txHash, time.Now())
	//3. Update status of the event
	err = ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	if err != nil {
		log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarContractCallApproved]")
		return err
	}
	return nil
}

func (ec *EvmClient) observeScalarExecuteData(decodedExecuteData *DecodedExecuteData) error {
	commandIds := make([]string, len(decodedExecuteData.CommandIds))
	for i, commandId := range decodedExecuteData.CommandIds {
		commandIds[i] = hex.EncodeToString(commandId[:])
	}
	log.Debug().
		Int("inputLength", len(decodedExecuteData.Input)).
		Strs("commandIds", commandIds).
		Msg("[EvmClient] [observeScalarExecuteData]")
	return nil
}
