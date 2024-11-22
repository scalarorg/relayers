package evm

import (
	"encoding/hex"
	"fmt"

	"github.com/rs/zerolog/log"
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

func (ec *EvmClient) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().Msgf("[EvmClient] [handleEventBusMessage] event: %v", event)
	switch event.EventType {
	case events.EVENT_SCALAR_CONTRACT_CALL_APPROVED:
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
	ec.observeScalarContractCallApproved(decodedExecuteData)
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
	log.Info().Any("signedTx", signedTx).Msg("[EvmClient] [handleScalarContractCallApproved]")
	//2. Update status of the event
	txHash := signedTx.Hash().String()
	err = ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	if err != nil {
		log.Error().Err(err).Str("txHash", txHash).Msg("[EvmClient] [handleScalarContractCallApproved]")
		return err
	}
	return nil
}

func (ec *EvmClient) observeScalarContractCallApproved(decodedExecuteData *DecodedExecuteData) error {
	commandIds := make([]string, len(decodedExecuteData.CommandIds))
	for i, commandId := range decodedExecuteData.CommandIds {
		commandIds[i] = hex.EncodeToString(commandId[:])
	}
	log.Debug().
		Int("inputLength", len(decodedExecuteData.Input)).
		Strs("commandIds", commandIds).
		Msg("[EvmClient] [ScalarContractCallApproved]")
	return nil
}