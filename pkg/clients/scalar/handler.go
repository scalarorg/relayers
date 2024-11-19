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
)

func (c *Client) handleContractCallApprovedEvents(ctx context.Context, events []IBCEvent[ContractCallApproved]) error {
	for _, event := range events {
		err := c.handleContractCallApprovedEvent(ctx, &event)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) handleContractCallApprovedEvent(ctx context.Context, event *IBCEvent[ContractCallApproved]) error {
	err := c.preprocessContractCallApprovedEvent(event)
	if err != nil {
		return err
	}
	//1. Get pending command from Scalar network
	destinationChain := event.Args.DestinationChain
	pendingCommands, err := c.queryClient.QueryPendingCommand(ctx, destinationChain)
	if err != nil {
		return fmt.Errorf("failed to get pending command: %w", err)
	}
	if len(pendingCommands) == 0 {
		return fmt.Errorf("no pending command found")
	}
	log.Debug().Msgf("[ScalarClient] [handleContractCallApprovedEvent] Pending commands: %+v", pendingCommands)
	//2. Sign the commands request
	signRes, err := c.network.SignCommandsRequest(ctx, destinationChain)
	if err != nil || signRes == nil || signRes.Code != 0 || strings.Contains(signRes.RawLog, "failed") || signRes.TxHash == "" {
		return fmt.Errorf("failed to sign commands request: %v, %w", signRes, err)
	}
	log.Debug().Msgf("[ScalarClient] [handleContractCallApprovedEvent] Successfully broadcasted sign commands request with txHash: %s. Waiting for sign event...", signRes.TxHash)
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
	c.eventBus.BroadcastEvent(&events.EventEnvelope{
		EventType:        events.EVENT_SCALAR_CONTRACT_CALL_APPROVED,
		DestinationChain: event.Args.DestinationChain,
		MessageID:        event.Args.MessageID,
		Data:             executeData,
	},
	)
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
func (c *Client) waitForExecuteData(ctx context.Context, destinationChain string, batchCommandId string) (string, error) {
	res, err := c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
	for {
		if err != nil {
			return "", fmt.Errorf("failed to get batched commands: %w", err)
		}
		if res.Status != 3 {
			time.Sleep(3 * time.Second)
			res, err = c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
			if err != nil {
				log.Error().Msgf("batched commands error: %v", err)
			}
		} else {
			break
		}
	}
	return res.ExecuteData, nil
}
func (c *Client) preprocessContractCallApprovedEvent(event *IBCEvent[ContractCallSubmitted]) error {
	log.Debug().Msgf("ContractCallSubmittedEvent: %v", event)
	//Check if the destination chain is supported
	// destChain := strings.ToLower(event.Args.DestinationChain)

	return nil
}

func (c *Client) handleEVMCompletedEvents(ctx context.Context, events []IBCEvent[EVMEventCompleted]) error {
	for _, event := range events {
		err := c.handleEVMCompletedEvent(ctx, &event)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) handleEVMCompletedEvent(ctx context.Context, event *IBCEvent[EVMEventCompleted]) error {
	err := c.preprocessEVMCompletedEvent(event)
	if err != nil {
		return err
	}
	status := db.FAILED
	var sequence *int = nil
	//1. Sign and broadcast RouteMessageRequest
	txRes, err := c.network.SendRouteMessageRequest(ctx, event.Args.ID, event.Args.Payload)
	if err != nil {
		log.Error().Msgf("failed to send route message request: %+v", err)
	}
	log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] txRes: %v", txRes)
	if strings.Contains(txRes.RawLog, "already executed") {
		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Already sent an executed tx for %s. Marked it as success.", event.Args.ID)
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
	err = c.dbAdapter.UpdateRelayDataStatueWithPacketSequence(event.Args.ID, status, sequence)
	if err != nil {
		return fmt.Errorf("failed to update contract call approved: %w", err)
	}
	return nil
}

func (c *Client) preprocessEVMCompletedEvent(event *IBCEvent[EVMEventCompleted]) error {
	log.Debug().Msgf("EVMCompletedEvent: %v", event)
	//Load payload from the db
	includeCallContract := true
	queryOption := &db.QueryOptions{
		IncludeCallContract: &includeCallContract,
	}
	relayData, err := c.dbAdapter.FindRelayDataById(event.Args.ID, queryOption)
	if err != nil {
		return fmt.Errorf("failed to get payload: %w", err)
	}
	event.Args.Payload = hex.EncodeToString(relayData.CallContract.Payload)
	return nil
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
