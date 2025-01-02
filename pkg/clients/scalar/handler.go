package scalar

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

func (c *Client) handleTokenSentEvents(ctx context.Context, events []IBCEvent[*types.EventTokenSent]) error {
	updates := make([]db.RelaydataExecuteResult, 0)
	// for _, event := range events {
	// 	result, err := c.handleTokenSentEvent(ctx, &event)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if result != nil {
	// 		updates = append(updates, *result)
	// 	}
	// }
	return c.dbAdapter.UpdateBatchRelayDataStatus(updates, len(updates))
}

// func (c *Client) handleTokenSentEvent(ctx context.Context, event *IBCEvent[types.EventTokenSent]) (*db.RelaydataExecuteResult, error) {
// 	err := c.preprocessTokenSentEvent(event)
// 	if err != nil {
// 		return nil, err
// 	}
// 	//1. Get pending command from Scalar network
// 	destinationChain := event.Args.DestinationChain.String()
// 	pendingCommands, err := c.queryClient.QueryPendingCommand(ctx, destinationChain)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get pending command: %w", err)
// 	}
// 	if len(pendingCommands) == 0 {
// 		log.Debug().Msgf("[ScalarClient] [handleTokenSentEvent] No pending command found")
// 		return nil, nil
// 	}
// 	log.Debug().Any("pendingCommands", pendingCommands).Msgf("[ScalarClient] [handleTokenSentEvent]")
// 	//2. Sign the commands request
// 	signRes, err := c.network.SignCommandsRequest(ctx, destinationChain)
// 	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
// 		return nil, fmt.Errorf("[ScalarClient] [handleTokenSentEvent] failed to sign commands request: %v, %w", signRes, err)
// 	}
// 	log.Debug().Str("TxHash", signRes.TxHash).Msg("[ScalarClient] [handleTokenSentEvent] Successfully broadcasted sign commands request. Waiting for sign event...")
// 	//3. Wait for the sign event
// 	//Todo: Check if the sign event is received
// 	batchCommandId, commandIDs := c.waitForSignCommandsEvent(ctx, signRes.TxHash)
// 	if batchCommandId == "" || commandIDs == "" {
// 		return nil, fmt.Errorf("BatchCommandId not found")
// 	}
// 	log.Debug().Str("BatchCommandId", batchCommandId).Msgf("[ScalarClient] [handleTokenSentEvent] Successfully received sign commands event.")
// 	// 2. Old version, loop for get ExecuteData from batch command id
// 	executeData, err := c.waitForExecuteData(ctx, destinationChain, batchCommandId)
// 	if err != nil {
// 		return nil, fmt.Errorf("[ScalarClient] [handleTokenSentEvent] failed to get execute data: %w", err)
// 	}
// 	eventEnvelope := events.EventEnvelope{
// 		EventType:        events.EVENT_SCALAR_DEST_CALL_APPROVED,
// 		DestinationChain: destinationChain,
// 		Data:             executeData,
// 	}
// 	log.Debug().Str("EventType", eventEnvelope.EventType).
// 		Str("DestinationChain", eventEnvelope.DestinationChain).
// 		Msg("[ScalarClient] [handleTokenSentEvent] broadcast to eventBus")
// 	// 3. Broadcast the execute data to the Event bus
// 	// Todo:After the executeData is broadcasted,
// 	// Update status of the RelayerData to Approved
// 	c.eventBus.BroadcastEvent(&eventEnvelope)
// 	return &db.RelaydataExecuteResult{
// 		Status:      db.APPROVED,
// 		RelayDataId: string(event.Args.DestinationChain),
// 	}, nil
// }

func (c *Client) preprocessTokenSentEvent(event *IBCEvent[types.EventTokenSent]) error {
	log.Debug().Interface("event", event).Msg("[ScalarClient] [preprocessTokenSentEvent]")
	//Check if the destination chain is supported
	// destChain := strings.ToLower(event.Args.DestinationChain)

	return nil
}

func (c *Client) handleDestCallApprovedEvents(ctx context.Context, events []IBCEvent[*types.DestCallApproved]) error {
	updates := make([]db.RelaydataExecuteResult, 0)
	for _, event := range events {
		result, err := c.handleDestCallApprovedEvent(ctx, &event)
		if err != nil {
			return err
		}
		if result != nil {
			updates = append(updates, *result)
		}
	}
	return c.dbAdapter.UpdateBatchRelayDataStatus(updates, len(updates))
}
func (c *Client) handleDestCallApprovedEvent(ctx context.Context, event *IBCEvent[*types.DestCallApproved]) (*db.RelaydataExecuteResult, error) {
	err := c.preprocessDestCallApprovedEvent(event)
	if err != nil {
		return nil, err
	}
	//1. Get pending command from Scalar network
	destinationChain := event.Args.DestinationChain.String()
	pendingCommands, err := c.queryClient.QueryPendingCommand(ctx, destinationChain)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending command: %w", err)
	}
	if len(pendingCommands) == 0 {
		log.Debug().Msgf("[ScalarClient] [handleDestCallApprovedEvent] No pending command found")
		return nil, nil
	}
	log.Debug().Any("pendingCommands", pendingCommands).Msgf("[ScalarClient] [handleDestCallApprovedEvent]")
	//2. Sign the commands request
	signRes, err := c.network.SignCommandsRequest(ctx, destinationChain)
	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
		return nil, fmt.Errorf("[ScalarClient] [handleDestCallApprovedEvent] failed to sign commands request: %v, %w", signRes, err)
	}
	log.Debug().Msgf("[ScalarClient] [handleDestCallApprovedEvent] Successfully broadcasted sign commands request with txHash: %s. Waiting for sign event...", signRes.TxHash)
	//3. Wait for the sign event
	//Todo: Check if the sign event is received
	batchCommandId, commandIDs := c.waitForSignCommandsEvent(ctx, signRes.TxHash)
	if batchCommandId == "" || commandIDs == "" {
		return nil, fmt.Errorf("BatchCommandId not found")
	}
	log.Debug().Msgf("[ScalarClient] [handleDestCallApprovedEvent] Successfully received sign commands event with batch command id: %s", batchCommandId)
	// 2. Old version, loop for get ExecuteData from batch command id
	batchCmdRes, err := c.waitForExecuteData(ctx, destinationChain, batchCommandId)
	if err != nil {
		return nil, fmt.Errorf("[ScalarClient] [handleDestCallApprovedEvent] failed to get execute data: %w", err)
	}
	eventEnvelope := events.EventEnvelope{
		EventType:        events.EVENT_SCALAR_DEST_CALL_APPROVED,
		DestinationChain: destinationChain,
		MessageID:        string(event.Args.EventID),
		Data:             batchCmdRes.ExecuteData,
	}
	log.Debug().Msgf("[ScalarClient] [handleContractCallApprovedEvent] broadcast to eventBus: EventType: %s, DestinationChain: %s, MessageID: %v",
		eventEnvelope.EventType, eventEnvelope.DestinationChain, eventEnvelope.MessageID)
	// 3. Broadcast the execute data to the Event bus
	// Todo:After the executeData is broadcasted,
	// Update status of the RelayerData to Approved
	c.eventBus.BroadcastEvent(&eventEnvelope)
	return &db.RelaydataExecuteResult{
		Status:      db.APPROVED,
		RelayDataId: string(event.Args.EventID),
	}, nil
}

func (c *Client) preprocessDestCallApprovedEvent(event *IBCEvent[*types.DestCallApproved]) error {
	log.Debug().Interface("event", event).Msg("[ScalarClient] [preprocessContractCallApprovedEvent]")
	//Check if the destination chain is supported
	// destChain := strings.ToLower(event.Args.DestinationChain)

	return nil
}

func (c *Client) handleSignCommandsEvents(ctx context.Context, events []IBCEvent[SignCommands]) error {
	for _, event := range events {
		err := c.handleSignCommandsEvent(ctx, &event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) handleSignCommandsEvent(ctx context.Context, event *IBCEvent[SignCommands]) error {
	//1. Get Tx info by tx hash
	batchCommandId, err := c.findBatchCommandId(ctx, event.Args.TxHash)
	if err != nil {
		return fmt.Errorf("[ScalarClient] [handleSignCommandsEvent] failed to find batch command id: %w", err)
	}
	// 2. Old version, loop for get ExecuteData from batch command id
	executeData, err := c.waitForExecuteData(ctx, event.Args.DestinationChain, batchCommandId)
	if err != nil {
		return fmt.Errorf("[ScalarClient] [handleSignCommandsEvent] failed to get execute data: %w", err)
	}
	log.Debug().Msgf("[ScalarClient] [handleSignCommandsEvent] found executeData: %s", executeData)
	// 3. Broadcast the execute data to the Event bus
	// Todo:After the executeData is broadcasted,
	// Update status of the RelayerData to Approved
	if c.eventBus != nil {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_SCALAR_DEST_CALL_APPROVED,
			DestinationChain: event.Args.DestinationChain,
			MessageID:        event.Args.MessageID,
			Data:             executeData,
		})
	} else {
		log.Warn().Msg("[ScalarClient] [handleSignCommandsEvent] event bus is undefined")
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
	//First time wait for 5 seconds
	time.Sleep(5 * time.Second)
	for {
		txRes, err = c.queryClient.QueryTx(ctx, txHash)
		if err != nil || txRes == nil || txRes.Code != 0 || len(txRes.Logs) == 0 {
			log.Debug().Err(err).Msgf("[ScalarClient] [waitForSignCommandsEvent]")
			//Wait for 2 seconds before retry
			time.Sleep(2 * time.Second)
			continue
		}
		log.Debug().
			Str("TxHash", txRes.TxHash).
			Any("Log events", txRes.Logs[0].Events).
			Msg("[ScalarClient] [waitForSignCommandsEvent]")
		batchCommandId := findEventAttribute(txRes.Logs[0].Events, "sign", "batchedCommandID")
		commandIDs := findEventAttribute(txRes.Logs[0].Events, "sign", "commandIDs")
		if batchCommandId == "" {
			log.Debug().Msgf("[ScalarClient] [waitForSignCommandsEvent] no batch command id found in the tx: %v", txRes)
		}
		return batchCommandId, commandIDs
	}
}
func (c *Client) waitForExecuteData(ctx context.Context, destinationChain string, batchCommandId string) (*types.BatchedCommandsResponse, error) {
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
func (c *Client) preprocessContractCallApprovedEvent(event *IBCEvent[ContractCallSubmitted]) error {
	log.Debug().Interface("event", event).Msg("[ScalarClient] [preprocessContractCallApprovedEvent]")
	//Check if the destination chain is supported
	// destChain := strings.ToLower(event.Args.DestinationChain)

	return nil
}

func (c *Client) handleEVMCompletedEvents(ctx context.Context, events []IBCEvent[*types.ChainEventCompleted]) error {
	for _, event := range events {
		err := c.handleEVMCompletedEvent(ctx, &event)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) handleEVMCompletedEvent(ctx context.Context, event *IBCEvent[*types.ChainEventCompleted]) error {
	payload, err := c.preprocessEVMCompletedEvent(event)
	if err != nil {
		return err
	}
	status := db.FAILED
	var sequence *int = nil
	var eventId = string(event.Args.EventID)
	//1. Sign and broadcast RouteMessageRequest
	txRes, err := c.network.SendRouteMessageRequest(ctx, eventId, payload.(string))
	if err != nil {
		log.Error().Msgf("failed to send route message request: %+v", err)
	}
	log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] txRes: %v", txRes)
	if strings.Contains(txRes.RawLog, "already executed") {
		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Already sent an executed tx for %s. Marked it as success.", eventId)
		status = db.SUCCESS
	} else {
		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Executed RouteMessageRequest %s.", txRes.TxHash)
		attrValue := findEventAttribute(txRes.Logs[0].Events, "send_packet", "packet_sequence")
		if attrValue != "" {
			value, _ := strconv.Atoi(attrValue)
			sequence = &value
		}
		status = db.FAILED
	}
	//2. Update the db
	err = c.dbAdapter.UpdateRelayDataStatueWithPacketSequence(eventId, status, sequence)
	if err != nil {
		return fmt.Errorf("failed to update contract call approved: %w", err)
	}
	return nil
}

func (c *Client) preprocessEVMCompletedEvent(event *IBCEvent[*types.ChainEventCompleted]) (any, error) {
	log.Debug().Msgf("EVMCompletedEvent: %v", event)
	// Load payload from the db
	includeCallContract := true
	queryOption := &db.QueryOptions{
		IncludeCallContract: &includeCallContract,
	}
	relayData, err := c.dbAdapter.FindRelayDataById(string(event.Args.EventID), queryOption)
	if err != nil {
		return "", fmt.Errorf("failed to get payload: %w", err)
	}
	payload := hex.EncodeToString(relayData.CallContract.Payload)
	return payload, nil
}

func (c *Client) handleAnyEvents(ctx context.Context, events []IBCEvent[any]) error {
	log.Debug().Msgf("[ScalarClient] [handleAnyEvents] events: %v", events)
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
