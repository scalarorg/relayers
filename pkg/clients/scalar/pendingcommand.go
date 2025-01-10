package scalar

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Periodically call to the scalar network to check if there is any pending SignCommand
// Then request signCommand request
func (c *Client) ProcessPendingCommands(ctx context.Context) {
	interval := time.Second * 5
	if c.networkConfig.CommandInterval > 0 {
		interval = time.Millisecond * time.Duration(c.networkConfig.CommandInterval)
	}
	//Start a goroutine to process sign command txs
	go c.processSignCommandTxs(ctx)
	//Start a goroutine to process batch commands
	go c.processBatchCommands(ctx)

	counter := 0
	for {
		counter += 1
		activedChains, err := c.queryClient.QueryActivedChains(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] Cannot get actived chains")
		}
		for _, chain := range activedChains {
			//Only check if there is no pending command is processing for the chain
			if value, ok := c.pendingChainCommands.Load(chain); !ok || value == nil {
				go c.checkChainPendingCommands(ctx, chain, counter)
			}
		}
		if counter >= 100 {
			counter = 0
		}
		time.Sleep(interval)

	}
}
func (c *Client) checkChainPendingCommands(ctx context.Context, chain string, counter int) {
	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, chain)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] failed to get pending command")
		return
	}
	if len(pendingCommands) > 0 {
		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] found pending commands")
		//1. Sign the commands request
		nexusChain := exported.ChainName(chain)
		if chainstypes.IsEvmChain(nexusChain) {
			//For evm chain, payload is already prepared in the scalar node, we just need to sign it before broadcast to evm node
			signRes, err := c.network.SignEvmCommandsRequest(ctx, chain)
			if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
				log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Failed to broadcast sign commands request!")
				return
			}
			c.pendingSignCommandTxs.Store(signRes.TxHash, chain)
			c.pendingChainCommands.Store(chain, 1)
			log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Successfully broadcasted sign commands request with txHash: %s. Add it to pendingSignCommandTxs...", signRes.TxHash)
		} else if chainstypes.IsBitcoinChain(nexusChain) {
			//For bitcoin chain, we need to get psbt from pending commands then send sign psbt request back to the scalar node
			psbtSigningRequest := types.PsbtSigningRequest{
				Commands: pendingCommands,
				Params:   c.GetPsbtParams(chain),
			}
			eventEnvelope := events.EventEnvelope{
				EventType:        events.EVENT_SCALAR_CREATE_PSBT_REQUEST,
				DestinationChain: chain,
				Data:             psbtSigningRequest,
			}
			c.eventBus.BroadcastEvent(&eventEnvelope)
		}

	} else {
		if counter >= 100 {
			log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [ProcessPendingCommands] No pending command found. This message is printed one of 100	")
		}
	}
}

func (c *Client) processSignCommandTxs(ctx context.Context) {
	for {
		//Get map txHash and chain
		var hashes = map[string]string{}
		c.pendingSignCommandTxs.Range(func(key, value interface{}) bool {
			hashes[key.(string)] = value.(string)
			return true
		})

		for txHash, chain := range hashes {
			txRes, err := c.queryClient.QueryTx(ctx, txHash)
			if err != nil {
				log.Debug().Err(err).Str("TxHash", txHash).Msgf("[ScalarClient] [processSignCommandTxs]")
			} else if txRes == nil || txRes.Code != 0 || txRes.Logs == nil || len(txRes.Logs) == 0 {
				log.Debug().
					Str("TxHash", txHash).
					Str("Chain", chain).
					Msg("[ScalarClient] [processSignCommandTxs] Sign command request not found")
			} else if len(txRes.Logs) > 0 {
				c.pendingChainCommands.Delete(chain)
				c.pendingSignCommandTxs.Delete(txHash)
				batchCommandId := findEventAttribute(txRes.Logs[0].Events, "sign", "batchedCommandID")
				//commandIDs = findEventAttribute(txRes.Logs[0].Events, "sign", "commandIDs")
				if batchCommandId == "" {
					log.Debug().
						Str("Chain", chain).
						Str("TxHash", txHash).
						Any("Logs", txRes.Logs).
						Msg("[ScalarClient] [processSignCommandTxs] Found transaction response but batchCommandId not found. Remove chain and txHash from pending buffer")
				} else {
					log.Debug().
						Str("TxHash", txHash).
						Str("Chain", chain).
						Str("BatchCommandId", batchCommandId).
						Msg("[ScalarClient] [processSignCommandTxs] BatchCommandId found, Set it  to the pending buffer for further processing.")
					c.pendingBatchCommands.Store(batchCommandId, chain)
				}
			}
		}
		time.Sleep(time.Second * 3)
	}
}
func (c *Client) processBatchCommands(ctx context.Context) {
	for {
		var hashes = map[string]string{}
		c.pendingBatchCommands.Range(func(key, value interface{}) bool {
			hashes[key.(string)] = value.(string)
			return true
		})
		for batchCommandId, destChain := range hashes {
			res, err := c.queryClient.QueryBatchedCommands(ctx, destChain, batchCommandId)
			if err != nil {
				log.Debug().Err(err).
					Str("Chain", destChain).
					Str("BatchCommandId", batchCommandId).
					Msgf("[ScalarClient] [processBatchCommands] batched command not found")
				continue
			}
			if res.Status == 3 {
				c.pendingBatchCommands.Delete(batchCommandId)
				log.Debug().
					Str("Chain", destChain).
					Any("BatchCommandResponse", res).
					Msg("[ScalarClient] [processBatchCommands] Found batchCommand response. Broadcast to eventBus")
				eventEnvelope := events.EventEnvelope{
					EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
					DestinationChain: destChain,
					Data:             res.ExecuteData,
				}
				c.eventBus.BroadcastEvent(&eventEnvelope)
			}
		}
		time.Sleep(time.Second * 3)
	}
}

// func (c *Client) processPendingCommandsForChain(ctx context.Context, destChain string) error {
// 	log.Debug().Str("Chain", destChain).Msg("[ScalarClient] [processPendingCommandsForChain]")
// 	//1. Sign the commands request
// 	signRes, err := c.network.SignCommandsRequest(ctx, destChain)
// 	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
// 		return fmt.Errorf("[ScalarClient] [processPendingCommandsForChain] failed to sign commands request: %v, %w", signRes, err)
// 	}
// 	log.Debug().Msgf("[ScalarClient] [processPendingCommandsForChain] Successfully broadcasted sign commands request with txHash: %s. Waiting for sign event...", signRes.TxHash)
// 	//3. Wait for the sign event
// 	//Todo: Check if the sign event is received

// 	batchCommandId, commandIDs := c.waitForSignCommandsEvent(ctx, signRes.TxHash)
// 	if batchCommandId == "" || commandIDs == "" {
// 		return fmt.Errorf("BatchCommandId not found")
// 	}
// 	log.Debug().Msgf("[ScalarClient] [processPendingCommandsForChain] Successfully received sign commands event with batch command id: %s", batchCommandId)
// 	// 2. Old version, loop for get ExecuteData from batch command id
// 	res, err := c.waitForExecuteData(ctx, destChain, batchCommandId)
// 	if err != nil {
// 		return fmt.Errorf("[ScalarClient] [processPendingCommandsForChain] failed to get execute data: %w", err)
// 	}
// 	log.Debug().Str("Chain", destChain).Any("BatchCommandResponse", res).Msg("[ScalarClient] [processPendingCommandsForChain] BatchCommand response")

// 	eventEnvelope := events.EventEnvelope{
// 		EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
// 		DestinationChain: destChain,
// 		Data:             res.ExecuteData,
// 	}
// 	log.Debug().Str("Chain", destChain).Str("BatchCommandId", res.ID).Any("CommandIDs", res.CommandIDs).
// 		Msgf("[ScalarClient] [processPendingCommandsForChain] broadcast to eventBus")
// 	// 3. Broadcast the execute data to the Event bus
// 	c.eventBus.BroadcastEvent(&eventEnvelope)

// 	return nil
// }
