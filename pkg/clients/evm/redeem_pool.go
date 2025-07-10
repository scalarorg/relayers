package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/pkg/events"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
)

func (c *EvmClient) StartPoolRedeemProcessing(ctx context.Context) {
	log.Info().Msg("[ScalarClient] Starting redeem pool processing")
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	groups, err := c.ScalarClient.GetCovenantGroups(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot get covenant groups")
		panic(err)
	}
	for _, group := range groups {
		go func(groupUid chainExported.Hash) {
			log.Info().Msgf("[ScalarClient] Starting redeem pool processing for group %s", groupUid.String())
			for {
				select {
				case <-ctx.Done():
					log.Info().Msg("[ScalarClient] Context cancelled, stopping redeem processing")
					return
				case <-ticker.C:
					if err := c.processNextPoolRedeemTx(groupUid); err != nil {
						log.Error().Err(err).Msg("[ScalarClient] Failed to process redeem tx")
					}
				}
			}
		}(group.UID)
	}
}

func (c *EvmClient) processNextPoolRedeemTx(groupUid chainExported.Hash) error {
	groupUidStr := groupUid.String()
	evmRedeemTxs, err := c.dbAdapter.FindPoolRedeemTxsInLastSession(groupUidStr)
	if err != nil {
		return fmt.Errorf("failed to get last redeem tx block: %w", err)
	}
	if len(evmRedeemTxs) == 0 {
		log.Debug().Msgf("[ScalarClient] No redeem txs found for group %s", groupUidStr)
		return nil
	}
	log.Info().Msgf("[ScalarClient] Processing %d redeem txs for group %s", len(evmRedeemTxs), groupUidStr)
	//Log process redeem txs of the last session
	btcRedeemTxs, err := c.dbAdapter.FindLastRedeemVaultTxs(groupUidStr)
	if err != nil {
		return fmt.Errorf("failed to get last redeem vault txs: %w", err)
	}
	if len(btcRedeemTxs) > 0 {
		txHashes := []string{}
		for _, tx := range btcRedeemTxs {
			txHashes = append(txHashes, tx.TxHash)
		}
		log.Info().Str("CustodianGroupUid", groupUidStr).
			Uint64("SessionSequence", btcRedeemTxs[0].SessionSequence).
			Strs("TxHashes", txHashes).
			Msgf("[ScalarClient] redeem session already broadcasted")
		return nil
	}
	mapChainRedeemTxs := c.groupEvmRedeemTxs(evmRedeemTxs)
	for chain, redeemTxs := range mapChainRedeemTxs {
		txHashes := make(map[string]string)
		for _, tx := range redeemTxs {
			txHashes[tx.TxHash] = tx.DestinationChain
		}
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_REDEEM_TOKEN,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data: events.ConfirmTxsRequest{
				ChainName: chain,
				TxHashs:   txHashes,
			},
		})
	}
	return nil
}

func (c *EvmClient) groupEvmRedeemTxs(evmRedeemTxs []*chains.EvmRedeemTx) map[string][]*chains.EvmRedeemTx {
	mapChainRedeemTxs := make(map[string][]*chains.EvmRedeemTx)
	for _, tx := range evmRedeemTxs {
		mapChainRedeemTxs[tx.SourceChain] = append(mapChainRedeemTxs[tx.SourceChain], tx)
	}
	return mapChainRedeemTxs
}
