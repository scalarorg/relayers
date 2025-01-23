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
			txs, _ := c.pendingTxs.GetTxs(c.EvmConfig.BlockTime)
			for _, tx := range txs {
				log.Debug().Msgf("[EvmClient] [watchPendingTxs] processing pending tx: %s", tx.TxHash)
				allEvents, err := c.PollTxForEvents(tx)
				if err != nil || allEvents == nil {
					log.Error().Err(err).Str("txHash", tx.TxHash).Msg("[EvmClient] [watchPendingTxs] failed to get transaction receipt")
					continue
				}
				if allEvents.ContractCallApproved != nil {
					log.Debug().Msg("[EvmClient] [PendingTxs] found ContractCallApproved event")
					err := c.HandleContractCallApproved(allEvents.ContractCallApproved.Args)
					if err != nil {
						log.Error().Err(err).Msg("[EvmClient] [PendingTxs] failed to handle ContractCallApproved event")
					}
				}
				//Handle Executed event after ContractCallApproved event
				if allEvents.Executed != nil {
					log.Debug().Msg("[EvmClient] [PendingTxs] found Executed event")
					err := c.HandleCommandExecuted(allEvents.Executed.Args)
					if err != nil {
						log.Error().Err(err).Msg("[EvmClient] [PendingTxs] failed to handle Executed event")
					}
				}
				//ContractCall event independent of the other events
				if allEvents.ContractCall != nil {
					log.Debug().Msg("[EvmClient] [PendingTxs] found ContractCall event")
					err := c.handleContractCall(allEvents.ContractCall.Args)
					if err != nil {
						log.Error().Err(err).Msg("[EvmClient] [PendingTxs] failed to handle ContractCall event")
					}
				}
				c.pendingTxs.RemoveTx(tx.TxHash)
			}
			time.Sleep(pending.PENDING_CHECK_INTERVAL)
		}
	}()
}

func (c *EvmClient) PollTxForEvents(pendingTx pending.PendingTx) (*parser.AllEvmEvents, error) {

	txReceipt, err := c.Client.TransactionReceipt(context.Background(), common.HexToHash(pendingTx.TxHash))
	if err != nil {
		return nil, err
	}
	events := parser.ParseLogs(txReceipt.Logs)

	return &events, nil
}
