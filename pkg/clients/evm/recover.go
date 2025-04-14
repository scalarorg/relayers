package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/scalarnet"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/events"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
)

var (
	ALL_EVENTS = []string{
		events.EVENT_EVM_CONTRACT_CALL,
		events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		//events.EVENT_EVM_COMMAND_EXECUTED,
		//events.EVENT_EVM_TOKEN_DEPLOYED,
		events.EVENT_EVM_REDEEM_TOKEN,
	}
)

// Go routine for process missing logs
func (c *EvmClient) ProcessMissingLogs() {
	mapEvents := map[string]abi.Event{}
	for _, event := range scalarGatewayAbi.Events {
		mapEvents[event.ID.String()] = event
	}
	for {
		logs := c.MissingLogs.GetLogs(10)
		if len(logs) == 0 {
			if c.MissingLogs.IsRecovered() {
				log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [ProcessMissingLogs] no logs to process, recovered flag is true, exit")
				break
			} else {
				log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [ProcessMissingLogs] no logs to process, recover is in progress, sleep 1 second then continue")
				time.Sleep(time.Second)
				continue
			}
		}
		log.Info().Str("Chain", c.EvmConfig.ID).Int("Number of logs", len(logs)).Msg("[EvmClient] [ProcessMissingLogs] processing logs")
		for _, txLog := range logs {
			topic := txLog.Topics[0].String()
			event, ok := mapEvents[topic]
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

func (c *EvmClient) RecoverAllEvents(ctx context.Context, groups []*covExported.CustodianGroup) error {
	currentBlockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Uint64("Current BlockNumber", currentBlockNumber).
		Msg("[EvmClient] [RecoverAllEvents] recovering all events")

	//Recover switched phase event
	mapSwitchedPhaseEvents, err := c.RecoverSwitchedPhaseEvent(ctx, currentBlockNumber, groups)
	if err != nil {
		return err
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverAllEvents] recovered %d switched phase events", len(mapSwitchedPhaseEvents))
	for groupUid, switchedPhaseEvent := range mapSwitchedPhaseEvents {
		log.Info().Str("Chain", c.EvmConfig.ID).Str("GroupUid", groupUid).Msgf("[EvmClient] [RecoverAllEvents] handle switched phase event")
		c.HandleSwitchPhase(switchedPhaseEvent)
	}
	c.MissingLogs.SetLastSwitchedPhaseEvents(mapSwitchedPhaseEvents)
	//Recover all other events
	err = c.RecoverEvents(ctx, ALL_EVENTS, currentBlockNumber)
	if err != nil {
		return err
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [RecoverAllEvents] recovered all events set recovered flag to true")
	c.MissingLogs.SetRecovered(true)
	return nil
}
func (c *EvmClient) RecoverSwitchedPhaseEvent(ctx context.Context, blockNumber uint64, groups []*covExported.CustodianGroup) (map[string]*contracts.IScalarGatewaySwitchPhase, error) {
	expectingGroups := map[string]string{}
	for _, group := range groups {
		groupUid := strings.TrimPrefix(group.UID.Hex(), "0x")
		expectingGroups[groupUid] = group.Name
	}
	groupSwitchPhases := map[string]*contracts.IScalarGatewaySwitchPhase{}
	event, ok := scalarGatewayAbi.Events[events.EVENT_EVM_SWITCHED_PHASE]
	if !ok {
		return nil, fmt.Errorf("switched phase event not found")
	}
	recoverRange := uint64(100000)
	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < 100000 {
		recoverRange = c.EvmConfig.RecoverRange
	}
	var fromBlock uint64
	if blockNumber < recoverRange {
		fromBlock = 0
	} else {
		fromBlock = blockNumber - recoverRange
	}
	toBlock := blockNumber
	for len(expectingGroups) > 0 {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(fromBlock)),
			ToBlock:   big.NewInt(int64(toBlock)),
			Addresses: []common.Address{c.GatewayAddress},
			Topics:    [][]common.Hash{{event.ID}},
		}
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			return nil, fmt.Errorf("failed to filter logs: %w", err)
		}
		for i := len(logs) - 1; i >= 0; i-- {
			switchedPhase := &contracts.IScalarGatewaySwitchPhase{
				Raw: logs[i],
			}
			err := parser.ParseEventData(&logs[i], event.Name, switchedPhase)
			if err != nil {
				return nil, fmt.Errorf("failed to parse event %s: %w", event.Name, err)
			}
			groupUid := hex.EncodeToString(switchedPhase.CustodianGroupId[:])
			groupUid = strings.TrimPrefix(groupUid, "0x")
			_, ok := groupSwitchPhases[groupUid]
			if !ok {
				log.Info().Str("groupUid", groupUid).Msg("[EvmClient] [RecoverSwitchedPhaseEvent] event log not found, set new one")
				groupSwitchPhases[groupUid] = switchedPhase
				delete(expectingGroups, groupUid)
			}
		}
		if fromBlock <= c.EvmConfig.StartBlock {
			break
		}
		toBlock = fromBlock - 1
		if fromBlock < recoverRange+c.EvmConfig.StartBlock {
			fromBlock = c.EvmConfig.StartBlock
		} else {
			fromBlock = fromBlock - recoverRange
		}
	}
	if len(expectingGroups) > 0 {
		return nil, fmt.Errorf("some groups are not found: %v", expectingGroups)
	}
	return groupSwitchPhases, nil
}
func (c *EvmClient) RecoverEvents(ctx context.Context, eventNames []string, currentBlockNumber uint64) error {
	topics := []common.Hash{}
	mapEvents := map[string]abi.Event{}
	for _, eventName := range eventNames {
		//We recover switched phase event in separate function
		if eventName == events.EVENT_EVM_SWITCHED_PHASE {
			continue
		}
		event, ok := scalarGatewayAbi.Events[eventName]
		if ok {
			topics = append(topics, event.ID)
			mapEvents[event.ID.String()] = event
		}
	}
	var lastCheckpoint *scalarnet.EventCheckPoint
	var err error
	if c.dbAdapter != nil {
		lastCheckpoint, err = c.dbAdapter.GetLastCheckPoint(c.EvmConfig.GetId(), c.EvmConfig.StartBlock)
		if err != nil {
			log.Warn().Err(err).Msgf("[EvmClient] [RecoverEvents] failed to get last checkpoint use default value")
		}
	} else {
		log.Warn().Msgf("[EvmClient] [RecoverEvents] dbAdapter is nil, use default value")
		lastCheckpoint = &scalarnet.EventCheckPoint{
			ChainName:   c.EvmConfig.ID,
			EventName:   "",
			BlockNumber: c.EvmConfig.StartBlock,
		}
	}
	log.Info().Str("Chain", c.EvmConfig.ID).
		Str("GatewayAddress", c.GatewayAddress.String()).
		Str("EventNames", strings.Join(eventNames, ",")).
		Any("LastCheckpoint", lastCheckpoint).Msg("[EvmClient] [RecoverEvents] start recovering events")
	recoverRange := uint64(100000)
	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < 100000 {
		recoverRange = c.EvmConfig.RecoverRange
	}
	fromBlock := lastCheckpoint.BlockNumber
	logCounter := 0
	for fromBlock < currentBlockNumber {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(fromBlock)),
			Addresses: []common.Address{c.GatewayAddress},
			Topics:    [][]common.Hash{topics},
		}
		if fromBlock+recoverRange < currentBlockNumber {
			query.ToBlock = big.NewInt(int64(fromBlock + recoverRange))
		} else {
			query.ToBlock = big.NewInt(int64(currentBlockNumber))
		}
		log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverEvents] querying logs fromBlock: %d, toBlock: %d", fromBlock, query.ToBlock.Uint64())
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to filter logs: %w", err)
		}
		if len(logs) > 0 {
			log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverEvents] found %d logs, fromBlock: %d, toBlock: %d", len(logs), fromBlock, query.ToBlock)
			c.MissingLogs.AppendLogs(logs)
			logCounter += len(logs)
			if c.dbAdapter != nil {
				c.UpdateLastCheckPoint(mapEvents, logs, query.ToBlock.Uint64())
			}
		} else {
			log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverEvents] no logs found, fromBlock: %d, toBlock: %d", fromBlock, query.ToBlock)
		}
		//Set fromBlock to the next block number for next iteration
		fromBlock = query.ToBlock.Uint64() + 1
	}
	log.Info().
		Str("Chain", c.EvmConfig.ID).
		Uint64("CurrentBlockNumber", currentBlockNumber).
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
