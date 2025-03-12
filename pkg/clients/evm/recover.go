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
		events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_COMMAND_EXECUTED,
	}
)

func (c *EvmClient) AppendLogs(logs []types.Log) {
	c.MissingLogs.AppendLogs(logs)
}

// Go routine for process missing logs
func (c *EvmClient) ProcessMissingLogs() {
	events := map[string]abi.Event{}
	for _, event := range scalarGatewayAbi.Events {
		events[event.ID.String()] = event
	}
	for !c.MissingLogs.IsRecovered() {
		logs := c.MissingLogs.GetLogs(10)
		for _, txLog := range logs {
			topic := txLog.Topics[0].String()
			event, ok := events[topic]
			if !ok {
				log.Error().Str("topic", topic).Any("txLog", txLog).Msg("[EvmClient] [ProcessMissingLogs] event not found")
				continue
			}
			log.Debug().
				Str("chainId", c.EvmConfig.GetId()).
				Str("eventName", event.Name).
				Str("txHash", txLog.TxHash.String()).
				Msg("[EvmClient] [ProcessMissingLogs] start processing missing event")
			err := c.handleEventLog(event, txLog)
			if err != nil {
				log.Error().Err(err).Msg("[EvmClient] [ProcessMissingLogs] failed to handle event log")
			}
		}
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [ProcessMissingLogs] finished processing all missing evm events")
}
func (c *EvmClient) RecoverAllEvents(ctx context.Context) error {
	log.Info().Str("Chain", c.EvmConfig.ID).Any("events", ALL_EVENTS).Msg("[EvmClient] [RecoverAllEvents] recovering all events")
	return c.RecoverEvents(ctx, ALL_EVENTS)
}
func (c *EvmClient) RecoverEvents(ctx context.Context, eventNames []string) error {
	topics := []common.Hash{}
	events := map[string]abi.Event{}
	for _, eventName := range eventNames {
		event, ok := scalarGatewayAbi.Events[eventName]
		if ok {
			topics = append(topics, event.ID)
			events[event.ID.String()] = event
		}
	}
	var lastCheckpoint *scalarnet.EventCheckPoint
	var err error
	if c.dbAdapter != nil {
		lastCheckpoint, err = c.dbAdapter.GetLastCheckPoint(c.EvmConfig.GetId(), c.EvmConfig.LastBlock)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [RecoverEvents] failed to get last checkpoint use default value")
		}
	} else {
		log.Warn().Msgf("[EvmClient] [RecoverEvents] dbAdapter is nil, use default value")
		lastCheckpoint = &scalarnet.EventCheckPoint{
			ChainName:   c.EvmConfig.ID,
			EventName:   "",
			BlockNumber: c.EvmConfig.LastBlock,
		}
	}
	log.Info().Str("Chain", c.EvmConfig.ID).
		Any("Topics", topics).
		Any("LastCheckpoint", lastCheckpoint).Msg("[EvmClient] [RecoverEvents] start recovering events")
	//Get current block number
	blockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Uint64("Current BlockNumber", blockNumber).Msg("[EvmClient] [RecoverEvents]")
	fromBlock := lastCheckpoint.BlockNumber
	recoverRange := uint64(100000)
	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < 100000 {
		recoverRange = c.EvmConfig.RecoverRange
	}
	logCounter := 0
	for fromBlock < blockNumber {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(fromBlock)),
			Addresses: []common.Address{c.GatewayAddress},
			Topics:    [][]common.Hash{topics},
		}
		if fromBlock+recoverRange < blockNumber {
			query.ToBlock = big.NewInt(int64(fromBlock + recoverRange))
		}
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to filter logs: %w", err)
		}
		//Set toBlock to the last block number for logging purpose
		var toBlock uint64 = blockNumber
		if query.ToBlock != nil {
			toBlock = query.ToBlock.Uint64()
		}
		if len(logs) > 0 {
			log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverEvents] found %d logs, fromBlock: %d, toBlock: %d", len(logs), fromBlock, toBlock)
			c.AppendLogs(logs)
			logCounter += len(logs)
			if c.dbAdapter != nil {
				c.UpdateLastCheckPoint(events, logs, toBlock)
			}
		} else {
			log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverEvents] no logs found, fromBlock: %d, toBlock: %d", fromBlock, toBlock)
		}
		if query.ToBlock != nil {
			blockNumber, err = c.Client.BlockNumber(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("[EvmClient] [RecoverEvents] failed to get current block number, fromBlock: %d, toBlock: %d", fromBlock, toBlock)
			}
		}
		//Set fromBlock to the next block number for next iteration
		fromBlock = toBlock + 1
	}
	c.MissingLogs.SetRecovered(true)
	log.Info().
		Str("Chain", c.EvmConfig.ID).
		Uint64("BlockNumber", blockNumber).
		Int("TotalLogs", logCounter).
		Msg("[EvmClient] [FinishRecover] recovered all events")
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
