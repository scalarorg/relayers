package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/pkg/events"
)

var (
	ALL_EVENTS = []string{
		events.EVENT_EVM_CONTRACT_CALL,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_COMMAND_EXECUTED,
	}
)

func (c *EvmClient) AppendLogs(logs []types.Log) {
	c.missingLogs.AppendLogs(logs)
}

// Go routine for process missing logs
func (c *EvmClient) ProcessMissingLogs() {
	events := map[string]abi.Event{}
	for _, event := range scalarGatewayAbi.Events {
		events[event.ID.String()] = event
	}
	for !c.missingLogs.IsRecovered() {
		logs := c.missingLogs.GetLogs(100)
		for _, txLog := range logs {
			topic := txLog.Topics[0].String()
			event, ok := events[topic]
			if !ok {
				log.Error().Str("topic", topic).Any("txLog", txLog).Msg("[EvmClient] [ProcessMissingLogs] event not found")
				continue
			}
			log.Debug().Str("eventName", event.Name).
				Str("txHash", txLog.TxHash.String()).
				Msg("[EvmClient] [RecoverEvent] start handling missing event")
			err := c.handleEventLog(event, txLog)
			if err != nil {
				log.Error().Err(err).Msg("[EvmClient] [ProcessMissingLogs] failed to handle event log")
			}
		}
	}
}
func (c *EvmClient) RecoverAllEvents(ctx context.Context) error {
	return c.RecoverEvents(ctx, ALL_EVENTS)
}
func (c *EvmClient) RecoverEvents(ctx context.Context, eventNames []string) error {
	topics := [][]common.Hash{}
	events := map[string]abi.Event{}
	for _, eventName := range eventNames {
		event, ok := scalarGatewayAbi.Events[eventName]
		if ok {
			topics = append(topics, []common.Hash{event.ID})
			events[event.ID.String()] = event
		}
	}
	lastCheckpoint, err := c.dbAdapter.GetLastCheckPoint(c.EvmConfig.GetId(), c.EvmConfig.LastBlock)
	if err != nil {
		return fmt.Errorf("failed to get last checkpoint: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Any("LastCheckpoint", lastCheckpoint).Msg("[EvmClient] [RecoverEvent]")
	//Get current block number
	blockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Uint64("Current BlockNumber", blockNumber).Msg("[EvmClient] [RecoverThenWatchForEvent]")
	if blockNumber-lastCheckpoint.BlockNumber > c.EvmConfig.MaxRecoverRange {
		lastCheckpoint.BlockNumber = blockNumber - c.EvmConfig.MaxRecoverRange + 1 //We add extra 1 to make sure we don't cover over configured range
	}
	lastBlock := lastCheckpoint.BlockNumber + c.EvmConfig.MaxRecoverRange
	for lastBlock < blockNumber {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(lastCheckpoint.BlockNumber)),
			ToBlock:   big.NewInt(int64(lastBlock)),
			Addresses: []common.Address{c.GatewayAddress},
			Topics:    topics,
		}
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to get current block number: %w", err)
		}
		c.AppendLogs(logs)
		c.UpdateLastCheckPoint(events, logs, lastBlock)
		blockNumber, err = c.Client.BlockNumber(context.Background())
		if err != nil {
			return fmt.Errorf("failed to get current block number: %w", err)
		}
	}
	return nil
}

func (c *EvmClient) UpdateLastCheckPoint(events map[string]abi.Event, logs []types.Log, lastBlock uint64) {
	eventCheckPoints := map[string]scalarnet.EventCheckPoint{}
	for _, txLog := range logs {
		topic := txLog.Topics[0].String()
		event, ok := events[topic]
		if !ok {
			log.Error().Str("topic", topic).Any("txLog", txLog).Msg("[EvmClient] [UpdateLastCheckPoint] event not found")
			continue
		}
		checkpoint, ok := eventCheckPoints[event.Name]
		if !ok {
			checkpoint = scalarnet.EventCheckPoint{
				ChainName:   c.EvmConfig.ID,
				EventName:   event.Name,
				BlockNumber: lastBlock,
				LogIndex:    txLog.Index,
				TxHash:      txLog.TxHash.String(),
			}
			eventCheckPoints[event.Name] = checkpoint
		} else {
			checkpoint.BlockNumber = lastBlock
			checkpoint.LogIndex = txLog.Index
			checkpoint.TxHash = txLog.TxHash.String()
		}
	}
	c.dbAdapter.UpdateLastEventCheckPoints(eventCheckPoints)
}
