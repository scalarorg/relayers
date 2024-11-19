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
		return c.handleElectrsVaultTransaction(event.Data.(events.ConfirmTxsRequest))

	}
	return nil
}

func (c *Client) handleElectrsVaultTransaction(confirmRequest events.ConfirmTxsRequest) error {
	_, err := c.ConfirmTxs(context.Background(), confirmRequest.ChainName, confirmRequest.TxHashs)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleElectrsVaultTransaction] error: %v", err)
		return err
	}
	return nil
}
