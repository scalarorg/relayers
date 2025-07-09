package scalar

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func (c *Client) StartUpcRedeemProcessing(ctx context.Context) {
	log.Info().Msg("[ScalarClient] Starting redeem upc processing")
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ScalarClient] Context cancelled, stopping redeem processing")
			return
		case <-ticker.C:
			if err := c.processNextUpcRedeemTx(); err != nil {
				log.Error().Err(err).Msg("[ScalarClient] Failed to process redeem tx")
			}
		}
	}
}

func (c *Client) processNextUpcRedeemTx() error {
	// evmRedeemTxs, err := c.dbAdapter.FindUpcRedeemTxsInLastSession()
	// if err != nil {
	// 	return fmt.Errorf("failed to get redeem txs: %w", err)
	// }
	return nil
}
