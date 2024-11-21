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

// Call ScalarGateway's execute method
// executeData is raw transaction data in hex string
// It is ready to be sent directly to the network
// If call to the contract method, then we to unpack the input arguments
// After the execute method is called, the gateway contract will emit 2 events
// 1. ContractCallApproved event -> relayer will handle this event for execute protocol's contract method
// 2. Executed event -> relayer will handle this event for create a record in the db for scanner

func (ec *EvmClient) handleScalarContractCallApproved(messageID string, executeData string) error {
	log.Debug().Msgf("[EvmClient] [handleScalarContractCallApproved] messageID: %s, executeData: %s", messageID, executeData)
	decodedExecuteData, err := DecodeExecuteData(executeData)
	if err != nil {
		return fmt.Errorf("failed to decode execute data: %w", err)
	}
	ec.observeScalarContractCallApproved(decodedExecuteData)
	// 1. Call ScalarGateway's execute method
	// Todo add retry
	tx, err := ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	if err != nil {
		return fmt.Errorf("failed to call execute: %w", err)
	}
	//Or send raw transaction to the network directly
	// txRaw := types.NewTx(types.DynamicFeeTx{
	// 	Data: decodedExecuteData.Input,
	// })
	// tx, err := ec.Client.SendTransaction(context.Background(), txRaw)
	// if err != nil {
	// 	return fmt.Errorf("failed to send raw transaction: %w", err)
	// }
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
