package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
)

const (
	UPC_REDEEM_TX_BATCH_SIZE = 10
)

func (c *EvmClient) StartUpcRedeemProcessing(ctx context.Context) {
	log.Info().Str("ChainId", c.EvmConfig.GetId()).
		Int("PollInterval in seconds", int(c.pollInterval.Seconds())).
		Msg("[EvmClient] Starting redeem upc processing")
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

func (c *EvmClient) processNextUpcRedeemTx() error {
	var err error
	if c.lastContractCallWithToken == nil {
		c.lastContractCallWithToken, err = c.dbAdapter.GetLastContractCallWithToken()
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] Error in Get last contract call with token")
		}
	}
	var contractCallTxs []*chains.ContractCallWithToken
	lastProcessBlock := uint64(0)
	lastProcessLogIndex := uint64(0)
	if c.lastContractCallWithToken != nil {
		lastProcessBlock = c.lastContractCallWithToken.BlockNumber
		lastProcessLogIndex = c.lastContractCallWithToken.LogIndex
	}
	log.Info().
		Uint64("lastProcessBlock", lastProcessBlock).
		Uint64("lastProcessLogIndex", lastProcessLogIndex).
		Msg("[ScalarClient] [processNextUpcRedeemTx] last process block and log index")
	contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(lastProcessBlock, lastProcessLogIndex, UPC_REDEEM_TX_BATCH_SIZE)
	if err != nil {
		return fmt.Errorf("failed to get new contract call txs: %w", err)
	}

	// if c.lastContractCallWithToken == nil {
	// 	log.Debug().Msg("[ScalarClient] No last contract call block, getting next contract call txs")
	// 	//No executed contract call command, get the new contract call txs

	// } else {
	// 	// Get uncompleted contract call txs for this block (not in command_executed)
	// 	contractCallTxs, err = c.dbAdapter.GetUnprocessedContractCallTxsByBlock(c.lastContractCallWithToken.BlockNumber)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to get uncompleted contract call txs for block %d: %w", c.lastContractCallBlock.BlockNumber, err)
	// 	}
	// 	if len(contractCallTxs) == 0 {
	// 		log.Info().Uint64("blockNumber", c.lastContractCallBlock.BlockNumber).
	// 			Str("status", c.lastContractCallBlock.Status).
	// 			Msg("[ScalarClient] Starting to process contract call block")
	// 		contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(c.lastContractCallBlock.BlockNumber)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to get contract call txs for block %d: %w", c.lastContractCallBlock.BlockNumber, err)
	// 		}
	// 	}
	// }

	if len(contractCallTxs) == 0 {
		log.Info().Msg("[ScalarClient] No more contract call with token txs to process, waiting for next block")
		return nil

	} else {
		log.Info().Int("txs", len(contractCallTxs)).
			Uint64("blockNumber", contractCallTxs[0].BlockNumber).
			Str("chain", contractCallTxs[0].SourceChain).
			Msg("[ScalarClient] Processing contract call with token txs")
		mapChainContractCallWithTokens := c.groupContractCallWithTokens(contractCallTxs)
		for chain, contractCallWithTokens := range mapChainContractCallWithTokens {
			txHashes := make(map[string]string)
			for _, tx := range contractCallWithTokens {
				txHashes[tx.TxHash] = tx.DestinationChain
			}
			log.Info().Str("chain", chain).
				Int("txs", len(txHashes)).
				Msg("[ScalarClient] Broadcasting contract call with token txs")
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_EVM_CONTRACT_CALL,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data: events.ConfirmTxsRequest{
					ChainName: chain,
					TxHashs:   txHashes,
				},
			})
		}
		c.storeProcessedContractCallWithTokens(contractCallTxs)
		// if c.lastContractCallBlock == nil || c.lastContractCallBlock.BlockNumber < contractCallTxs[0].BlockNumber {
		// 	c.lastContractCallBlock = &relayer.ContractCallBlock{
		// 		BlockNumber:      contractCallTxs[0].BlockNumber,
		// 		Chain:            contractCallTxs[0].SourceChain,
		// 		Status:           string(relayer.BlockStatusProcessing),
		// 		TransactionCount: len(contractCallTxs),
		// 		ProcessedTxCount: 0,
		// 	}
		// 	log.Info().Uint64("blockNumber", c.lastContractCallBlock.BlockNumber).
		// 		Str("chain", c.lastContractCallBlock.Chain).
		// 		Msg("[ScalarClient] Creating new contract call block")
		// 	c.dbAdapter.CreateContractCallBlock(c.lastContractCallBlock)
		// }
	}
	return nil
}

func (c *EvmClient) storeProcessedContractCallWithTokens(contractCallWithTokens []*chains.ContractCallWithToken) {
	relayerContractCallWithTokens := make([]*relayer.ContractCallWithToken, len(contractCallWithTokens))
	for i, tx := range contractCallWithTokens {
		relayerContractCallWithTokens[i] = &relayer.ContractCallWithToken{
			BlockNumber:          tx.BlockNumber,
			TxHash:               tx.TxHash,
			LogIndex:             uint64(tx.LogIndex),
			SourceChain:          tx.SourceChain,
			DestinationChain:     tx.DestinationChain,
			Status:               relayer.ContractCallStatusPending,
			TokenContractAddress: tx.TokenContractAddress,
			Symbol:               tx.Symbol,
			Amount:               tx.Amount,
		}
	}
	c.lastContractCallWithToken = relayerContractCallWithTokens[len(contractCallWithTokens)-1]
	err := c.dbAdapter.CreateContractCallWithTokens(relayerContractCallWithTokens)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] Failed to store processed contract call with tokens")
	}
}

func (c *EvmClient) groupContractCallWithTokens(contractCallWithTokens []*chains.ContractCallWithToken) map[string][]*chains.ContractCallWithToken {
	mapChainContractCallWithTokens := make(map[string][]*chains.ContractCallWithToken)
	for _, tx := range contractCallWithTokens {
		mapChainContractCallWithTokens[tx.SourceChain] = append(mapChainContractCallWithTokens[tx.SourceChain], tx)
	}
	return mapChainContractCallWithTokens
}
