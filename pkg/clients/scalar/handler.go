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
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

func (c *Client) handleContractCallApprovedEvent(ctx context.Context, event *IBCEvent[ContractCallApproved]) error {
	err := c.preprocessContractCallApprovedEvent(event)
	if err != nil {
		return err
	}
	//1. Get pending command from Scalar network
	destinationChain := event.Args.DestinationChain
	pendingCommands, err := c.queryClient.QueryPendingCommand(ctx, destinationChain)
	log.Debug().Msgf("Pending command: %v", pendingCommands)
	if len(pendingCommands) > 0 {
		return fmt.Errorf("no pending command found")
	}
	//2. Sign the commands request
	signRes, err := c.network.SignCommandsRequest(ctx, destinationChain)
	if err != nil {
		return fmt.Errorf("failed to sign commands request: %w", err)
	}
	if strings.Contains(signRes.RawLog, "failed") {
		return fmt.Errorf("failed to sign commands request: %v", signRes.RawLog)
	}
	// 3. Old version, loop for get ExecuteData from batch command id
	executeData, err := c.waitForExecuteData(ctx, destinationChain, signRes)
	if err != nil {
		return fmt.Errorf("failed to get execute data: %w", err)
	}
	// 4. Update the db
	// err = c.dbAdapter.UpdateContractCallApproved(event.Args.MessageID, executeData)
	// if err != nil {
	// 	return fmt.Errorf("failed to update contract call approved: %w", err)
	// }
	// 5. Broadcast the execute data to the Event bus
	// Todo:After the executeData is broadcasted,
	// Update status of the RelayerData to Approved
	c.eventBus.BroadcastEvent(&events.EventEnvelope{
		EventType:        events.EVENT_SCALAR_CONTRACT_CALL_APPROVED,
		DestinationChain: destinationChain,
		MessageID:        event.Args.MessageID,
		Data:             executeData,
	},
	)
	return nil
}

func (c *Client) waitForExecuteData(ctx context.Context, destinationChain string, signRes *sdk.TxResponse) (string, error) {
	batchCommandId := findEventAttribute(signRes.Logs[0].Events, "sign", "batchedCommandID")
	if batchCommandId == "" {
		return "", fmt.Errorf("failed to find batch command id")
	}
	res, err := c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
	for {
		if err != nil {
			return "", fmt.Errorf("failed to get batched commands: %w", err)
		}
		if res.Status != 3 {
			time.Sleep(3 * time.Second)
			res, err = c.queryClient.QueryBatchedCommands(ctx, destinationChain, batchCommandId)
		}
		return res.ExecuteData, nil
	}
}
func (c *Client) preprocessContractCallApprovedEvent(event *IBCEvent[ContractCallSubmitted]) error {
	log.Debug().Msgf("ContractCallSubmittedEvent: %v", event)
	//Check if the destination chain is supported
	// destChain := strings.ToLower(event.Args.DestinationChain)

	return nil
}

func (c *Client) handleEVMCompletedEvent(ctx context.Context, event *IBCEvent[EVMEventCompleted]) error {
	err := c.preprocessEVMCompletedEvent(event)
	if err != nil {
		return err
	}
	status := relaydata.FAILED
	var sequence *int = nil
	//1. Sign and broadcast RouteMessageRequest
	txRes, err := c.network.SendRouteMessageRequest(ctx, event.Args.ID, event.Args.Payload)
	if err != nil {
		log.Error().Msgf("failed to send route message request: %+v", err)
	}
	log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] txRes: %v", txRes)
	if strings.Contains(txRes.RawLog, "already executed") {
		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Already sent an executed tx for %s. Marked it as success.", event.Args.ID)
		status = relaydata.SUCCESS
	} else {
		log.Debug().Msgf("[ScalarClient] [handleEVMCompletedEvent] Executed RouteMessageRequest %s.", txRes.TxHash)
		attrValue := findEventAttribute(txRes.Logs[0].Events, "send_packet", "packet_sequence")
		if attrValue != "" {
			value, _ := strconv.Atoi(attrValue)
			sequence = &value
		}
		status = relaydata.FAILED
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

func extractPacketSequence(events []sdk.StringEvent) *int {
	for _, event := range events {
		if event.Type == "send_packet" {
			for _, attr := range event.Attributes {
				if attr.Key == "packet_sequence" {
					sequence, _ := strconv.Atoi(attr.Value)
					return &sequence
				}
			}
		}
	}
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
