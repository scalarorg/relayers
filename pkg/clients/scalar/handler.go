package scalar

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
	protocol "github.com/scalarorg/scalar-core/x/protocol/exported"
)

func (c *Client) handleTokenSentEvents(ctx context.Context, events []*chainstypes.EventTokenSent) error {
	tokenSentApproveds := []*scalarnet.TokenSentApproved{}
	mapChains := make(map[string]int, 0)
	for _, event := range events {
		chain := string(event.DestinationChain)
		counter, ok := mapChains[chain]
		if !ok {
			counter = 0
		}
		mapChains[chain] = counter + 1
		model := models.EventTokenSent2Model(event)
		//Check if event_id is not empty
		if model.EventID == "" {
			log.Warn().Msgf("[ScalarClient] [handleTokenSentEvents] event_id is empty for event %v", event)
			continue
		}
		model.Status = string(chains.TokenSentStatusApproved)
		tokenSentApproveds = append(tokenSentApproveds, &model)
	}
	if len(tokenSentApproveds) == 0 {
		log.Debug().Msg("[ScalarClient] [handleTokenSentEvents] No token sent approveds to save")
		return nil
	}
	err := c.dbAdapter.SaveTokenSentApproveds(tokenSentApproveds)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [handleTokenSentEvents] failed to save token sent approveds: %v", err)
		return err
	}
	for chain, counter := range mapChains {
		log.Debug().Str("Chain", chain).
			Int("EventCounter", counter).
			Msg("[ScalarClient] [handleTokenSentEvents] enqueue PendingTransferRequest for chain")
		err := c.broadcaster.CreatePendingTransfersRequest(chain)
		if err != nil {
			log.Error().Err(err).
				Str("Chain", chain).
				Msgf("[ScalarClient] [handleTokenSentEvents] failed to enqueue PendingTransferRequest.")
		} else {
			log.Debug().Msgf("[ScalarClient] [handleTokenSentEvents] Successfully enqueue PendingTransferRequest for chain %s", chain)
			//Todo: Update token sent status to signing
			// err = c.dbAdapter.UpdateTokenSentsStatus(chain, chains.ContractCallStatusSigning)
			// if err != nil {
			// 	log.Error().Err(err).
			// 		Str("Chain", chain).
			// 		Msgf("[ScalarClient] [handleTokenSentEvents] failed to update token sent status")
			// }
		}
	}
	return nil
}

// For TokenSentEvent from scalar network, relayer need to create pending transfer request and send to the scalar network
// func (c *Client) handleTokenSentEvent(ctx context.Context, event *IBCEvent[*chainstypes.EventTokenSent]) (*db.RelaydataExecuteResult, error) {
// 	log.Debug().Interface("event", event).Msg("[ScalarClient] [handleTokenSentEvent]")
// 	//1. Get pending transfer from Scalar network
// 	destinationChain := event.Args.DestinationChain.String()
// 	txRes, err := c.network.CreatePendingTransfersRequest(ctx, destinationChain)
// 	if err != nil || txRes == nil || txRes.Code != 0 || strings.Contains(txRes.RawLog, "failed") || txRes.TxHash == "" {
// 		return nil, fmt.Errorf("[ScalarClient] [handleTokenSentEvent] failed to sign transfer request: %v, %w", txRes, err)
// 	}
// 	log.Debug().Msgf("Successfull create pending transfer request for chain %s", destinationChain)
// 	//1. Get pending command from Scalar network
// 	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, destinationChain)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get pending command: %w", err)
// 	}
// 	if len(pendingCommands) == 0 {
// 		log.Debug().Msgf("[ScalarClient] [handleTokenSentEvent] No pending command found")
// 		return nil, nil
// 	}
// 	log.Debug().Any("pendingCommands", pendingCommands).Msgf("[ScalarClient] [handleTokenSentEvent]")
// 	//2. Sign the commands request
// 	destChainName := exported.ChainName(destinationChain)
// 	// Jan 09, 2025: Apply for transfer token from EVM to EVM only
// 	// For send token from EVM to BTC, user need to call to the method gatewayContract.ContractCallWithToken
// 	if chainstypes.IsEvmChain(destChainName) {
// 		return c.signEvmCommandsRequest(ctx, string(event.Args.EventID), destinationChain)
// 	} else {
// 		log.Debug().Msgf("[ScalarClient] [handleTokenSentEvent] Not support chain: %s", destChainName)
// 		return nil, nil
// 	}
// }

func (c *Client) handleMintCommandEvents(ctx context.Context, events []*chainstypes.MintCommand) error {
	//Store the mint command to the db
	cmdModels := make([]scalarnet.MintCommand, len(events))
	for i, event := range events {
		model := CreateMintCommandFromScalarEvent(event)
		//model.TxHash = event.Hash
		cmdModels[i] = model
	}
	return c.dbAdapter.CreateOrUpdateMintCommands(cmdModels)

}

func (c *Client) handleContractCallApprovedEvents(ctx context.Context, events []*chainstypes.ContractCallApproved) error {
	entities := make([]scalarnet.ContractCallApproved, len(events))
	for i, event := range events {
		entities[i] = CreateCallContractApprovedFromScalarEvent(event)
	}
	return c.dbAdapter.CreateOrUpdateContractCallApproveds(entities)
	// updates := make([]db.RelaydataExecuteResult, 0)
	// for _, event := range events {
	// 	result, err := c.handleContractCallApprovedEvent(ctx, &event)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if result != nil {
	// 		updates = append(updates, *result)
	// 	}
	// }
	// return c.dbAdapter.UpdateBatchRelayDataStatus(updates, len(updates))
}
func (c *Client) handleRedeemTokenApprovedEvents(ctx context.Context, events []*chainstypes.EventRedeemTokenApproved) error {
	entities := make([]scalarnet.ScalarRedeemTokenApproved, len(events))
	for i, event := range events {
		entities[i] = scalarnet.ScalarRedeemTokenApproved{
			EventID:     string(event.EventID),
			SourceChain: string(event.Chain),
			//SourceTxHash:     event.Hash,
			CommandID:        hex.EncodeToString(event.CommandID[:]),
			Sender:           event.Sender,
			DestinationChain: string(event.DestinationChain),
			ContractAddress:  event.ContractAddress,
			PayloadHash:      hex.EncodeToString(event.PayloadHash[:]),
		}
	}
	return c.dbAdapter.CreateBatchValue(entities, len(entities))
}

// func (c *Client) handleContractCallApprovedEvent(ctx context.Context, event *IBCEvent[*chainstypes.ContractCallApproved]) (*db.RelaydataExecuteResult, error) {
// 	log.Debug().Interface("event", event).Msg("[ScalarClient] [preprocessContractCallApprovedEvent]")
// 	//1. Get pending command from Scalar network
// 	destinationChain := event.Args.DestinationChain.String()
// 	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, destinationChain)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get pending command: %w", err)
// 	}
// 	if len(pendingCommands) == 0 {
// 		log.Debug().Msgf("[ScalarClient] [handleContractCallApprovedEvent] No pending command found")
// 		return nil, nil
// 	}
// 	log.Debug().Any("pendingCommands", pendingCommands).Msgf("[ScalarClient] [handleContractCallApprovedEvent]")
// 	//2. Sign the commands request
// 	if chainstypes.IsEvmChain(exported.ChainName(destinationChain)) {
// 		return c.signEvmCommandsRequest(ctx, string(event.Args.EventID), destinationChain)
// 	} else {
// 		//For Vault Tx from btc, scalar client emit EventTokenSent
// 		return nil, nil
// 	}
// }

func (c *Client) handleContractCallWithMintApprovedEvents(ctx context.Context, events []*chainstypes.EventContractCallWithMintApproved) error {
	log.Debug().Msgf("[ScalarClient] [handleContractCallWithTokenApprovedEvents] update ContractCallWithToken status to Approved")
	entities := make([]scalarnet.ContractCallApprovedWithMint, len(events))
	for i, event := range events {
		entities[i] = CreateCallContractApprovedWithMintFromScalarEvent(event)
	}
	return c.dbAdapter.CreateOrUpdateContractCallApprovedWithMints(entities)
}

// func (c *Client) handleContractCallWithTokenApprovedEvent(ctx context.Context, event *IBCEvent[*chainstypes.EventContractCallWithMintApproved]) (*db.RelaydataExecuteResult, error) {
// 	log.Debug().Interface("event", event).Msg("[ScalarClient] [handleContractCallWithTokenApprovedEvent]")
// 	//1. Get pending command from Scalar network
// 	destinationChain := event.Args.DestinationChain.String()
// 	pendingCommands, err := c.queryClient.QueryPendingCommands(ctx, destinationChain)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get pending command: %w", err)
// 	}
// 	if len(pendingCommands) == 0 {
// 		log.Debug().Msgf("[ScalarClient] [handleContractCallWithTokenApprovedEvent] No pending command found")
// 		return nil, nil
// 	}
// 	log.Debug().Any("pendingCommands", pendingCommands).Msgf("[ScalarClient] [handleContractCallWithTokenApprovedEvent]")
// 	//2. Sign the commands request
// 	chainName := exported.ChainName(destinationChain)
// 	if chainstypes.IsEvmChain(chainName) {
// 		return c.signEvmCommandsRequest(ctx, string(event.Args.EventID), destinationChain)
// 	} else if chainstypes.IsBitcoinChain(chainName) {
// 		//Request btc client form psbt from pending commands then send sign psbt request back to the scalar node
// 		psbtSigningRequest := types.PsbtSigningRequest{
// 			Commands: pendingCommands,
// 			Params:   c.GetPsbtParams(destinationChain),
// 		}
// 		eventEnvelope := events.EventEnvelope{
// 			EventType:        events.EVENT_SCALAR_CREATE_PSBT_REQUEST,
// 			DestinationChain: destinationChain,
// 			MessageID:        string(event.Args.EventID),
// 			Data:             psbtSigningRequest,
// 		}
// 		c.eventBus.BroadcastEvent(&eventEnvelope)
// 		return nil, nil
// 	}
// 	return nil, nil
// }

// func (c *Client) signEvmCommandsRequest(ctx context.Context, eventId string, destinationChain string) (*db.RelaydataExecuteResult, error) {
// 	signRes, err := c.network.SignCommandsRequest(ctx, destinationChain)
// 	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
// 		return nil, fmt.Errorf("[ScalarClient] [handleContractCallApprovedEvent] failed to sign commands request: %v, %w", signRes, err)
// 	}
// 	log.Debug().Msgf("[ScalarClient] [signEvmCommandsRequest] Successfully broadcasted sign commands request with txHash: %s. Waiting for sign event...", signRes.TxHash)
// 	//Relayer is waiting for event CommandBatchSigned
// 	//3. Wait for the sign event
// 	//Todo: Check if the sign event is received
// 	batchCommandId, commandIDs := c.waitForSignCommandsEvent(ctx, signRes.TxHash)
// 	if batchCommandId == "" || commandIDs == "" {
// 		return nil, fmt.Errorf("BatchCommandId not found")
// 	}
// 	return nil, nil
// }

func (c *Client) handleCommandBatchSignedEvents(ctx context.Context, events []*chainstypes.CommandBatchSigned) error {
	for _, event := range events {
		err := c.handleCommantBatchSignedsEvent(ctx, event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) handleCommantBatchSignedsEvent(ctx context.Context, event *chainstypes.CommandBatchSigned) error {
	destinationChain := string(event.Chain)
	batchedCmds, err := c.getBatchedCommands(ctx, destinationChain, event.CommandBatchID)
	if err != nil {
		return err
	}

	if c.eventBus == nil || batchedCmds.Status != chainstypes.BatchSigned {
		return nil
	}
	log.Debug().
		Str("Chain", destinationChain).
		Str("BatchCommandId", hex.EncodeToString(event.CommandBatchID)).
		Msgf("[ScalarClient] [handleCommantBatchSignedsEvent] found batched command signed")
	return c.processBatchedCommandSigned(ctx, destinationChain, batchedCmds)
}

func (c *Client) getBatchedCommands(ctx context.Context, chain string, batchID []byte) (*chainstypes.BatchedCommandsResponse, error) {
	client, err := c.GetQueryClient().GetChainQueryServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %w", err)
	}

	res, err := client.BatchedCommands(ctx, &chainstypes.BatchedCommandsRequest{
		Chain: chain,
		Id:    hex.EncodeToString(batchID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get execute data: %w", err)
	}

	//log.Debug().Str("Chain", chain).Str("BatchID", hex.EncodeToString(batchID)).Msgf("[ScalarClient] found executeData: %s", res.ExecuteData)
	log.Debug().Str("Chain", chain).Str("BatchID", hex.EncodeToString(batchID)).Msgf("[ScalarClient] found executeData")
	return res, nil
}

func (c *Client) processBatchedCommandSigned(ctx context.Context, chain string, batchedCmds *chainstypes.BatchedCommandsResponse) error {
	chainName := exported.ChainName(chain)
	batchID := batchedCmds.ID
	//batchID := hex.EncodeToString(event.Args.CommandBatchID)
	// c.pendingCommands.BatchCommandsMutex.Lock()
	// defer c.pendingCommands.BatchCommandsMutex.Unlock()
	// _, ok := c.pendingCommands.BatchCommands.Load(batchID)
	// if ok {
	log.Debug().Str("Chain", chain).Str("BatchCommandId", batchID).Msg("[ScalarClient] [processBatchedCommandSigned] broadcast batch command.")
	eventEnvelope := events.EventEnvelope{
		EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
		DestinationChain: chain,
		CommandIDs:       batchedCmds.CommandIDs,
		Data:             batchedCmds,
	}
	c.eventBus.BroadcastEvent(&eventEnvelope)
	c.UpdateBatchCommandSigned(ctx, chain, batchedCmds)
	if chainstypes.IsBitcoinChain(chainName) {
		return c.handleBitcoinBatchCommands(chain, batchedCmds)
	}

	//c.updateCommandStatuses(ctx, chain, batchedCmds.CommandIDs)
	//c.pendingCommands.BatchCommands.Delete(batchID)
	return nil
	// } else {
	// 	log.Debug().Str("Chain", chain).Str("BatchCommandId", batchID).Msgf("[ScalarClient] [processBatchedCommandSigned] batch command not found or already processed")
	// 	return nil
	// }
}

func (c *Client) handleBitcoinBatchCommands(chain string, batchedCmds *chainstypes.BatchedCommandsResponse) error {
	liquidityModel, err := extractLiquidityModel(string(batchedCmds.KeyID))
	if err != nil {
		log.Error().Err(err).Str("Chain", chain).Msg("failed to extract liquidity model")
		return err
	}

	if liquidityModel == protocol.LIQUIDITY_MODEL_UPC {
		log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [handleBitcoinBatchCommands] liquidityModel: %s. Do nothing.", liquidityModel)
		// c.pendingCommands.DeleteUpcPendingCommands(chain)
	} else if liquidityModel == protocol.LIQUIDITY_MODEL_POOL {
		log.Debug().Str("Chain", chain).Msgf("[ScalarClient] [handleBitcoinBatchCommands] liquidityModel: %s. Delete first psbt.", liquidityModel)
		c.pendingCommands.DeleteFirstPsbt(chain)
	}
	// Send event to the event bus
	// c.eventBus.BroadcastEvent(&events.EventEnvelope{
	// 	EventType:        events.EVENT_SCALAR_BATCHCOMMAND_SIGNED,
	// 	DestinationChain: chain,
	// 	Data:             batchedCmds,
	// })
	return nil
}

func (c *Client) updateCommandStatuses(ctx context.Context, chain string, commandIDs []string) error {
	client, err := c.GetQueryClient().GetChainQueryServiceClient()
	if err != nil {
		return fmt.Errorf("failed to create service client: %w", err)
	}

	for _, cmdID := range commandIDs {
		cmdRes, err := client.Command(ctx, &chainstypes.CommandRequest{
			Chain: chain,
			ID:    cmdID,
		})
		if err != nil {
			return fmt.Errorf("failed to get command by ID: %w", err)
		}
		log.Debug().Str("CommandId", cmdID).Any("Command", cmdRes).Msg("Command response")
	}
	return nil
}

func (c *Client) findBatchCommandId(ctx context.Context, txHash string) (string, error) {
	txInfo, err := c.queryClient.QueryTx(ctx, txHash)
	if err != nil || txInfo == nil || txInfo.Code != 0 {
		return "", fmt.Errorf("failed to get tx info: %v, %w", txInfo, err)
	}
	if len(txInfo.Logs) == 0 {
		return "", fmt.Errorf("[ScalarClient] [handleContractCallApprovedEvent] no events found in the tx: %v", txInfo)
	}
	log.Debug().Msgf("[ScalarClient] [findBatchCommandId] txInfo: %v", txInfo)
	batchCommandId := findEventAttribute(txInfo.Logs[0].Events, "sign", "batchedCommandID")
	if batchCommandId == "" {
		return "", fmt.Errorf("[ScalarClient] [findBatchCommandId] failed to find batch command id")
	}
	return batchCommandId, nil
}
func (c *Client) waitForSignCommandsEvent(ctx context.Context, txHash string) (string, string) {
	var txRes *sdk.TxResponse
	var err error
	var batchCommandId string
	var commandIDs string
	//First time wait for 5 seconds
	time.Sleep(5 * time.Second)
	for {
		txRes, err = c.queryClient.QueryTx(ctx, txHash)
		if err != nil {
			log.Debug().Err(err).Str("TxHash", txHash).Msgf("[ScalarClient] [waitForSignCommandsEvent]")
			//Wait for 2 seconds before retry
			time.Sleep(2 * time.Second)
			continue
		} else if txRes == nil || txRes.Code != 0 || txRes.Logs == nil || len(txRes.Logs) == 0 {
			log.Debug().
				Str("TxHash", txHash).
				Msg("[ScalarClient] [waitForSignCommandsEvent] txResponse not found")
		} else if len(txRes.Logs) > 0 {
			batchCommandId = findEventAttribute(txRes.Logs[0].Events, "sign", "batchedCommandID")
			commandIDs = findEventAttribute(txRes.Logs[0].Events, "sign", "commandIDs")
		}
		return batchCommandId, commandIDs
	}
}
func (c *Client) waitForExecuteData(ctx context.Context, destinationChain string, batchCommandId string) (*chainstypes.BatchedCommandsResponse, error) {
	res, err := c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
	for {
		if err != nil {
			return nil, fmt.Errorf("failed to get batched commands: %w", err)
		}
		if res.Status != 3 {
			time.Sleep(3 * time.Second)
			res, err = c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
			if err != nil {
				log.Error().Err(err).
					Str("destinationChain", destinationChain).
					Str("batchCommandId", batchCommandId).
					Msg("[ScalarClient] [waitForExecuteData]")
			}
		} else {
			break
		}
	}
	return res, nil
}

func (c *Client) handleCompletedEvents(ctx context.Context, events []*chainstypes.ChainEventCompleted) error {
	mapEventIDs := make(map[string][]string)
	for _, event := range events {
		mapEventIDs[event.Type] = append(mapEventIDs[event.Type], string(event.EventID))
	}
	for eventType, eventIDs := range mapEventIDs {
		switch eventType {
		case "Event_ContractCallWithToken":
			c.handleContractCallWithTokenCompletedEvents(eventIDs)
		case "Event_TokenSent":
			c.handleTokenSentCompletedEvents(eventIDs)
			// case "Event_ContractCall":
			// 	c.handleContractCallCompletedEvents(eventIDs)
		}
	}
	return nil
}

func (c *Client) handleContractCallWithTokenCompletedEvents(eventIDs []string) error {
	c.dbAdapter.RelayerClient.Model(&scalarnet.ContractCallApprovedWithMint{}).Where("event_id IN (?)", eventIDs).Update("status", chains.ContractCallStatusSuccess)
	return nil
}

func (c *Client) handleTokenSentCompletedEvents(eventIDs []string) error {
	c.dbAdapter.RelayerClient.Model(&scalarnet.TokenSentApproved{}).Where("event_id IN (?)", eventIDs).Update("status", chains.TokenSentStatusSuccess)
	return nil
}

func (c *Client) handleSwitchPhaseStartedEvents(ctx context.Context, switchPhaseEvents []*covTypes.SwitchPhaseStarted) error {
	//Loop through all the evm chains and update the status
	for _, event := range switchPhaseEvents {
		eventEnvelope := events.EventEnvelope{
			EventType:        events.EVENT_SCALAR_SWITCH_PHASE_STARTED,
			DestinationChain: string(event.Chain),
			Data:             event,
		}
		c.eventBus.BroadcastEvent(&eventEnvelope)
	}
	return nil
}

func (c *Client) handleSwitchPhaseCompletedEvents(ctx context.Context, switchPhaseEvents []*covTypes.SwitchPhaseCompleted) error {
	//call sign pending command for each custodian pool

	return nil
}

func findEventAttribute(events []sdk.StringEvent, eventType string, attrKey string) string {
	for _, event := range events {
		if event.Type == eventType {
			for _, attr := range event.Attributes {
				if attr.Key == attrKey {
					return attr.Value
				}
			}
		}
	}
	return ""
}

// func (c *Client) handleContractCallCompletedEvent(ctx context.Context, event *IBCEvent[*chainstypes.ChainEventCompleted]) error {
// 	callConttract, err := c.dbAdapter.FindContractCallWithTokenPayloadByEventId(string(event.Args.EventID))
// 	if err != nil {
// 		return fmt.Errorf("failed to get payload: %w", err)
// 	}
// 	payload := hex.EncodeToString(callConttract.Payload)
// 	status := chains.ContractCallStatusFailed
// 	var sequence *int = nil
// 	var eventId = string(event.Args.EventID)
// 	//1. Sign and broadcast RouteMessageRequest
// 	txRes, err := c.network.SendRouteMessageRequest(ctx, eventId, payload)
// 	if err != nil {
// 		log.Error().Msgf("failed to send route message request: %+v", err)
// 	}
// 	log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] txRes: %v", txRes)
// 	if strings.Contains(txRes.RawLog, "already executed") {
// 		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Already sent an executed tx for %s. Marked it as success.", eventId)
// 		status = chains.ContractCallStatusSuccess
// 	} else {
// 		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Executed RouteMessageRequest %s.", txRes.TxHash)
// 		attrValue := findEventAttribute(txRes.Logs[0].Events, "send_packet", "packet_sequence")
// 		if attrValue != "" {
// 			value, _ := strconv.Atoi(attrValue)
// 			sequence = &value
// 		}
// 		status = chains.ContractCallStatusFailed
// 	}
// 	//2. Update the db
// 	err = c.dbAdapter.UpdateEventStatusWithPacketSequence(eventId, status, sequence)
// 	if err != nil {
// 		return fmt.Errorf("failed to update Event status: %w", err)
// 	}
// 	return nil
// }

// func (c *Client) handleTokenSentCompletedEvent(ctx context.Context, event *IBCEvent[*chainstypes.ChainEventCompleted]) error {
// 	callConttract, err := c.dbAdapter.FindContractCallWithTokenPayloadByEventId(string(event.Args.EventID))
// 	if err != nil {
// 		return fmt.Errorf("failed to get payload: %w", err)
// 	}
// 	payload := hex.EncodeToString(callConttract.Payload)
// 	if err != nil {
// 		return err
// 	}
// 	status := chains.ContractCallStatusFailed
// 	var sequence *int = nil
// 	var eventId = string(event.Args.EventID)
// 	//1. Sign and broadcast RouteMessageRequest
// 	txRes, err := c.network.SendRouteMessageRequest(ctx, eventId, payload)
// 	if err != nil {
// 		log.Error().Msgf("failed to send route message request: %+v", err)
// 	}
// 	log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] txRes: %v", txRes)
// 	if strings.Contains(txRes.RawLog, "already executed") {
// 		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Already sent an executed tx for %s. Marked it as success.", eventId)
// 		status = chains.ContractCallStatusSuccess
// 	} else {
// 		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Executed RouteMessageRequest %s.", txRes.TxHash)
// 		attrValue := findEventAttribute(txRes.Logs[0].Events, "send_packet", "packet_sequence")
// 		if attrValue != "" {
// 			value, _ := strconv.Atoi(attrValue)
// 			sequence = &value
// 		}
// 		status = chains.ContractCallStatusFailed
// 	}
// 	//2. Update the db
// 	err = c.dbAdapter.UpdateEventStatusWithPacketSequence(eventId, status, sequence)
// 	if err != nil {
// 		return fmt.Errorf("failed to update Event status: %w", err)
// 	}
// 	return nil
// }
