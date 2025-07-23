package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
	pkgTypes "github.com/scalarorg/relayers/pkg/types"
)

// Constants are already defined in bridge.go

func (c *EvmClient) StartTransferProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[EvmClient] Starting transfer processing")

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ScalarClient] Context cancelled, stopping transfer processing")
			return
		case <-ticker.C:
			if err := c.processNextTokenSent(); err != nil {
				log.Error().Err(err).Msg("[ScalarClient] Failed to process token sent")
			}
		}
	}
}
func (c *EvmClient) getLastTokenSentBlock() (*relayer.TokenSentBlock, error) {
	tokenSentBlock, err := c.dbAdapter.GetLastTokenSentBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get next uncompleted token sent block: %w", err)
	}
	if tokenSentBlock == nil {
		log.Debug().Msg("[ScalarClient] No token sent blocks processed")
		lastTokenSents, err := c.dbAdapter.FindLastExecutedTokenSentCommand(c.EvmConfig.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get uncompleted token sents for block %d: %w", 0, err)
		}
		if len(lastTokenSents) > 0 {
			tokenSentBlock = &relayer.TokenSentBlock{
				BlockNumber:      lastTokenSents[0].BlockNumber,
				Chain:            c.EvmConfig.GetId(),
				Status:           string(relayer.BlockStatusProcessing),
				TransactionCount: len(lastTokenSents),
				ProcessedTxCount: len(lastTokenSents),
			}
			err = c.dbAdapter.CreateTokenSentBlock(tokenSentBlock)
			if err != nil {
				return nil, fmt.Errorf("failed to create token sent block: %w", err)
			}
			log.Info().Int("tokenSentsCount", len(lastTokenSents)).
				Msg("[ScalarClient] create token new sent block")
			return tokenSentBlock, nil
		}
	}
	return tokenSentBlock, nil
}
func (c *EvmClient) processNextTokenSent() error {
	// If we don't have a current processing block, get the next uncompleted one
	var tokenSents []*chains.TokenSent
	var err error
	//1. Find out the last processing token sent block
	if c.lastTokenSentBlock == nil {
		c.lastTokenSentBlock, err = c.getLastTokenSentBlock()
		if err != nil {
			return fmt.Errorf("failed to get last token sent block: %w", err)
		}
	}
	if c.lastTokenSentBlock == nil {
		//No executed token sent command, get the new token sents
		tokenSents, err = c.dbAdapter.GetNextTokenSents(0)
		if err != nil {
			return fmt.Errorf("failed to get new token sents: %w", err)
		}
	} else {
		// Get uncompleted token sents for this block (not in command_executed)
		tokenSents, err = c.dbAdapter.GetUnprocessedTokenSentsByBlock(c.lastTokenSentBlock.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted token sents for block %d: %w", c.lastTokenSentBlock.BlockNumber, err)
		}
		if len(tokenSents) == 0 {
			log.Info().Uint64("blockNumber", c.lastTokenSentBlock.BlockNumber).
				Str("status", c.lastTokenSentBlock.Status).
				Msg("[ScalarClient] Starting to process token sent block")
			tokenSents, err = c.dbAdapter.GetNextTokenSents(c.lastTokenSentBlock.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get token sents for block %d: %w", c.lastTokenSentBlock.BlockNumber, err)
			}
		}

	}

	if len(tokenSents) == 0 {
		if c.lastTokenSentBlock != nil {
			log.Info().Uint64("blockNumber", c.lastTokenSentBlock.BlockNumber).
				Msg("[ScalarClient] No more token sents to process, waiting for next token sent block")

			return nil
		} else {
			log.Info().Msg("[ScalarClient] No token sent blocks processed")
			return nil
		}
	}

	c.lastTokenSentBlock = &relayer.TokenSentBlock{
		BlockNumber:      tokenSents[0].BlockNumber,
		Chain:            tokenSents[0].SourceChain,
		Status:           string(relayer.BlockStatusProcessing),
		TransactionCount: len(tokenSents),
		ProcessedTxCount: 0,
	}
	c.dbAdapter.CreateTokenSentBlock(c.lastTokenSentBlock)

	err = c.confirmTokenSents(c.lastTokenSentBlock, tokenSents)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] Failed to confirm token sents")
		return err
	}

	log.Info().Uint64("blockNumber", c.lastTokenSentBlock.BlockNumber).
		Int("uncompletedTxs", len(tokenSents)).
		Msg("[ScalarClient] Processing uncompleted token sents in token sent block")

	return nil
}

// func (c *Client) formConfirmTokenSentsRequestV2(tokenSentBlock *relayer.TokenSentBlock, tokenSents []*chains.TokenSent) ([]*chainstypes.ConfirmSourceTxsRequestV2, error) {
// 	confirmTxs := make([]*chainsTypes.ConfirmSourceTxsRequestV2, 0)
// 	// Get block hash
// 	blockHash, err := chainsExported.HashFromHex(tokenSentBlock.BlockHash)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse block hash: %w", err)
// 	}

// 	lastConfirmTx := &chainsTypes.ConfirmSourceTxsRequestV2{
// 		Chain: nexus.ChainName(tokenSentBlock.Chain),
// 		Batch: &chainsTypes.TrustedTxsByBlock{
// 			BlockHash: blockHash,
// 			Txs:       make([]*chainsTypes.TrustedTx, 0),
// 		},
// 	}

// 	for _, tokenSent := range tokenSents {
// 		// Convert transaction hash
// 		txHash, err := chainsExported.HashFromHex(tokenSent.TxHash)
// 		if err != nil {
// 			log.Error().Err(err).Str("txHash", tokenSent.TxHash).
// 				Msg("[ScalarClient] Failed to parse transaction hash, skipping")
// 			continue
// 		}

// 		// Convert merkle proof - MerkleProof is []byte, need to decode it
// 		var merklePath []chainsExported.Hash
// 		for i := 0; i+HASH_LENGTH <= len(tokenSent.MerkleProof); i += HASH_LENGTH {
// 			merklePath = append(merklePath, chainsExported.Hash(tokenSent.MerkleProof[i:i+HASH_LENGTH]))
// 		}

// 		trustedTx := &chainstypes.TrustedTx{
// 			Hash:                     txHash,
// 			TxIndex:                  uint64(tokenSent.TxPosition),
// 			Raw:                      tokenSent.RawTx,
// 			MerklePath:               merklePath,
// 			PrevOutpointScriptPubkey: tokenSent.StakerPubkey,
// 		}
// 		if len(lastConfirmTx.Batch.Txs) >= CONFIRM_BATCH_SIZE {
// 			confirmTxs = append(confirmTxs, lastConfirmTx)
// 			lastConfirmTx = &chainsTypes.ConfirmSourceTxsRequestV2{
// 				Chain: nexus.ChainName(tokenSentBlock.Chain),
// 				Batch: &chainsTypes.TrustedTxsByBlock{
// 					BlockHash: blockHash,
// 					Txs:       make([]*chainsTypes.TrustedTx, 0),
// 				},
// 			}
// 		}
// 		lastConfirmTx.Batch.Txs = append(lastConfirmTx.Batch.Txs, trustedTx)
// 	}
// 	if len(lastConfirmTx.Batch.Txs) > 0 {
// 		confirmTxs = append(confirmTxs, lastConfirmTx)
// 	}

// 	return confirmTxs, nil
// }

// CreateTokenSentBlockFromTransactions creates a TokenSentBlock record from token sent transactions
func (c *EvmClient) CreateTokenSentBlockFromTransactions(blockNumber uint64, blockHash string, chain string, tokenSents []*chains.TokenSent) error {
	tokenSentBlock := &relayer.TokenSentBlock{
		BlockNumber:      blockNumber,
		BlockHash:        blockHash,
		Chain:            chain,
		Status:           "pending",
		TransactionCount: len(tokenSents),
		ProcessedTxCount: 0,
	}

	err := c.dbAdapter.CreateTokenSentBlock(tokenSentBlock)
	if err != nil {
		return fmt.Errorf("failed to create token sent block: %w", err)
	}

	log.Info().Uint64("blockNumber", blockNumber).
		Int("txCount", len(tokenSents)).
		Msg("[ScalarClient] Created token sent block record")

	return nil
}

// processTokenSent processes a single token sent transaction
func (c *EvmClient) confirmTokenSents(tokenSentBlock *relayer.TokenSentBlock, tokenSents []*chains.TokenSent) error {
	// Form ConfirmSourceTxsRequestV2 for single transaction
	chunks := make([]map[string]string, 0)
	for i := 0; i < len(tokenSents); i += pkgTypes.CONFIRM_BATCH_SIZE {
		end := i + pkgTypes.CONFIRM_BATCH_SIZE
		if end > len(tokenSents) {
			end = len(tokenSents)
		}
		chunk := make(map[string]string, 0)
		for _, tokenSent := range tokenSents[i:end] {
			chunk[tokenSent.TxHash] = tokenSent.DestinationChain
		}
		chunks = append(chunks, chunk)
	}
	for _, chunk := range chunks {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_EVM_TOKEN_SENT,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data: events.ConfirmTxsRequest{
				ChainName: tokenSentBlock.Chain,
				TxHashs:   chunk,
			},
		})
		// err := c.broadcaster.ConfirmEvmTxs(tokenSentBlock.Chain, chunk)
		// if err != nil {
		// 	return fmt.Errorf("failed to confirm token sent transactions: %w", err)
		// }
	}

	log.Debug().Uint64("blockNumber", tokenSentBlock.BlockNumber).
		Int("txCount", len(tokenSents)).
		Uint64("blockNumber", tokenSentBlock.BlockNumber).
		Msg("[ScalarClient] Successfully confirmed token sent transactions")

	return nil
}
