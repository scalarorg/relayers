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
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
	"gorm.io/gorm"
)

const (
	POOL_REDEEM_TX_BATCH_SIZE = 10
)

func (c *EvmClient) StartRedeemSessionProcessing(ctx context.Context, groups []*covExported.CustodianGroup) {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Msgf("[EvmClient] Starting redeem pool session processing for %d groups", len(groups))
	wg := sync.WaitGroup{}
	for _, group := range groups {
		wg.Add(1)
		go func(groupUid chainExported.Hash) {
			defer wg.Done()
			ticker := time.NewTicker(c.pollInterval)
			defer ticker.Stop()
			log.Info().Str("ChainId", c.EvmConfig.GetId()).Msgf("[EvmClient] Starting redeem session switch phase processing for group %s", groupUid.String())
			//Get current session phase from scalar core
			for {
				select {
				case <-ctx.Done():
					log.Info().Msg("[EvmClient] Context cancelled, stopping redeem session switch phase processing")
					return
				case <-ticker.C:
					redeemSession, err := c.ScalarClient.GetRedeemSession(groupUid)
					if err != nil {
						log.Error().Err(err).Msgf("[EvmClient] Failed to get session phase for group %s", groupUid.String())
						continue
					}
					if err := c.processNextSwitchPhase(groupUid, redeemSession.Session); err != nil {
						log.Error().Err(err).Msg("[EvmClient] Failed to process pool redeem tx")
					}
				}
			}
		}(group.UID)
	}
	wg.Wait()
}
func (c *EvmClient) processNextSwitchPhase(groupUid chainExported.Hash, session *covTypes.RedeemSession) error {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Str("GroupUid", groupUid.String()).
		Any("Session", session).
		Msg("[EvmClient] Processing next switch phase")
	switchedPhaseEvent, err := c.dbAdapter.GetNextSwitchPhaseEvent(c.EvmConfig.GetId(), groupUid.String(), session.Sequence, session.CurrentPhase)
	if err == gorm.ErrRecordNotFound {
		log.Info().Msg("[EvmClient] No next switch phase event found")
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get next switch phase event: %w", err)
	}
	log.Info().Any("SwitchedPhase", switchedPhaseEvent).Msg("[EvmClient] Found next switch phase event")
	//c.HandleSwitchPhase(switchedPhaseEvent)
	if c.eventBus != nil {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_SWITCHED_PHASE,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             switchedPhaseEvent,
		})
	}
	return nil
}
func (c *EvmClient) StartPoolRedeemProcessing(ctx context.Context, groups []*covExported.CustodianGroup) {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[EvmClient] Starting redeem pool processing")
	wg := sync.WaitGroup{}
	for _, group := range groups {
		wg.Add(1)
		go func(groupUid chainExported.Hash) {
			ticker := time.NewTicker(c.pollInterval)
			defer ticker.Stop()
			defer wg.Done()
			log.Info().Str("ChainId", c.EvmConfig.GetId()).Msgf("[EvmClient] Starting redeem pool processing for group %s", groupUid.String())
			//Check if utxo snapshot is ready
			//Waiting for utxo snapshot initialied
			err := c.ScalarClient.WaitForUtxoSnapshot(groupUid)
			if err != nil {
				log.Error().Err(err).Msgf("[Relayer] [StartPoolRedeemProcessing] cannot wait for utxo snapshot for group %s", groupUid.String())
				return
			}
			log.Info().Msgf("[Relayer] [StartPoolRedeemProcessing] utxo snapshot is ready for group %s", groupUid.String())
			for {
				select {
				case <-ctx.Done():
					log.Info().Msg("[EvmClient] Context cancelled, stopping redeem processing")
					return
				case <-ticker.C:
					if err := c.processNextPoolRedeemTx(groupUid); err != nil {
						log.Error().Err(err).Msg("[EvmClient] Failed to process pool redeem tx")
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
