package btc

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
	pkgTypes "github.com/scalarorg/relayers/pkg/types"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
	chainsTypes "github.com/scalarorg/scalar-core/x/chains/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"
)

func (c *BtcClient) StartBridgeProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.btcConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[BtcClient] Starting bridge processing")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ScalarClient] Context cancelled, stopping bridge processing")
			return
		case <-ticker.C:
			if err := c.processNextVaultBlock(); err != nil {
				log.Error().Err(err).Msg("[ScalarClient] Failed to process vault block")
			}
		}
	}
}
func (c *BtcClient) getLastVaultBlock() (*relayer.VaultBlock, error) {
	vaultBlock, err := c.dbAdapter.GetLastVaultBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get last vault block: %w", err)
	}
	if vaultBlock == nil {
		log.Debug().Msg("[ScalarClient] No vault blocks processed")
		vaultTxs, err := c.dbAdapter.FindLastVaultBlockByChain(c.btcConfig.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get uncompleted vault transactions for block %d: %w", 0, err)
		}
		if len(vaultTxs) > 0 {
			vaultBlock = &relayer.VaultBlock{
				BlockNumber:      vaultTxs[0].BlockNumber,
				Chain:            c.btcConfig.GetId(),
				Status:           string(relayer.BlockStatusProcessing),
				TransactionCount: len(vaultTxs),
				ProcessedTxCount: len(vaultTxs),
			}
			err = c.dbAdapter.CreateVaultBlock(vaultBlock)
			if err != nil {
				return nil, fmt.Errorf("failed to create vault block: %w", err)
			}
			log.Info().Int("vaultTxsCount", len(vaultTxs)).
				Msg("[ScalarClient] create vault new block")
			return vaultBlock, nil
		}
	}
	return vaultBlock, nil
}

func (c *BtcClient) processNextVaultBlock() error {
	// If we don't have a current processing block, get the next uncompleted one
	var vaultTxs []*chains.VaultTransaction
	var err error
	if c.lastVaultBlock == nil {
		c.lastVaultBlock, err = c.getLastVaultBlock()
		if err != nil {
			return fmt.Errorf("failed to get next uncompleted vault block: %w", err)
		}
	}
	if c.lastVaultBlock == nil {
		//No executed vault tx command, get the new vault txs
		vaultTxs, err = c.dbAdapter.GetNextVaultTransactions(0)
		if err != nil {
			return fmt.Errorf("failed to get new vault txs: %w", err)
		}
	} else {
		// Get uncompleted vault transactions for this block (not in command_executed)
		vaultTxs, err = c.dbAdapter.GetUnprocessedVaultTransactionsByBlock(c.lastVaultBlock.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted vault transactions for block %d: %w", c.lastVaultBlock.BlockNumber, err)
		}
		if len(vaultTxs) == 0 {
			log.Info().Uint64("blockNumber", c.lastVaultBlock.BlockNumber).
				Str("status", c.lastVaultBlock.Status).
				Msg("[ScalarClient] new unfinished vault block to process. Get vault txs from next block")
			vaultTxs, err = c.dbAdapter.GetNextVaultTransactions(c.lastVaultBlock.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get vault transactions for block %d: %w", c.lastVaultBlock.BlockNumber, err)
			}
		}
	}

	if len(vaultTxs) == 0 {
		if c.lastVaultBlock != nil {
			log.Info().Uint64("LastVaultBlockNumber", c.lastVaultBlock.BlockNumber).
				Msg("[ScalarClient] No more vault transactions to process, waiting for next vault block")

			return nil
		} else {
			log.Info().Msg("[ScalarClient] No vault blocks processed")
			return nil
		}
	}

	if c.lastVaultBlock == nil || c.lastVaultBlock.BlockNumber < vaultTxs[0].BlockNumber {
		c.lastVaultBlock = &relayer.VaultBlock{
			BlockNumber:      vaultTxs[0].BlockNumber,
			BlockHash:        vaultTxs[0].BlockHash,
			Chain:            vaultTxs[0].Chain,
			Status:           string(relayer.BlockStatusProcessing),
			TransactionCount: len(vaultTxs),
			ProcessedTxCount: 0,
		}
		//Store vault block to the relayerdb
		c.dbAdapter.CreateVaultBlock(c.lastVaultBlock)
	}
	err = c.confirmVaultTransactions(c.lastVaultBlock, vaultTxs)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] Failed to confirm vault transactions")
		return err
	}

	log.Info().Uint64("blockNumber", c.lastVaultBlock.BlockNumber).
		Int("uncompletedTxs", len(vaultTxs)).
		Msg("[ScalarClient] Processing uncompleted transactions in vault block")

	return nil
}

func (c *BtcClient) formConfirmSourceTxsRequestV2(vaultBlock *relayer.VaultBlock, vaultTxs []*chains.VaultTransaction) ([]*chainsTypes.ConfirmSourceTxsRequestV2, error) {
	confirmTxs := make([]*chainsTypes.ConfirmSourceTxsRequestV2, 0)
	// Get block hash
	blockHash, err := chainsExported.HashFromHex(vaultBlock.BlockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block hash: %w", err)
	}

	lastConfirmTx := &chainsTypes.ConfirmSourceTxsRequestV2{
		Chain: nexus.ChainName(vaultBlock.Chain),
		Batch: &chainsTypes.TrustedTxsByBlock{
			BlockHash: blockHash,
			Txs:       make([]*chainsTypes.TrustedTx, 0),
		},
	}

	for _, vaultTx := range vaultTxs {
		// Convert transaction hash
		txHash, err := chainsExported.HashFromHex(vaultTx.TxHash)
		if err != nil {
			log.Error().Err(err).Str("txHash", vaultTx.TxHash).
				Msg("[ScalarClient] Failed to parse transaction hash, skipping")
			continue
		}

		// Convert merkle proof - MerkleProof is []byte, need to decode it
		var merklePath []chainsExported.Hash
		for i := 0; i+pkgTypes.HASH_LENGTH <= len(vaultTx.MerkleProof); i += pkgTypes.HASH_LENGTH {
			merklePath = append(merklePath, chainsExported.Hash(vaultTx.MerkleProof[i:i+pkgTypes.HASH_LENGTH]))
		}
		stakerScriptPubkey, err := hex.DecodeString(vaultTx.StakerScriptPubkey)
		if err != nil {
			return nil, fmt.Errorf("failed to decode staker script pubkey: %w", err)
		}
		trustedTx := &chainsTypes.TrustedTx{
			Hash:                     txHash,
			TxIndex:                  uint64(vaultTx.TxPosition),
			Raw:                      vaultTx.RawTx,
			MerklePath:               merklePath,
			PrevOutpointScriptPubkey: stakerScriptPubkey,
		}
		if len(lastConfirmTx.Batch.Txs) >= pkgTypes.CONFIRM_BATCH_SIZE {
			confirmTxs = append(confirmTxs, lastConfirmTx)
			lastConfirmTx = &chainsTypes.ConfirmSourceTxsRequestV2{
				Chain: nexus.ChainName(vaultBlock.Chain),
				Batch: &chainsTypes.TrustedTxsByBlock{
					BlockHash: blockHash,
					Txs:       make([]*chainsTypes.TrustedTx, 0),
				},
			}
		}
		lastConfirmTx.Batch.Txs = append(lastConfirmTx.Batch.Txs, trustedTx)
	}
	if len(lastConfirmTx.Batch.Txs) > 0 {
		confirmTxs = append(confirmTxs, lastConfirmTx)
	}

	return confirmTxs, nil
}

// HandleVaultBlockBroadcastResponse handles the response from broadcaster
// func (c *Client) HandleVaultBlockBroadcastResponse(blockNumber uint64, broadcastTxHash string, success bool) error {
// 	if success {
// 		// Increment processed transaction count
// 		err := c.dbAdapter.IncrementProcessedTxCount(blockNumber)
// 		if err != nil {
// 			return fmt.Errorf("failed to increment processed transaction count: %w", err)
// 		}

// 		// Check if block is fully processed
// 		isFullyProcessed, err := c.dbAdapter.IsBlockFullyProcessed(blockNumber)
// 		if err != nil {
// 			return fmt.Errorf("failed to check if block is fully processed: %w", err)
// 		}

// 		if isFullyProcessed {
// 			// Mark block as completed
// 			err = c.dbAdapter.MarkBlockAsCompleted(blockNumber, broadcastTxHash)
// 			if err != nil {
// 				return fmt.Errorf("failed to mark block as completed: %w", err)
// 			}
// 		}

// 		log.Info().Uint64("blockNumber", blockNumber).
// 			Str("broadcastTxHash", broadcastTxHash).
// 			Msg("[ScalarClient] Vault transaction broadcast successful")
// 	} else {
// 		// Mark block as failed
// 		err := c.dbAdapter.UpdateVaultBlockStatus(blockNumber, "failed", broadcastTxHash)
// 		if err != nil {
// 			return fmt.Errorf("failed to update vault block status to failed: %w", err)
// 		}

// 		log.Error().Uint64("blockNumber", blockNumber).
// 			Str("broadcastTxHash", broadcastTxHash).
// 			Msg("[ScalarClient] Vault transaction broadcast failed")
// 	}

// 	return nil
// }

// CreateVaultBlockFromTransactions creates a VaultBlock record from vault transactions
func (c *BtcClient) CreateVaultBlockFromTransactions(blockNumber uint64, blockHash string, chain string, vaultTxs []*chains.VaultTransaction) error {
	vaultBlock := &relayer.VaultBlock{
		BlockNumber:      blockNumber,
		BlockHash:        blockHash,
		Chain:            chain,
		Status:           "pending",
		TransactionCount: len(vaultTxs),
		ProcessedTxCount: 0,
	}

	err := c.dbAdapter.CreateVaultBlock(vaultBlock)
	if err != nil {
		return fmt.Errorf("failed to create vault block: %w", err)
	}

	log.Info().Uint64("blockNumber", blockNumber).
		Int("txCount", len(vaultTxs)).
		Msg("[ScalarClient] Created vault block record")

	return nil
}

// processVaultTransaction processes a single vault transaction
func (c *BtcClient) confirmVaultTransactions(vaultBlock *relayer.VaultBlock, vaultTxs []*chains.VaultTransaction) error {
	// Form ConfirmSourceTxsRequestV2 for single transaction
	confirmRequests, err := c.formConfirmSourceTxsRequestV2(vaultBlock, vaultTxs)
	if err != nil {
		return fmt.Errorf("failed to form confirm request for vaultBlock %+v: %w", vaultBlock, err)
	}

	// Send to broadcaster
	for _, confirmRequest := range confirmRequests {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_BTC_VAULT_BLOCK,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			MessageID:        uuid.New().String(),
			Data:             confirmRequest,
		})
		// err = c.broadcaster.ConfirmBtcVaultBlock(confirmRequest)
		// if err != nil {
		// 	return fmt.Errorf("failed to send confirm request to broadcaster for block %d with hash %s: %w", vaultBlock.BlockNumber, vaultBlock.BlockHash, err)
		// }
	}
	log.Debug().Uint64("blockNumber", vaultBlock.BlockNumber).
		Int("txCount", len(vaultTxs)).
		Uint64("blockNumber", vaultBlock.BlockNumber).
		Str("blockHash", vaultBlock.BlockHash).
		Msg("[ScalarClient] Successfully confirmed vault transactions")

	return nil
}
