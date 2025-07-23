package btc

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *BtcClient) StartRedeemExecutedProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.btcConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[BtcClient] Starting redeem executed processing")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ScalarClient] Context cancelled, stopping bridge processing")
			return
		case <-ticker.C:
			if err := c.processNextRedeemExecuted(); err != nil {
				log.Error().Err(err).Msg("[ScalarClient] Failed to process vault block")
			}
		}
	}
}

// Find lastest redeem executed and process them
func (c *BtcClient) processNextRedeemExecuted() error {
	var btcRedeemTxs []*chains.BtcRedeemTx
	var err error
	if c.lastRedeemTx == nil {
		c.lastRedeemTx, err = c.dbAdapter.GetEvmLastRedeemTx()
		if err != nil {
			return fmt.Errorf("failed to get next uncompleted vault block: %w", err)
		}
	}

	if c.lastRedeemTx == nil {
		//No executed vault tx command, get the new vault txs
		btcRedeemTxs, err = c.dbAdapter.GetNextBtcRedeemExecuteds(0)
		if err != nil {
			return fmt.Errorf("failed to get new vault txs: %w", err)
		}
	} else {
		// Get uncompleted vault transactions for this block (not in command_executed)
		btcRedeemTxs, err = c.dbAdapter.GetBtcRedeemExecutedsByBlock(c.lastRedeemTx.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted vault transactions for block %d: %w", c.lastRedeemTx.BlockNumber, err)
		}
		if len(btcRedeemTxs) == 0 {
			log.Info().Uint64("blockNumber", c.lastRedeemTx.BlockNumber).
				Str("status", c.lastRedeemTx.Status).
				Msg("[ScalarClient] new unfinished redeem block to process. Get redeem txs from next block")
			btcRedeemTxs, err = c.dbAdapter.GetNextBtcRedeemExecuteds(c.lastRedeemTx.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get redeem transactions for block %d: %w", c.lastRedeemTx.BlockNumber, err)
			}
		}
	}

	if len(btcRedeemTxs) > 0 {
		redeemTxGroups := c.groupRedeemTxs(btcRedeemTxs)
		for _, redeemTxGroup := range redeemTxGroups {
			if redeemTxGroup.GroupUid == "" {
				//For upc redeem, we need to broadcast the redeem txs confirm request to the scalar gateway
				//Update db only
			} else {
				c.eventBus.BroadcastEvent(&events.EventEnvelope{
					EventType:        events.EVENT_BTC_REDEEM_TRANSACTION,
					DestinationChain: events.SCALAR_NETWORK_NAME,
					Data:             redeemTxGroup,
				})
				// err = c.BroadcastRedeemTxsConfirmRequest(redeemTxGroup.Chain, redeemTxGroup.GroupUid, redeemTxGroup.TxHashs)
				// if err != nil {
				// 	log.Error().Err(err).Msgf("[ScalarClient] [broadcastRedeemTxsConfirm] failed to broadcast redeem txs confirm request")
				// 	return err
				// }
			}
		}
		// c.lastRedeemTx = &relayer.EvmRedeemTx{
		// 	BlockNumber: btcRedeemTxs[0].BlockNumber,
		// 	TxHash:      btcRedeemTxs[0].TxHash,
		// 	Chain:       btcRedeemTxs[0].Chain,
		// 	Status:      string(relayer.BlockStatusProcessing),
		// }
		// //Store vault block to the relayerdb
		// c.dbAdapter.CreateRedeemBlock(c.lastRedeemTx)
	} else {
		log.Info().Msg("[ScalarClient] No redeem executed transactions to process.")
	}

	return nil
}

func (c *BtcClient) getLastRedeemBlock() (*relayer.EvmRedeemTx, error) {
	evmRedeemTx, err := c.dbAdapter.GetEvmLastRedeemTx()
	if err != nil {
		return nil, fmt.Errorf("failed to get last redeem block: %w", err)
	}
	return evmRedeemTx, nil
}

func (c *BtcClient) groupRedeemTxs(btcRedeemTxs []*chains.BtcRedeemTx) []*events.BtcRedeemTxEvents {
	redeemTxGroups := make([]*events.BtcRedeemTxEvents, 0)
	for _, tx := range btcRedeemTxs {
		var group *events.BtcRedeemTxEvents
		for _, group := range redeemTxGroups {
			if group.Chain == tx.Chain && group.GroupUid == tx.CustodianGroupUid && group.Sequence == tx.SessionSequence {
				group = group
				break
			}
		}
		if group == nil {
			group = &events.BtcRedeemTxEvents{
				Chain:       tx.Chain,
				GroupUid:    tx.CustodianGroupUid,
				Sequence:    tx.SessionSequence,
				BlockNumber: tx.BlockNumber,
				RedeemTxs:   make([]*chains.BtcRedeemTx, 0),
			}
			redeemTxGroups = append(redeemTxGroups, group)
		}
		group.RedeemTxs = append(group.RedeemTxs, tx)
	}
	return redeemTxGroups
}
