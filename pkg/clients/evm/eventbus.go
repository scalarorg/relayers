package evm

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

func (ec *EvmClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Debug().Str("eventType", event.EventType).
		Str("messageID", event.MessageID).
		Str("destinationChain", event.DestinationChain).
		Msg("[EvmClient] [handleEventBusMessage]")
	switch event.EventType {
	case events.EVENT_SCALAR_BATCHCOMMAND_SIGNED:
		//Emitted from scalar.handleDestCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarBatchCommandSigned(event.DestinationChain, event.Data.(string))
		if err != nil {
			log.Error().
				Err(err).
				Str("eventData", event.Data.(string)).
				Msg("[EvmClient] [handleEventBusMessage]")
			return err
		}
	case events.EVENT_SCALAR_DEST_CALL_APPROVED:
		//Emitted from scalar.handleDestCallApprovedEvent with event.Data as executeData
		err := ec.handleScalarDestCallApproved(event.MessageID, event.Data.(string))
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

func (ec *EvmClient) handleScalarBatchCommandSigned(chainId string, executeData string) error {
	log.Debug().
		Str("ChainId", chainId).
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarBatchCommandSigned]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarDestCallApproved(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.evmConfig.GetId()).
			Msg("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarBatchCommandSigned] auth is nil")
	}
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

func (ec *EvmClient) handleScalarDestCallApproved(messageID string, executeData string) error {
	log.Debug().
		Str("messageID", messageID).
		Str("executeData", executeData).
		Msg("[EvmClient] [handleScalarDestCallApproved]")
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarDestCallApproved(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	if ec.auth == nil {
		log.Error().
			Str("chainId", ec.evmConfig.GetId()).
			Msg("[EvmClient] [handleScalarDestCallApproved] auth is nil")
		return fmt.Errorf("[EvmClient] [handleScalarDestCallApproved] auth is nil")
	}
	signedTx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		log.Error().Err(err).
			Str("input", hex.EncodeToString(decodedExecuteData.Input)).
			Str("contractAddress", ec.evmConfig.Gateway).
			Str("signer", ec.auth.From.String()).
			Msg("[EvmClient] [handleScalarDestCallApproved]")
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
		Msg("[EvmClient] [handleScalarDestCallApproved]")
	txHash := signedTx.Hash().String()
	//2. Add the transaction waiting to be mined
	ec.pendingTxs.AddTx(txHash, time.Now())
	//3. Update status of the event
	err = ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	if err != nil {
		log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarDestCallApproved]")
		return err
	}
	return nil
}

func (ec *EvmClient) observeScalarDestCallApproved(decodedExecuteData *DecodedExecuteData) error {
	commandIds := make([]string, len(decodedExecuteData.CommandIds))
	for i, commandId := range decodedExecuteData.CommandIds {
		commandIds[i] = hex.EncodeToString(commandId[:])
	}
	log.Debug().
		Int("inputLength", len(decodedExecuteData.Input)).
		Strs("commandIds", commandIds).
		Msg("[EvmClient] [ScalarDestCallApproved]")
	return nil
}
