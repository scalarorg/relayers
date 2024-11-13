package scalar

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) handleEventBusMessage(event *events.EventEnvelope) error {
	log.Info().Msgf("[ScalarClient] [handleEventBusMessage]: %v", event)
	switch event.EventType {
	case events.EVENT_ELECTRS_VAULT_TRANSACTION:
		//Broadcast from scalar.handleContractCallApprovedEvent
		return c.handleElectrsVaultTransaction(event.Data.(map[string][]string))

	}
	return nil
}

func (c *Client) handleElectrsVaultTransaction(chainTxHashs map[string][]string) error {
	for chainName, txHashs := range chainTxHashs {
		c.ConfirmTxs(context.Background(), chainName, txHashs)
	}
	return nil
}
