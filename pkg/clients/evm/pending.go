package evm

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/clients/evm/pending"
)

func (c *EvmClient) AddPendingTx(txHash string, timestamp time.Time) {
	c.pendingTxs.AddTx(txHash, timestamp)
}

// Process pending transactions
func (c *EvmClient) WatchPendingTxs() {
	go func() {
		for {
			//Get pending txs that are older than the block time
			txs, _ := c.pendingTxs.GetTxs(c.evmConfig.BlockTime)
			for _, tx := range txs {
				log.Debug().Msgf("[EvmClient] [watchPendingTxs] processing pending tx: %s", tx.TxHash)
				c.PollTxForEvents(tx)
			}
			time.Sleep(pending.PENDING_CHECK_INTERVAL)
		}
	}()
}

func (c *EvmClient) PollTxForEvents(pendingTx pending.PendingTx) (*parser.AllEvmEvents, error) {

	txReceipt, err := c.Client.TransactionReceipt(context.Background(), common.HexToHash(pendingTx.TxHash))
	if err != nil {
		log.Error().Err(err).Str("txHash", txReceipt.TxHash.String()).Msg("[EvmClient] [pollTxForEvents] failed to get transaction receipt")
		return nil, err
	}
	events, err := parser.ParseLogs(txReceipt.Logs)
	if err != nil {
		log.Error().Err(err).Str("txHash", txReceipt.TxHash.String()).Msg("[EvmClient] [pollTxForEvents] failed to parse logs")
		return nil, err
	}

	return &events, nil
}
