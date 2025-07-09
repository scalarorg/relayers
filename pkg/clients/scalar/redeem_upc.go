package scalar

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) StartUpcRedeemProcessing(ctx context.Context) {
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

func (c *Client) processNextUpcRedeemTx() error {
	var err error
	if c.processCheckPoint.LastContractCallBlock == nil {
		c.processCheckPoint.LastContractCallBlock, err = c.getLastContractCallBlock()
		if err != nil {
			return fmt.Errorf("failed to get last token sent block: %w", err)
		}
	}
	var contractCallTxs []*chains.ContractCallWithToken
	if c.processCheckPoint.LastContractCallBlock == nil {
		//No executed contract call command, get the new contract call txs
		contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(0)
		if err != nil {
			return fmt.Errorf("failed to get new contract call txs: %w", err)
		}
	} else {
		// Get uncompleted contract call txs for this block (not in command_executed)
		contractCallTxs, err = c.dbAdapter.GetUnprocessedContractCallTxsByBlock(c.processCheckPoint.LastContractCallBlock.BlockNumber)
		if err != nil {
			return fmt.Errorf("failed to get uncompleted contract call txs for block %d: %w", c.processCheckPoint.LastContractCallBlock.BlockNumber, err)
		}
		if len(contractCallTxs) == 0 {
			log.Info().Uint64("blockNumber", c.processCheckPoint.LastContractCallBlock.BlockNumber).
				Str("status", c.processCheckPoint.LastContractCallBlock.Status).
				Msg("[ScalarClient] Starting to process contract call block")
			contractCallTxs, err = c.dbAdapter.GetNextContractCallWithTokens(c.processCheckPoint.LastContractCallBlock.BlockNumber)
			if err != nil {
				return fmt.Errorf("failed to get contract call txs for block %d: %w", c.processCheckPoint.LastContractCallBlock.BlockNumber, err)
			}
		}
	}
	if len(contractCallTxs) == 0 {
		log.Info().Msg("[ScalarClient] No more contract call with token txs to process, waiting for next block")
		return nil

	} else {
		mapChainContractCallWithTokens := c.groupContractCallWithTokens(contractCallTxs)
		var overallErr error
		for chain, contractCallWithTokens := range mapChainContractCallWithTokens {
			txHashes := make(map[string]string)
			for _, tx := range contractCallWithTokens {
				txHashes[tx.TxHash] = tx.DestinationChain
			}
			err := c.requestConfirmEvmTxs(events.ConfirmTxsRequest{
				ChainName: chain,
				TxHashs:   txHashes,
			})
			if err != nil {
				log.Error().Err(err).Msgf("[ScalarClient] [processNextPoolRedeemTx] failed to request confirm evm txs")
				overallErr = err
			}
		}
		if overallErr != nil {
			log.Error().Err(overallErr).Msgf("[ScalarClient] [processNextUpcRedeemTx] failed to request confirm evm txs")
			return overallErr
		}
		c.processCheckPoint.LastContractCallBlock = &relayer.ContractCallBlock{
			BlockNumber:      contractCallTxs[0].BlockNumber,
			Chain:            contractCallTxs[0].SourceChain,
			Status:           string(relayer.BlockStatusProcessing),
			TransactionCount: len(contractCallTxs),
			ProcessedTxCount: 0,
		}
		c.dbAdapter.CreateContractCallBlock(c.processCheckPoint.LastContractCallBlock)
		log.Info().Uint64("blockNumber", c.processCheckPoint.LastContractCallBlock.BlockNumber).
			Int("uncompletedTxs", len(contractCallTxs)).
			Msg("[ScalarClient] Processing uncompleted contract call with token txs in contract call block")
	}
	return nil
}

func (c *Client) getLastContractCallBlock() (*relayer.ContractCallBlock, error) {
	contractCallBlock, err := c.dbAdapter.GetLastContractCallBlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get last contract call block: %w", err)
	}
	return contractCallBlock, nil
}

func (c *Client) groupContractCallWithTokens(contractCallWithTokens []*chains.ContractCallWithToken) map[string][]*chains.ContractCallWithToken {
	mapChainContractCallWithTokens := make(map[string][]*chains.ContractCallWithToken)
	for _, tx := range contractCallWithTokens {
		mapChainContractCallWithTokens[tx.SourceChain] = append(mapChainContractCallWithTokens[tx.SourceChain], tx)
	}
	return mapChainContractCallWithTokens
}
