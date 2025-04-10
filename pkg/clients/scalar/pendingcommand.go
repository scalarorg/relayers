package scalar

import (
	"context"
	"encoding/binary"
	"encoding/hex"
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
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/scalarorg/relayers/pkg/utils"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
	protocol "github.com/scalarorg/scalar-core/x/protocol/exported"
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
	// For btc command, some how we cannot get the sign command request from the scalar node
	// Process only in SignCommandSigned event
	// go c.processSignCommandTxs(ctx)
	// Start a goroutine to process batch commands
	// go c.processBatchCommands(ctx)
	//Start a goroutine to process pending commands in pooling model
	// go c.processPsbtCommands(ctx)
	// Start a goroutine to process pending commands in upc model
	// go c.processUpcCommands(ctx)
	counter := 0
	for {
		counter += 1
		activedChains, err := c.queryClient.QueryActivedChains(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] Cannot get actived chains")
		}
		for _, chain := range activedChains {
			go func() {
				if c.tryProcessPendingCommands(ctx, chain) {
					log.Debug().Str("Chain", chain).Msg("[ScalarClient] [ProcessPendingCommands] Found new pending commands")
				} else if counter >= 100 {
					counter = 0
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
func (c *Client) tryProcessPendingCommands(ctx context.Context, chain string) bool {
	//1. Check if the latest batch command is in signing process
	// deprecated
	// We add all sign command requests to the broadcaster's map buffer, this ensures that we will not send duplicate sign command requests
	// isInSigningProcess := c.isSigningProgress(ctx, chain)
	// if isInSigningProcess {
	// 	log.Debug().Str("Chain", chain).Msg("[ScalarClient] [tryProcessPendingCommands] The latest batch command is in signing process, skip send new signing request")
	// 	return false
	// }
	//2. Get pending commands
	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, chain)
	if err != nil {
		log.Error().Err(err).Msg("[ScalarClient] [ProcessSigCommand] failed to get pending command")
		return false
	}
	if len(pendingCommands) == 0 {
		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [tryProcessPendingCommands] No pending commands found")
		return false
	}
	//Store pending commands to db

	nexusChain := exported.ChainName(chain)
	if chainstypes.IsEvmChain(nexusChain) {
		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [tryProcessPendingCommands] add SignEvmCommandRequest to the broadcaster's buffer!")
		//For evm chain, payload is already prepared in the scalar node, we just need to sign it before broadcast to evm node
		err := c.broadcaster.AddSignEvmCommandsRequest(chain)
		if err != nil {
			log.Error().Err(err).Msg("[ScalarClient] [checkChainPendingCommands] failed to enqueue evm commands request")
			return true
		}
		// signRes, err := c.broadcaster.SignEvmCommandsRequest(chain)
		// if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
		// 	log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Failed to broadcast sign commands request!")
		// 	return true
		// }
		// c.pendingCommands.StoreSignRequest(chain, signRes.TxHash)
		// log.Debug().Msgf("[ScalarClient] [checkChainPendingCommands] Successfully broadcasted sign commands request with txHash: %s. Add it to pendingSignCommandTxs...", signRes.TxHash)
	} else if chainstypes.IsBitcoinChain(nexusChain) {
		return c.processBtcPendingCommands(chain, pendingCommands)
	}
	return true
}
func (c *Client) processBtcPendingCommands(chain string, pendingCommands []chainstypes.QueryCommandResponse) bool {
	// Find out the liquidity model by key_id of the first command
	liquidityModel, err := extractLiquidityModel(pendingCommands[0].KeyID)
	if err != nil {
		log.Error().Err(err).Str("Chain", chain).Str("FirstCommandKeyID", pendingCommands[0].KeyID).Msg("[ScalarClient] [processBtcPendingCommands] failed to extract liquidity model")
		return false
	}
	if liquidityModel == protocol.LIQUIDITY_MODEL_UPC {
		// If pending commands are in upc model, which allready have psbts, we just need to sign it
		log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [processBtcPendingCommands] found %d pending commands in upc model. Store them to the pending buffer", len(pendingCommands))
		// c.pendingCommands.StoreUpcPendingCommands(chain, len(pendingCommands))
		err := c.broadcaster.AddSignUpcCommandsRequest(chain)
		if err != nil {
			log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processUpcCommands] Failed to add SignUpcCommandRequest to the broadcaster's buffer")
		} else {
			log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processUpcCommands] Successfully added SignUpcCommandsRequest to the broadcaster's buffer")
		}

	} else if liquidityModel == protocol.LIQUIDITY_MODEL_POOL {
		// We do not need to create psbt for pooling model
		// Scalar core will create psbt on timeout when finish preparing phase

		// commandOutpoints := c.tryCreateCommandOutpoints(pendingCommands)
		// if len(commandOutpoints) > 0 {
		// 	log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [processBtcPendingCommands] found %d commands in pooling model. Sent create psbt request to the bitcoin client", len(commandOutpoints))
		// 	psbtSigningRequest := types.CreatePsbtRequest{
		// 		Outpoints: commandOutpoints,
		// 		Params:    c.GetPsbtParams(chain),
		// 	}
		// 	eventEnvelope := events.EventEnvelope{
		// 		EventType:        events.EVENT_SCALAR_CREATE_PSBT_REQUEST,
		// 		DestinationChain: chain,
		// 		Data:             psbtSigningRequest,
		// 	}
		// 	c.eventBus.BroadcastEvent(&eventEnvelope)
		// } else {
		// 	log.Warn().Str("Chain", chain).Msg("[ScalarClient] [checkChainPendingCommands] cannot extract outpoints from pending commands in pooling model. Something wrong!!!")
		// }
	}
	return true
}
func extractLiquidityModel(keyID string) (protocol.LiquidityModel, error) {
	parts := strings.Split(keyID, "|")
	if len(parts) != 2 {
		return protocol.LiquidityModel(0), fmt.Errorf("ParseContractCallWithTokenToBTCKeyID > invalid keyID")
	}

	modelBytes, err := hex.DecodeString(parts[0])
	if err != nil {
		return protocol.LiquidityModel(0), fmt.Errorf("ParseContractCallWithTokenToBTCKeyID > invalid model")
	}

	modelInt := binary.BigEndian.Uint32(modelBytes)
	if _, ok := protocol.LiquidityModel_name[int32(modelInt)]; !ok {
		return protocol.LiquidityModel(0), fmt.Errorf("ParseContractCallWithTokenToBTCKeyID > invalid model")
	}
	return protocol.LiquidityModel(modelInt), nil
}
func (c *Client) tryCreateCommandOutpoints(pendingCommands []chainstypes.QueryCommandResponse) []types.CommandOutPoint {
	outpoints := []types.CommandOutPoint{}
	for _, cmd := range pendingCommands {
		commandOutPoint, err := TryExtractCommandOutPoint(cmd)
		if err == nil {
			outpoints = append(outpoints, commandOutPoint)
		} else {
			log.Debug().Any("Command", cmd).Msg("[BtcClient] [tryCreatePsbtRequest] failed to parse command to outpoint")
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
			CommandID:  cmd.ID,
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
		hashes := c.pendingCommands.GetAllSignRequestTxs()

		for txHash, chains := range hashes {
			txRes, err := c.queryClient.QueryTx(ctx, txHash)
			if err != nil {
				log.Warn().Str("TxHash", txHash).Msgf("[ScalarClient] [processSignCommandTxs] Tx not found, will try later")
			} else if txRes == nil || txRes.Code != 0 || txRes.Logs == nil || len(txRes.Logs) == 0 {
				log.Debug().
					Strs("Chain", chains).
					Str("TxHash", txHash).
					Msg("[ScalarClient] [processSignCommandTxs] Sign command request not found")
				// c.pendingCommands.DeleteSignRequestTx(txHash)
			} else if len(txRes.Logs) > 0 {
				c.pendingCommands.DeleteSignRequestTx(txHash)
				for _, txLog := range txRes.Logs {
					batchedCommand := findBatchedCommand(txLog.Events)
					if batchedCommand.BatchCommandId != "" {
						log.Debug().
							Str("TxHash", txHash).
							Str("Chain", batchedCommand.Chain).
							Str("BatchCommandId", batchedCommand.BatchCommandId).
							Strs("CommandIDs", batchedCommand.CommandIDs).
							Msg("[ScalarClient] [processSignCommandTxs] BatchCommandId found, Set it  to the pending buffer for further processing.")

						c.pendingCommands.StoreBatchCommand(batchedCommand.BatchCommandId, batchedCommand.Chain)
					}
				}
			}
		}
		time.Sleep(c.getSleepInterval())
	}
}

// Find batchCommandId and corresponding chain in the events
// Return error if no batchCommandId found
func findBatchedCommand(events []sdk.StringEvent) BatchedCommand {
	batchedCommand := BatchedCommand{}
	for _, event := range events {
		if event.Type == "sign" {
			for _, attr := range event.Attributes {
				if attr.Key == "batchedCommandID" {
					batchedCommand.BatchCommandId = attr.Value
				} else if attr.Key == "chain" {
					batchedCommand.Chain = attr.Value
				} else if attr.Key == "commandIDs" {
					batchedCommand.CommandIDs = strings.Split(attr.Value, ",")
				}
			}
			break
		}
	}
	return batchedCommand
}
func (c *Client) processBatchCommands(ctx context.Context) {
	for {
		hashes := c.pendingCommands.GetAlllBatchCommands()
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
				log.Debug().
					Str("Chain", destChain).
					Str("BatchCommandId", batchCommandId).
					Msgf("[ScalarClient] [processBatchCommands] found batched command signed")
				err := c.processBatchedCommandSigned(ctx, destChain, res)
				if err != nil {
					log.Error().Err(err).
						Str("Chain", destChain).
						Str("BatchCommandId", batchCommandId).
						Msgf("[ScalarClient] [processBatchCommands] failed to process batched command signed: %v", err)
				}
				// c.pendingCommands.DeleteBatchCommand(batchCommandId)
				// log.Debug().
				// 	Str("Chain", destChain).
				// 	Any("BatchCommandResponse", res).
				// 	Msg("[ScalarClient] [processBatchCommands] Found batchCommand response. Broadcast to eventBus")
				// c.UpdateBatchCommandSigned(ctx, destChain, res)
				// eventEnvelope := events.EventEnvelope{
				// 	EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
				// 	DestinationChain: destChain,
				// 	CommandIDs:       res.CommandIDs,
				// 	Data:             res,
				// }
				// c.eventBus.BroadcastEvent(&eventEnvelope)
			} else if res.Status == chainstypes.BatchAborted {
				c.pendingCommands.DeleteBatchCommand(batchCommandId)
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

//	func (c *Client) appendPendingPsbt(chain string, psbts []covtypes.Psbt) {
//		pendingPsbt, ok := c.pendingPsbtCommands.Load(chain)
//		if ok {
//			newPsbts := append(pendingPsbt.([]covtypes.Psbt), psbts...)
//			c.pendingPsbtCommands.Store(chain, newPsbts)
//		} else {
//			c.pendingPsbtCommands.Store(chain, psbts)
//		}
//	}
//
//	func (c *Client) removePendingPsbt(chain string, psbt covtypes.Psbt) {
//		if value, ok := c.pendingPsbtCommands.Load(chain); ok && value != nil {
//			psbts := value.([]covtypes.Psbt)
//			if len(psbts) > 0 && bytes.Equal(psbts[0], psbt) {
//				psbts = psbts[1:]
//				c.pendingPsbtCommands.Store(chain, psbts)
//			}
//		}
//	}
// /*
// * Process pending commands in pooling model
// * 1. Get the first psbt from the pending buffer
// * 2. Check if the latest batch command is not in signing process
// * 3. Send new sign psbt request
// * 4. Wait for the sign event
// * 5. Store the tx hash to the pending buffer
// * 6. Remove the psbt from the pending buffer
//  */
// func (c *Client) processPsbtCommands(ctx context.Context) {
// 	for {
// 		hashes := c.pendingCommands.GetFirstPsbts()
// 		for chain, psbt := range hashes {
// 			//Check if the latest batch command is not in signing process
// 			//Query the latest batch command by set batchedCommandId to empty string
// 			// isInSigningProcess := c.hasSigningBatchCommandInNetwork(ctx, chain)
// 			// if isInSigningProcess {
// 			// 	log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] The latest batch command is in signing process, skip send new sign command request")
// 			// 	continue
// 			// }
// 			err := c.broadcaster.AddSignPsbtCommandsRequest(chain, psbt)
// 			if err != nil {
// 				log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] failed to add SignPsbtCommandRequest to the broadcaster's buffer")
// 			} else {
// 				log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processPsbtCommands] Successfully added SignPsbtCommandRequest to the broadcaster's buffer. Remove it from pending buffer")
// 				c.pendingCommands.RemovePsbt(chain, psbt)
// 			}
// 		}
// 		time.Sleep(time.Second * 3)
// 	}
// }

/*
* Process pending commands in upc model
* 1. Check if there are pending upc commands
* 2. Add the pending upc commands to the broadcaster if there is no signing batch command in the network
 */
// func (c *Client) processUpcCommands(ctx context.Context) {
// 	for {
// 		hashes := c.pendingCommands.GetUpcPendingCommands()
// 		for chain, count := range hashes {
// 			log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [processUpcCommands] found %d pending commands in upc model", count)
// 			//Check if the latest batch command is not in signing process
// 			//Query the latest batch command by set batchedCommandId to empty string
// 			// isInSigningProcess := c.hasSigningBatchCommandInNetwork(ctx, chain)
// 			// if isInSigningProcess {
// 			// 	log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processUpcCommands] The latest batch command is in signing process, skip send new sign command request")
// 			// 	continue
// 			// }

// 			added := c.broadcaster.AddSignUpcCommandsRequest(chain)
// 			if !added {
// 				log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processUpcCommands] failed to add SignUpcCommandRequest to the broadcaster's buffer")
// 			} else {
// 				log.Debug().Str("Chain", chain).Msg("[ScalarClient] [processUpcCommands] Successfully added SignUpcCommandsRequest to the broadcaster's buffer")
// 			}
// 		}
// 		time.Sleep(time.Second * 3)
// 	}
// }

// Check if there is a signing progress
// func (c *Client) isSigningProgress(ctx context.Context, chain string) bool {
// 	//1. Check if there is a pending sign command request
// 	if value, ok := c.pendingCommands.LoadSignRequest(chain); ok {
// 		log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [isSigningProgress] There is a pending sign command request with txHash: %v", value)
// 		return true
// 	}
// 	nexusChain := exported.ChainName(chain)
// 	if chainstypes.IsBitcoinChain(nexusChain) {
// 		// Check if there is a upc pending batch command
// 		hasPendingCmd := c.pendingCommands.HasPendingCommands(chain)
// 		if hasPendingCmd {
// 			log.Debug().Str("Chain", chain).Msg("[ScalarClient] [isSigningProgress] There is a pending batch command, skip send new signing request")
// 			return true
// 		}
// 	}
// 	return c.hasSigningBatchCommandInNetwork(ctx, chain)
// }

// Check if there is signing batch command in the scalar network
// func (c *Client) hasSigningBatchCommandInNetwork(ctx context.Context, chain string) bool {
// 	res, err := c.queryClient.QueryBatchedCommands(ctx, chain, "")
// 	if err != nil {
// 		log.Warn().Err(err).Str("Chain", chain).Msg("[ScalarClient] [hasSigningBatchCommandInNetwork] latest batch command not found")
// 		return false
// 	} else if res == nil {
// 		log.Debug().Str("Chain", chain).Msg("[ScalarClient] [hasSigningBatchCommandInNetwork] No batch command found")
// 		return false
// 	} else {
// 		if res.Status == chainstypes.BatchSigning {
// 			log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [hasSigningBatchCommandInNetwork] There is a signing batch in network with id: %v", res.ID)
// 			return true
// 		} else {
// 			log.Debug().Str("Chain", chain).Str("LatestBatchCommandId", res.ID).Msgf("[ScalarClient] [hasSigningBatchCommandInNetwork] Batch command status %v", res.Status)
// 			return false
// 		}
// 	}
// }

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
	}
	for cmdType, cmdIds := range commandByType {
		switch cmdType {
		case "approveContractCallWithMint":
			c.dbAdapter.UpdateContractCallWithMintsStatus(ctx, cmdIds, chains.ContractCallStatusExecuting)
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
		Status:         scalarnet.CommandStatusPending,
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
