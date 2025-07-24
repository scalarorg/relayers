package evm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
)

const (
	POOL_REDEEM_TX_BATCH_SIZE = 10
)

func (c *EvmClient) StartPoolRedeemProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[EvmClient] Starting redeem pool processing")
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	groups, err := c.ScalarClient.GetCovenantGroups(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot get covenant groups")
		panic(err)
	}
	wg := sync.WaitGroup{}
	for _, group := range groups {
		wg.Add(1)
		go func(groupUid chainExported.Hash) {
			defer wg.Done()
			log.Info().Str("ChainId", c.EvmConfig.GetId()).Msgf("[ScalarClient] Starting redeem pool processing for group %s", groupUid.String())
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
	//Keep main thread alive
	wg.Wait()
}

func (c *EvmClient) processNextPoolRedeemTx(groupUid chainExported.Hash) error {
	groupUidStr := groupUid.String()
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Msgf("[ScalarClient] [processNextPoolRedeemTx] Processing redeem txs for group %s", groupUidStr)
	lastPoolRedeemTx, err := c.getLastPoolRedeemTx(groupUidStr)
	if err != nil {
		return fmt.Errorf("failed to get last pool redeem tx: %w", err)
	}
	lastRedeemBlock := uint64(0)
	lastRedeemLogIndex := uint64(0)
	if lastPoolRedeemTx != nil {
		lastRedeemBlock = lastPoolRedeemTx.BlockNumber
		lastRedeemLogIndex = lastPoolRedeemTx.LogIndex
	}
	log.Info().Uint64("LastRedeemBlock", lastRedeemBlock).
		Uint64("LastRedeemLogIndex", lastRedeemLogIndex).
		Msg("[ScalarClient] [processNextPoolRedeemTx] Get last redeem tx")
	evmRedeemTxs, err := c.dbAdapter.FindPoolRedeemTxsInLastSession(groupUidStr, lastRedeemBlock, lastRedeemLogIndex, POOL_REDEEM_TX_BATCH_SIZE)
	if err != nil {
		log.Error().Err(err).Msgf("[ScalarClient] [processNextPoolRedeemTx] failed to get redeem txs in last session")
	}
	if len(evmRedeemTxs) == 0 {
		log.Debug().Msgf("[ScalarClient] [processNextPoolRedeemTx] No redeem txs found for group %s", groupUidStr)
		return nil
	} else {
		c.storeProcessedPoolRedeemTxs(groupUidStr, evmRedeemTxs)
	}
	lastPoolRedeemTx, _ = c.getLastPoolRedeemTx(groupUidStr)
	log.Info().
		Any("LastPoolRedeemTx", lastPoolRedeemTx).
		Msgf("[ScalarClient] Processing %d redeem txs for group %s", len(evmRedeemTxs), groupUidStr)
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
		log.Info().
			Str("Chain", chain).
			Str("CustodianGroupUid", groupUidStr).
			Int("TxCount", len(redeemTxs)).
			Msg("[ScalarClient] [processNextPoolRedeemTx] Broadcasting redeem txs")
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

func (c *EvmClient) storeProcessedPoolRedeemTxs(groupUid string, evmRedeemTxs []*chains.EvmRedeemTx) {
	relayerEvmRedeemTxs := make([]*relayer.EvmRedeemTx, len(evmRedeemTxs))
	for i, tx := range evmRedeemTxs {
		relayerEvmRedeemTxs[i] = &relayer.EvmRedeemTx{
			BlockNumber: tx.BlockNumber,
			TxHash:      tx.TxHash,
			Chain:       tx.SourceChain,
			Status:      string(relayer.BlockStatusProcessing),
			LogIndex:    uint64(tx.LogIndex),
		}
	}
	if len(relayerEvmRedeemTxs) > 0 {
		if c.lastPoolRedeems == nil {
			c.lastPoolRedeems = make(map[string]*relayer.EvmRedeemTx)
		}
		lastEvmRedeemTx := relayerEvmRedeemTxs[len(relayerEvmRedeemTxs)-1]
		c.lastPoolRedeems[groupUid] = &relayer.EvmRedeemTx{
			BlockNumber: lastEvmRedeemTx.BlockNumber,
			TxHash:      lastEvmRedeemTx.TxHash,
			Chain:       lastEvmRedeemTx.Chain,
			Status:      string(relayer.BlockStatusProcessing),
			LogIndex:    uint64(lastEvmRedeemTx.LogIndex),
		}
		err := c.dbAdapter.CreateProcessedEvmRedeemTxes(relayerEvmRedeemTxs)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] Failed to store processed pool redeem txs")
		}
	}
}
