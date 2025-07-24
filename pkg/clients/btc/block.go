package btc

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
)

func (c *BtcClient) StartNewBlockProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.btcConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[BtcClient] Starting new block processing")
	if err := c.processNextBlock(); err != nil {
		log.Error().Err(err).Msg("[BtcClient] Failed to process new block")
	}
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[BtcClient] Context cancelled, stopping new block processing")
			return
		case <-ticker.C:
			if err := c.processNextBlock(); err != nil {
				log.Error().Err(err).Msg("[BtcClient] Failed to process new block")
			}
		}
	}
}

func (c *BtcClient) processNextBlock() error {
	log.Info().Str("ChainId", c.btcConfig.GetId()).Msg("[BtcClient] [processNextBlock] Try to process next block")
	lastBlockHeader, err := c.dbAdapter.GetLastBtcBlockHeader(c.btcConfig.GetId())
	if err != nil {
		return fmt.Errorf("failed to get last btc block: %w", err)
	}
	if c.lastBlockHeader == nil || c.lastBlockHeader.Height < lastBlockHeader.Height {
		c.lastBlockHeader = lastBlockHeader
		log.Info().Int("blockHeight", lastBlockHeader.Height).Msg("[BtcClient] New block found")
		if c.eventBus != nil {
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_BTC_NEW_BLOCK,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data: &types.BtcBlockHeaderWithChain{
					ChainId:     c.btcConfig.GetId(),
					BlockHeader: lastBlockHeader,
				},
			})
		}
	} else {
		log.Info().Int("blockHeight", lastBlockHeader.Height).Msg("[BtcClient] last block header is already processed")
	}
	return nil
}
