package scalar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/go-utils/encode"
	utiltypes "github.com/scalarorg/bitcoin-vault/go-utils/types"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/scalarorg/relayers/pkg/utils"
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
	//Store pending commands to db
	log.Debug().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] found pending commands")
	err = c.StorePendingCommands(ctx, chain, pendingCommands)
	if err != nil {
		log.Error().Err(err).Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] failed to store pending commands")
	}
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

		// If pending commands is for pooling model then we need to create a single psbt for whole batch command
		// Otherwise, pending command contains a single command, which allready has psbt, we just need to sign it
		commandOutpoints := c.tryCreateCommandOutpoints(pendingCommands)
		if len(commandOutpoints) == 0 {
			log.Error().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] psbt already formed. Just append empty psbt to pending commands and waiting to broadcast signing request")
			emptyPsbt := covtypes.Psbt{}
			c.appendPendingPsbt(chain, []covtypes.Psbt{emptyPsbt})

		} else {
			psbtSigningRequest := types.CreatePsbtRequest{
				Outpoints: commandOutpoints,
				Params:    c.GetPsbtParams(chain),
			}
			eventEnvelope := events.EventEnvelope{
				EventType:        events.EVENT_SCALAR_CREATE_PSBT_REQUEST,
				DestinationChain: chain,
				Data:             psbtSigningRequest,
			}
			c.eventBus.BroadcastEvent(&eventEnvelope)
		}
	}
	return true
}
func (c *Client) tryCreateCommandOutpoints(pendingCommands []chainstypes.QueryCommandResponse) []types.CommandOutPoint {
	outpoints := []types.CommandOutPoint{}
	for _, cmd := range pendingCommands {
		commandOutPoint, err := TryExtractCommandOutPoint(cmd)
		if err == nil {
			outpoints = append(outpoints, commandOutPoint)
		} else {
			log.Debug().Err(err).Msg("[BtcClient] [tryCreatePsbtRequest] failed to parse command to outpoint")
		}
	}
	return outpoints
}
func TryExtractCommandOutPoint(cmd chainstypes.QueryCommandResponse) (types.CommandOutPoint, error) {
	var commandOutPoint types.CommandOutPoint
	amount, err := strconv.ParseUint(cmd.Params["amount"], 10, 64)
	if err != nil {
		return commandOutPoint, fmt.Errorf("cannot parse param %s to int value", cmd.Params["amount"])
	}
	payload, err := utils.DecodeContractCallWithTokenPayload(cmd.Payload)
	//feeOpts, rbf, pkScript, err := encode.DecodeContractCallWithTokenPayload(cmd.Payload)
	if err != nil {
		return commandOutPoint, fmt.Errorf("failed to decode contract call with token payload: %w", err)
	}
	if payload.PayloadType == encode.ContractCallWithTokenPayloadType_CustodianOnly {
		commandOutPoint = types.CommandOutPoint{
			BTCFeeOpts: payload.CustodianOnly.FeeOptions,
			RBF:        payload.CustodianOnly.RBF,
			OutPoint: utiltypes.UnstakingOutput{
				Amount:        amount,
				LockingScript: payload.CustodianOnly.RecipientChainIdentifier,
			},
		}
		return commandOutPoint, nil
	} else {
		return commandOutPoint, fmt.Errorf("unsupported payload type: %v", payload.PayloadType)
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
				c.UpdateBatchCommandSigned(ctx, destChain, res)
				eventEnvelope := events.EventEnvelope{
					EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
					DestinationChain: destChain,
					CommandIDs:       res.CommandIDs,
					Data:             res,
				}
				c.eventBus.BroadcastEvent(&eventEnvelope)
			} else if res.Status == chainstypes.BatchAborted {
				c.pendingBatchCommands.Delete(batchCommandId)
				log.Debug().
					Str("Chain", destChain).
					Str("BatchCommandId", batchCommandId).
					Msgf("[ScalarClient] [processBatchCommands] BatchCommand is aborted. Remove it from pending buffer")
			} else {
				log.Debug().
					Str("Chain", destChain).
					Str("BatchCommandId", batchCommandId).
					Msgf("[ScalarClient] [processBatchCommands] Current batch status %v: ", res.Status)
			}
		}
		time.Sleep(c.getSleepInterval())
	}
}
func (c *Client) appendPendingPsbt(chain string, psbts []covtypes.Psbt) {
	pendingPsbt, ok := c.pendingPsbtCommands.Load(chain)
	if ok {
		newPsbts := append(pendingPsbt.([]covtypes.Psbt), psbts...)
		c.pendingPsbtCommands.Store(chain, newPsbts)
	} else {
		c.pendingPsbtCommands.Store(chain, psbts)
	}
}
func (c *Client) removePendingPsbt(chain string, psbt covtypes.Psbt) {
	if value, ok := c.pendingPsbtCommands.Load(chain); ok && value != nil {
		psbts := value.([]covtypes.Psbt)
		if len(psbts) > 0 && bytes.Equal(psbts[0], psbt) {
			psbts = psbts[1:]
			c.pendingPsbtCommands.Store(chain, psbts)
		}
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
			var signRes *sdk.TxResponse
			var err error
			if len(psbt) > 0 {
				signRes, err = c.network.SignPsbtCommandsRequest(context.Background(), chain, psbt)
			} else {
				signRes, err = c.network.SignBtcCommandsRequest(context.Background(), chain)
			}
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
				c.removePendingPsbt(chain, psbt)
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
func (c *Client) StorePendingCommands(ctx context.Context, chain string, pendingCommands []chainstypes.QueryCommandResponse) error {
	return nil
}
func (c *Client) UpdateBatchCommandSigned(ctx context.Context, destChain string, batchCmds *chainstypes.BatchedCommandsResponse) error {
	commandByType := map[string][]string{}
	commands := []*scalarnet.Command{}
	for _, cmdId := range batchCmds.CommandIDs {
		cmdRes, err := c.queryClient.QueryCommand(ctx, destChain, cmdId)
		if err == nil {
			log.Debug().Str("CommandId", cmdId).Any("Command", cmdRes).Msg("[ScalarClient] [UpdateBatchCommandSigned] found command")
			cmds, ok := commandByType[cmdRes.Type]
			if !ok {
				commandByType[cmdRes.Type] = []string{cmdId}
			} else {
				commandByType[cmdRes.Type] = append(cmds, cmdId)
			}
			commands = append(commands, CommandResponse2Model(destChain, batchCmds.ID, cmdRes))
		} else {
			return fmt.Errorf("[ScalarClient] [UpdateBatchCommandSigned] failed to get command by ID: %w", err)

		}
		log.Debug().Str("CommandId", cmdId).Any("Command", cmdRes).Msg("Command response")
	}
	for cmdType, cmdIds := range commandByType {
		switch cmdType {
		case "approveContractCallWithMint":
			c.dbAdapter.UpdateContractCallWithMintsStatus(ctx, cmdIds, chains.TokenSentStatusExecuting)
		case "mintToken":
			c.dbAdapter.UpdateTokenSentsStatus(ctx, cmdIds, chains.TokenSentStatusExecuting)
		}
	}
	c.dbAdapter.SaveCommands(commands)
	return nil
}

func CommandResponse2Model(chainId string, batchCommandId string, command *chainstypes.CommandResponse) *scalarnet.Command {
	params, err := json.Marshal(command.Params)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] [CommandResponse2Model] failed to marshal command params")
	}
	return &scalarnet.Command{
		CommandID:      command.ID,
		BatchCommandID: batchCommandId,
		ChainID:        chainId,
		Params:         string(params),
		KeyID:          command.KeyID,
		CommandType:    command.Type,
		Status:         scalarnet.CommandPending,
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
