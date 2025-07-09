package scalar

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
)

func (c *Client) StartPoolRedeemProcessing(ctx context.Context) {
	log.Info().Msg("[ScalarClient] Starting redeem pool processing")
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	groups, err := c.GetCovenantGroups(ctx)
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

// func (c *Client) getLastRedeemTxBlock() (*relayer.RedeemBlock, error) {
// 	redeemTxBlock, err := c.dbAdapter.GetLastRedeemTxBlock()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get last redeem tx block: %w", err)
// 	}
// 	if redeemTxBlock == nil {
// 		log.Debug().Msg("[ScalarClient] No token sent blocks processed")
// 		redeemTxs, err := c.dbAdapter.FindLastVaultRedeemTx(c.networkConfig.GetId())
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to get uncompleted token sents for block %d: %w", 0, err)
// 		}
// 		if len(redeemTxs) > 0 {
// 			redeemTxBlock = &relayer.RedeemBlock{
// 				BlockNumber:      redeemTxs[0].BlockNumber,
// 				Chain:            c.networkConfig.GetId(),
// 				Status:           string(relayer.BlockStatusProcessing),
// 				TransactionCount: len(redeemTxs),
// 				ProcessedTxCount: len(redeemTxs),
// 			}
// 			err = c.dbAdapter.CreateRedeemBlock(redeemTxBlock)
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to create token sent block: %w", err)
// 			}
// 			log.Info().Int("redeemTxsCount", len(redeemTxs)).
// 				Msg("[ScalarClient] create redeem new block")
// 			return redeemTxBlock, nil
// 		}
// 	}
// 	return redeemTxBlock, nil
// }

func (c *Client) processNextPoolRedeemTx(groupUid chainExported.Hash) error {
	evmRedeemTxs, err := c.dbAdapter.FindPoolRedeemTxsInLastSession(groupUid.String())
	if err != nil {
		return fmt.Errorf("failed to get last redeem tx block: %w", err)
	}
	if len(evmRedeemTxs) == 0 {
		log.Debug().Msgf("[ScalarClient] No redeem txs found for group %s", groupUid.String())
		return nil
	}
	log.Info().Msgf("[ScalarClient] Processing %d redeem txs for group %s", len(evmRedeemTxs), groupUid.String())
	//Log process redeem txs of the last session
	btcRedeemTxs, err := c.dbAdapter.FindLastRedeemVaultTxs(groupUid.String())
	if err != nil {
		return fmt.Errorf("failed to get last redeem vault txs: %w", err)
	}
	if len(btcRedeemTxs) > 0 {
		txHashes := []string{}
		for _, tx := range btcRedeemTxs {
			txHashes = append(txHashes, tx.TxHash)
		}
		log.Info().Hex("CustodianGroupUid", groupUid.Bytes()).
			Uint64("SessionSequence", btcRedeemTxs[0].SessionSequence).
			Strs("TxHashes", txHashes).
			Msgf("[ScalarClient] redeem session already broadcasted")
	}
	log.Info().Hex("CustodianGroupUid", groupUid.Bytes()).
		Uint64("SessionSequence", evmRedeemTxs[0].SessionSequence).
		Msgf("[ScalarClient] Redeem transactions are not broadcasted. Processing...")

	return nil
}
