package scalar

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Todo: make it configurable
const (
	INTERVAL_DEFAULT = time.Second * 12
)

func (c *Client) getSleepInterval() time.Duration {
	interval := INTERVAL_DEFAULT
	if c.networkConfig.BlockTime > 0 {
		interval = time.Second * time.Duration(c.networkConfig.BlockTime)
	}
	return interval
}

// Periodically call to the scalar network to check if there is any pending SignCommand
// Then request signCommand request
func (c *Client) ProcessPendingCommands(ctx context.Context) {
	//Start a goroutine to process sign command txs
	go c.processSignCommandTxs(ctx)
	//Start a goroutine to process batch commands
	go c.processBatchCommands(ctx)
	//Start a goroutine to process psbt commands
	go c.processPsbtCommands(ctx)
	counter := 0
	for {
		counter += 1
		activedChains, err := c.queryClient.QueryActivedChains(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] Cannot get actived chains")
		}
		for _, chain := range activedChains {
			go func() {
				if c.checkChainPendingCommands(ctx, chain) {
					log.Debug().Str("Chain", chain).Msg("[ScalarClient] [ProcessPendingCommands] Found new pending commands")
				} else if counter >= 100 {
					log.Debug().Str("Chain", chain).Msg("[ScalarClient] [ProcessPendingCommands] No pending command found. This message is printed one of 100")
				}
			}()
		}
		if counter >= 100 {
			log.Debug().Msgf("[ScalarClient] [ProcessPendingCommands] routine is running. Sleep for %fs second then continue. This message is printed one of 100", c.getSleepInterval().Seconds())
			counter = 0
		}
		time.Sleep(c.getSleepInterval())
	}
}

// Return true if there is new pendingCommands
func (c *Client) checkChainPendingCommands(ctx context.Context, chain string) bool {
	//1. Check if there is a pending sign command request
	if _, ok := c.pendingSignCommandTxs.Load(chain); ok {
		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] There is a pending sign command request, skip send new signing request")
		return false
	}
	//1. Check if the latest batch command is in signing process
	isInSigningProcess := c.checkLatestBatchCommandIsSigning(ctx, chain)
	if isInSigningProcess {
		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] The latest batch command is in signing process, skip send new signing request")
		return false
	}
	//2. Get pending commands
	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, chain)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] failed to get pending command")
		return false
	}
	if len(pendingCommands) == 0 {
		return false
	}
	log.Debug().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] found pending commands")
	//1. Sign the commands request
	nexusChain := exported.ChainName(chain)
	if chainstypes.IsEvmChain(nexusChain) {
		//For evm chain, payload is already prepared in the scalar node, we just need to sign it before broadcast to evm node
		signRes, err := c.network.SignEvmCommandsRequest(ctx, chain)
		if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
			log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Failed to broadcast sign commands request!")
			return true
		}
		c.pendingSignCommandTxs.Store(chain, signRes.TxHash)
		log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Successfully broadcasted sign commands request with txHash: %s. Add it to pendingSignCommandTxs...", signRes.TxHash)
	} else if chainstypes.IsBitcoinChain(nexusChain) {
		//For bitcoin chain, we need to get psbt from pending commands then send sign psbt request back to the scalar node
		psbtSigningRequest := types.CreatePsbtRequest{
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
	return true
}

func (c *Client) processSignCommandTxs(ctx context.Context) {
	for {
		//Get map txHash and chain
		var hashes = map[string]string{}
		c.pendingSignCommandTxs.Range(func(key, value interface{}) bool {
			hashes[key.(string)] = value.(string)
			return true
		})

		for chain, txHash := range hashes {
			txRes, err := c.queryClient.QueryTx(ctx, txHash)
			if err != nil {
				log.Debug().Err(err).Str("TxHash", txHash).Msgf("[ScalarClient] [processSignCommandTxs]")
			} else if txRes == nil || txRes.Code != 0 || txRes.Logs == nil || len(txRes.Logs) == 0 {
				log.Debug().
					Str("Chain", chain).
					Str("TxHash", txHash).
					Msg("[ScalarClient] [processSignCommandTxs] Sign command request not found remove it from pending buffer")
				c.pendingSignCommandTxs.Delete(chain)
			} else if len(txRes.Logs) > 0 {
				c.pendingSignCommandTxs.Delete(chain)
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
		time.Sleep(c.getSleepInterval())
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
			if res.Status == chainstypes.BatchSigned {
				c.pendingBatchCommands.Delete(batchCommandId)
				log.Debug().
					Str("Chain", destChain).
					Any("BatchCommandResponse", res).
					Msg("[ScalarClient] [processBatchCommands] Found batchCommand response. Broadcast to eventBus")
				eventEnvelope := events.EventEnvelope{
					EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
					DestinationChain: destChain,
					CommandIDs:       res.CommandIDs,
					Data:             res,
				}
				c.eventBus.BroadcastEvent(&eventEnvelope)
			} else {
				log.Debug().
					Str("Chain", destChain).
					Str("BatchCommandId", batchCommandId).
					Msgf("[ScalarClient] [processBatchCommands] BatchCommand is not signed. Current status %v: ", res.Status)
			}
		}
		time.Sleep(c.getSleepInterval())
	}
}

func (c *Client) processPsbtCommands(ctx context.Context) {
	for {
		var hashes = map[string]covtypes.Psbt{}
		c.pendingPsbtCommands.Range(func(key, value interface{}) bool {
			psbts := value.([]covtypes.Psbt)
			if len(psbts) > 0 {
				hashes[key.(string)] = psbts[0]
			}
			return true
		})
		for chain, psbt := range hashes {
			//Check if the latest batch command is not in signing process
			//Query the latest batch command by set batchedCommandId to empty string
			isInSigningProcess := c.checkLatestBatchCommandIsSigning(ctx, chain)
			if isInSigningProcess {
				log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] The latest batch command is in signing process, skip send new sign command request")
				continue
			}

			log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] Send new sign psbt request")
			signRes, err := c.network.SignBtcCommandsRequest(context.Background(), chain, psbt)
			if err != nil {
				log.Error().Err(err).Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] failed to sign psbt request")
			} else if signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") {
				log.Error().Str("Chain", chain).Any("SignResponse", signRes).Msg("[ScalarClient] [processPsbtCommands] failed to sign psbt request")
			} else if signRes.TxHash == "" {
				log.Error().Str("Chain", chain).Any("SignResponse", signRes).Msg("[ScalarClient] [processPsbtCommands] failed to sign psbt request (without tx hash)")
			} else {
				log.Debug().Str("Chain", chain).Str("TxHash", signRes.TxHash).Msg("[ScalarClient] [processPsbtCommands] Successfully signed psbt request")
				//Add the tx hash to the pending buffer
				c.pendingSignCommandTxs.Store(chain, signRes.TxHash)
				//Remove the psbt from the pending buffer
				if value, ok := c.pendingPsbtCommands.Load(chain); ok && value != nil {
					psbts := value.([]covtypes.Psbt)
					if len(psbts) > 0 && bytes.Equal(psbts[0], psbt) {
						psbts = psbts[1:]
						c.pendingPsbtCommands.Store(chain, psbts)
					}
				}
			}
		}
		time.Sleep(time.Second * 3)
	}
}
func (c *Client) checkLatestBatchCommandIsSigning(ctx context.Context, chain string) bool {
	res, err := c.queryClient.QueryBatchedCommands(ctx, chain, "")
	if err != nil || res == nil {
		return false
	} else {
		return res.Status == chainstypes.BatchSigning
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
