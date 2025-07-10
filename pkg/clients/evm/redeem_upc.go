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

func (c *EvmClient) StartUpcRedeemProcessing(ctx context.Context) {
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

func (c *EvmClient) processNextUpcRedeemTx() error {
	var err error
	if c.lastContractCallBlock == nil {
		c.lastContractCallBlock, err = c.getLastContractCallBlock()
		if err != nil {
			return fmt.Errorf("failed to get last token sent block: %w", err)
		}
	}
	var contractCallTxs []*chains.ContractCallWithToken
	if c.lastContractCallBlock == nil {
		//No executed contract call command, get the new contract call txs
		contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(0)
		if err != nil {
			return fmt.Errorf("failed to get new contract call txs: %w", err)
		}
	} else {
		// Get uncompleted contract call txs for this block (not in command_executed)
		contractCallTxs, err = c.dbAdapter.GetUnprocessedContractCallTxsByBlock(c.lastContractCallBlock.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted contract call txs for block %d: %w", c.lastContractCallBlock.BlockNumber, err)
		}
		if len(contractCallTxs) == 0 {
			log.Info().Uint64("blockNumber", c.lastContractCallBlock.BlockNumber).
				Str("status", c.lastContractCallBlock.Status).
				Msg("[ScalarClient] Starting to process contract call block")
			contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(c.lastContractCallBlock.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get contract call txs for block %d: %w", c.lastContractCallBlock.BlockNumber, err)
			}
		}
	}
	if len(contractCallTxs) == 0 {
		log.Info().Msg("[ScalarClient] No more contract call with token txs to process, waiting for next block")
		return nil

	} else {
		mapChainContractCallWithTokens := c.groupContractCallWithTokens(contractCallTxs)
		for chain, contractCallWithTokens := range mapChainContractCallWithTokens {
			txHashes := make(map[string]string)
			for _, tx := range contractCallWithTokens {
				txHashes[tx.TxHash] = tx.DestinationChain
			}
			c.eventBus.BroadcastEvent(&events.EventEnvelope{
				EventType:        events.EVENT_EVM_CONTRACT_CALL,
				DestinationChain: events.SCALAR_NETWORK_NAME,
				Data: events.ConfirmTxsRequest{
					ChainName: chain,
					TxHashs:   txHashes,
				},
			})
		}
		c.lastContractCallBlock = &relayer.ContractCallBlock{
			BlockNumber:      contractCallTxs[0].BlockNumber,
			Chain:            contractCallTxs[0].SourceChain,
			Status:           string(relayer.BlockStatusProcessing),
			TransactionCount: len(contractCallTxs),
			ProcessedTxCount: 0,
		}
		c.dbAdapter.CreateContractCallBlock(c.lastContractCallBlock)
		log.Info().Uint64("blockNumber", c.lastContractCallBlock.BlockNumber).
			Int("uncompletedTxs", len(contractCallTxs)).
			Msg("[ScalarClient] Processing uncompleted contract call with token txs in contract call block")
	}
	return nil
}

func (c *EvmClient) getLastContractCallBlock() (*relayer.ContractCallBlock, error) {
	contractCallBlock, err := c.dbAdapter.GetLastContractCallBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get last contract call block: %w", err)
	}
	return contractCallBlock, nil
}

func (c *EvmClient) groupContractCallWithTokens(contractCallWithTokens []*chains.ContractCallWithToken) map[string][]*chains.ContractCallWithToken {
	mapChainContractCallWithTokens := make(map[string][]*chains.ContractCallWithToken)
	for _, tx := range contractCallWithTokens {
		mapChainContractCallWithTokens[tx.SourceChain] = append(mapChainContractCallWithTokens[tx.SourceChain], tx)
	}
	return mapChainContractCallWithTokens
}
