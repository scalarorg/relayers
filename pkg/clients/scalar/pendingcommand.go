package scalar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/events"
)

// Periodically call to the scalar network to check if there is any pending SignCommand
// Then request signCommand request
func (c *Client) ProcessPendingCommands(ctx context.Context) {
	interval := time.Second
	if c.networkConfig.CommandInterval > 0 {
		interval = time.Millisecond * time.Duration(c.networkConfig.CommandInterval)
	}
	counter := 0
	for {
		counter += 1
		activedChains, err := c.queryClient.QueryActivedChains(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] Cannot get actived chains")
		}
		chainsWithPendingCmds := []string{}
		for _, chain := range activedChains {
			pendingCommands, err := c.queryClient.QueryPendingCommand(ctx, chain)
			if err != nil {
				log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] failed to get pending command")
			} else if len(pendingCommands) > 0 {
				chainsWithPendingCmds = append(chainsWithPendingCmds, chain)
			} else {
				// log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [ProcessPendingCommands] No pending command found")
			}
		}
		if len(chainsWithPendingCmds) > 0 {
			log.Info().Msgf("Found chains with pending command %v", chainsWithPendingCmds)
			for _, chain := range chainsWithPendingCmds {
				go func() {
					err := c.processPendingCommandsForChain(ctx, chain)
					if err != nil {
						log.Error().Err(err).Msg("[ScalarClient] ProcessPendingCommands with error")
					}
				}()
			}
		}
		time.Sleep(interval)
		if counter >= 100 {
			log.Info().Msgf("No pending commands found. Sleep for %ds then retry.(This message is printed one of 100)", int64(interval.Seconds()))
			counter = 0
		}
	}
}

func (c *Client) processPendingCommandsForChain(ctx context.Context, destChain string) error {
	log.Debug().Str("Chain", destChain).Msg("[ScalarClient] [processPendingCommandsForChain]")
	//1. Sign the commands request
	signRes, err := c.network.SignCommandsRequest(ctx, destChain)
	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
		return fmt.Errorf("[ScalarClient] [processPendingCommandsForChain] failed to sign commands request: %v, %w", signRes, err)
	}
	log.Debug().Msgf("[ScalarClient] [processPendingCommandsForChain] Successfully broadcasted sign commands request with txHash: %s. Waiting for sign event...", signRes.TxHash)
	//3. Wait for the sign event
	//Todo: Check if the sign event is received

	batchCommandId, commandIDs := c.waitForSignCommandsEvent(ctx, signRes.TxHash)
	if batchCommandId == "" || commandIDs == "" {
		return fmt.Errorf("BatchCommandId not found")
	}
	log.Debug().Msgf("[ScalarClient] [processPendingCommandsForChain] Successfully received sign commands event with batch command id: %s", batchCommandId)
	// 2. Old version, loop for get ExecuteData from batch command id
	res, err := c.waitForExecuteData(ctx, destChain, batchCommandId)
	if err != nil {
		return fmt.Errorf("[ScalarClient] [processPendingCommandsForChain] failed to get execute data: %w", err)
	}
	log.Debug().Str("Chain", destChain).Any("BatchCommandResponse", res).Msg("[ScalarClient] [processPendingCommandsForChain] BatchCommand response")

	eventEnvelope := events.EventEnvelope{
		EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
		DestinationChain: destChain,
		Data:             res.ExecuteData,
	}
	log.Debug().Str("Chain", destChain).Str("BatchCommandId", res.ID).Any("CommandIDs", res.CommandIDs).
		Msgf("[ScalarClient] [processPendingCommandsForChain] broadcast to eventBus")
	// 3. Broadcast the execute data to the Event bus
	c.eventBus.BroadcastEvent(&eventEnvelope)

	return nil
}
