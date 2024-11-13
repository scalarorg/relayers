package evm

import (
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
)

func (ec *EvmClient) handleEventBusMessage(event *types.EventEnvelope) error {
	log.Info().Msgf("[EvmClient] [handleEventBusMessage]: %v", event)
	switch event.EventType {
	case events.EVENT_SCALAR_CONTRACT_CALL_APPROVED:
		//Broadcast from scalar.handleContractCallApprovedEvent
		return ec.handleScalarContractCallApproved(event.Data.(string))

	}
	return nil
}

func (ec *EvmClient) handleScalarContractCallApproved(executeData string) error {

	return nil
}
