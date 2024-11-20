package evm

import (
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
		return ec.handleScalarContractCallApproved(event.MessageID, event.Data.(string))

	}
	return nil
}

func (ec *EvmClient) handleScalarContractCallApproved(messageID string, executeData string) error {
	log.Debug().Msgf("[EvmClient] [handleScalarContractCallApproved] messageID: %s, executeData: %s", messageID, executeData)
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarContractCallApproved(decodedExecuteData)
	//1. Call ScalarGateway's execute method
	//Todo add retry
	tx, err := ec.Gateway.Execute(ec.auth, []byte(executeData))
	if err != nil {
		return fmt.Errorf("failed to call execute: %w", err)
	}
	log.Info().Msgf("[EvmClient] [handleScalarContractCallApproved] tx: %v", tx)
	//2. Update status of the event
	txHash := tx.Hash().String()
	ec.dbAdapter.UpdateRelayDataStatueWithExecuteHash(messageID, relaydata.SUCCESS, &txHash)
	return nil
}

func (ec *EvmClient) observeScalarContractCallApproved(decodedExecuteData *DecodedExecuteData) error {
	log.Debug().Msgf("[EvmClient] [observeScalarContractCallApproved] decodedExecuteData: %v", decodedExecuteData)
	return nil
}
