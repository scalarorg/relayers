package evm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	chainsModel "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/events"
	pkgTypes "github.com/scalarorg/relayers/pkg/types"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
	chains "github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
)

var (
	ALL_EVENTS = []string{
		events.EVENT_EVM_CONTRACT_CALL,
		events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		events.EVENT_EVM_TOKEN_SENT,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED,
		events.EVENT_EVM_COMMAND_EXECUTED,
		//events.EVENT_EVM_TOKEN_DEPLOYED,
		//events.EVENT_EVM_REDEEM_TOKEN,
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
	//Waiting for all redeem confirm request to be handled and redeem commands ready in the pending command queue

	c.WaitForRedeemCommandConfirmed(c.MissingLogs.RedeemTxs)
	//Recover all redeem commands
	mapExecutingEvents := c.MissingLogs.GetExecutingEvents()
	groupUids := []string{}
	for groupUid, executingEvent := range mapExecutingEvents {
		log.Info().Str("Chain", c.EvmConfig.ID).Str("GroupUid", groupUid).Msgf("[EvmClient] [RecoverAllEvents] handle switched phase event to executing")
		c.HandleSwitchPhase(executingEvent)
		groupUids = append(groupUids, groupUid)
	}
	//Wait for scalar network switch to Executing phase
	c.WaitForSwitchingToPhase(groupUids, covExported.Executing)
	log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [ProcessMissingLogs] finished processing all missing evm events")
}

// Recover all events after recovering
func (c *EvmClient) RecoverAllEvents(ctx context.Context, groups []*covExported.CustodianGroup) error {
	currentBlockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Uint64("Current BlockNumber", currentBlockNumber).
		Msg("[EvmClient] [RecoverAllEvents] recovering all events")

	//Recover switched phase event
	// mapPreparingEvents, mapExecutingEvents, err := c.RecoverSwitchedPhaseEvent(ctx, currentBlockNumber, groups)
	// if err != nil {
	// 	return err
	// }
	// log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverAllEvents] recovered %d preparing events and %d executing events", len(mapPreparingEvents), len(mapExecutingEvents))
	// //First handle all preparing events
	// //TODO: turn on the flag Recovering
	// groupUids := []string{}
	// for groupUid, preparingEvent := range mapPreparingEvents {
	// 	log.Info().Str("Chain", c.EvmConfig.ID).Str("GroupUid", groupUid).Msgf("[EvmClient] [RecoverAllEvents] handle switched phase event to preparing")
	// 	c.HandleSwitchPhase(preparingEvent)
	// 	groupUids = append(groupUids, groupUid)
	// }
	// c.MissingLogs.SetLastSwitchedEvents(mapPreparingEvents, mapExecutingEvents)
	// //Wait for scalar network switch to preparing phase
	// c.WaitForSwitchingToPhase(groupUids, covExported.Preparing)
	//Recover all other events
	err = c.RecoverEvents(ctx, ALL_EVENTS, currentBlockNumber)
	if err != nil {
		return err
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Msg("[EvmClient] [RecoverAllEvents] recovered all events set recovered flag to true")
	c.MissingLogs.SetRecovered(true)
	return nil
}

func (c *EvmClient) WaitForSwitchingToPhase(groupUids []string, expectedPhase covExported.Phase) error {
	waitingGroupUids := map[string]bool{}
	for _, groupUid := range groupUids {
		waitingGroupUids[groupUid] = true
	}
	for len(waitingGroupUids) > 0 {
		time.Sleep(2 * time.Second)
		for groupHex := range waitingGroupUids {
			redeemSession, err := c.ScalarClient.GetRedeemSession(groupHex)
			if err != nil {
				log.Warn().Err(err).Msgf("[EvmClient] [RecoverAllEvents] failed to get current redeem session from scalarnet")
			}
			if redeemSession.Session.CurrentPhase == expectedPhase {
				delete(waitingGroupUids, groupHex)
			}
			log.Info().Str("Chain", c.EvmConfig.ID).Str("GroupUid", groupHex).
				Any("Session", redeemSession.Session).
				Msgf("[EvmClient] [WaitForSwitchingToPhase] waiting for group %d to switch to expected phase %v", len(waitingGroupUids), expectedPhase)
		}
	}
	return nil
}

func (c *EvmClient) WaitForRedeemCommandConfirmed(redeemTxs map[string][]string) {
	wg := sync.WaitGroup{}
	for chainId, txs := range redeemTxs {
		wg.Add(1)
		go func(chainId string, txs []string) {
			defer wg.Done()
			err := c.waitForPendingCommands(chainId, txs)
			if err != nil {
				log.Error().Err(err).Msgf("[EvmClient] [WaitForRedeemCommandConfirmed] failed to wait for redeem command confirmed: %s", err)
			}
		}(chainId, txs)
	}
	wg.Wait()
}
func (c *EvmClient) waitForPendingCommands(chainId string, sourceTxs []string) error {
	chainClient := c.ScalarClient.GetChainQueryServiceClient()
	request := &chains.PendingCommandsRequest{
		Chain: chainId,
	}
	waitingTxs := map[string]bool{}
	for _, tx := range sourceTxs {
		waitingTxs[tx] = true
	}
	for len(waitingTxs) > 0 {
		time.Sleep(3 * time.Second)
		pendingCommands, err := chainClient.PendingCommands(context.Background(), request)
		if err != nil {
			log.Error().Err(err).Msgf("[EvmClient] [waitForPendingCommands] failed to get pending commands: %s", err)
			return fmt.Errorf("failed to get pending commands: %w", err)
		}
		for _, command := range pendingCommands.Commands {
			txHash := command.Params["sourceTxHash"]
			log.Info().Str("Chain", c.EvmConfig.ID).Str("TxHash", txHash).Msg("[EvmClient] [waitForPendingCommands] found pending command")
			delete(waitingTxs, txHash)
		}
	}
	return nil
}

/*
For each evm chain, we need to recover from the last Event which switch to PrepringPhase
So, we need to recover one event Preparing if it is last
or 2 last events, Preparing and Executing, beetween 2 this events, relayer push all redeem transactions of current session
*/
// func (c *EvmClient) RecoverSwitchedPhaseEvent(ctx context.Context, blockNumber uint64, groups []*covExported.CustodianGroup) (
// 	map[string]*contracts.IScalarGatewaySwitchPhase, map[string]*contracts.IScalarGatewaySwitchPhase, error) {
// 	expectingGroups := map[string]string{}
// 	for _, group := range groups {
// 		groupUid := strings.TrimPrefix(group.UID.Hex(), "0x")
// 		expectingGroups[groupUid] = group.Name
// 	}
// 	mapPreparingEvents := map[string]*contracts.IScalarGatewaySwitchPhase{}
// 	mapExecutingEvents := map[string]*contracts.IScalarGatewaySwitchPhase{}
// 	event, ok := scalarGatewayAbi.Events[events.EVENT_EVM_SWITCHED_PHASE]
// 	if !ok {
// 		return nil, nil, fmt.Errorf("switched phase event not found")
// 	}
// 	recoverRange := uint64(100000)
// 	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < 100000 {
// 		recoverRange = c.EvmConfig.RecoverRange
// 	}
// 	var fromBlock uint64
// 	if blockNumber < recoverRange {
// 		fromBlock = 0
// 	} else {
// 		fromBlock = blockNumber - recoverRange
// 	}
// 	toBlock := blockNumber
// 	for len(expectingGroups) > 0 {
// 		query := ethereum.FilterQuery{
// 			FromBlock: big.NewInt(int64(fromBlock)),
// 			ToBlock:   big.NewInt(int64(toBlock)),
// 			Addresses: []common.Address{c.GatewayAddress},
// 			Topics:    [][]common.Hash{{event.ID}},
// 		}
// 		logs, err := c.Client.FilterLogs(context.Background(), query)
// 		if err != nil {
// 			return nil, nil, fmt.Errorf("failed to filter logs: %w", err)
// 		}
// 		for i := len(logs) - 1; i >= 0; i-- {
// 			switchedPhase := &contracts.IScalarGatewaySwitchPhase{
// 				Raw: logs[i],
// 			}
// 			err := parser.ParseEventData(&logs[i], event.Name, switchedPhase)
// 			if err != nil {
// 				return nil, nil, fmt.Errorf("failed to parse event %s: %w", event.Name, err)
// 			}
// 			groupUid := hex.EncodeToString(switchedPhase.CustodianGroupId[:])
// 			groupUid = strings.TrimPrefix(groupUid, "0x")
// 			switch switchedPhase.To {
// 			case uint8(covExported.Preparing):
// 				log.Info().Str("groupUid", groupUid).Msg("[EvmClient] [RecoverSwitchedPhaseEvent] found preparing event")
// 				_, ok := mapPreparingEvents[groupUid]
// 				if !ok {
// 					mapPreparingEvents[groupUid] = switchedPhase
// 				}
// 				delete(expectingGroups, groupUid)
// 			case uint8(covExported.Executing):
// 				log.Info().Str("groupUid", groupUid).Msg("[EvmClient] [RecoverSwitchedPhaseEvent] found executing event")
// 				_, ok := mapExecutingEvents[groupUid]
// 				if !ok {
// 					mapExecutingEvents[groupUid] = switchedPhase
// 				}
// 			}
// 		}
// 		if fromBlock <= c.EvmConfig.StartBlock {
// 			break
// 		}
// 		toBlock = fromBlock - 1
// 		if fromBlock < recoverRange+c.EvmConfig.StartBlock {
// 			fromBlock = c.EvmConfig.StartBlock
// 		} else {
// 			fromBlock = fromBlock - recoverRange
// 		}
// 	}
// 	if len(expectingGroups) > 0 {
// 		return nil, nil, fmt.Errorf("some groups are not found: %v", expectingGroups)
// 	}
// 	return mapPreparingEvents, mapExecutingEvents, nil
// }
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

/*
Recover all redeem sessions with redeem events from the last switch phase event back to the startBlock in the config
*/
func (c *EvmClient) RecoverAllRedeemSessions(groups []chainsExported.Hash,
	redeemTokenChannel chan *chainsModel.ContractCallWithToken) (*pkgTypes.ChainRedeemSessions, error) {
	log.Info().Str("Chain", c.EvmConfig.ID).
		Uint64("Start block", c.EvmConfig.StartBlock).
		Uint64("Recover range", c.EvmConfig.RecoverRange).
		Msgf("[EvmClient] [RecoverAllRedeemSessions] start recovering redeem sessions")
	currentBlockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("ChainId", c.EvmConfig.ID).Uint64("Current block", currentBlockNumber).Msg("[EvmClient] RecoverAllRedeemSessions")
	if currentBlockNumber < c.EvmConfig.StartBlock {
		return nil, fmt.Errorf("invalid config: start block %d is greater than current block number %d", c.EvmConfig.StartBlock, currentBlockNumber)
	}
	allGroups := map[string]bool{}
	for _, group := range groups {
		groupUid := hex.EncodeToString(group[:])
		allGroups[groupUid] = true
	}
	switchPhaseEvent := scalarGatewayAbi.Events[events.EVENT_EVM_SWITCHED_PHASE]
	redeemTokenEvent := scalarGatewayAbi.Events[events.EVENT_EVM_REDEEM_TOKEN]
	chainRedeemSessions := &pkgTypes.ChainRedeemSessions{
		SwitchPhaseEvents: map[string][]*contracts.IScalarGatewaySwitchPhase{},
		RedeemTokenEvents: map[string][]*contracts.IScalarGatewayRedeemToken{},
	}
	maxSequences := map[string]uint64{}
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.GatewayAddress},
		Topics:    [][]common.Hash{{switchPhaseEvent.ID, redeemTokenEvent.ID}},
	}
	recoverRange := uint64(100000)
	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < recoverRange {
		recoverRange = c.EvmConfig.RecoverRange
	}
	fromBlock := currentBlockNumber - recoverRange + 1
	if fromBlock < c.EvmConfig.StartBlock {
		fromBlock = c.EvmConfig.StartBlock
	}
	toBlock := currentBlockNumber
	switchedPhaseCounter := 0
	for toBlock > c.EvmConfig.StartBlock {
		time.Sleep(5 * time.Second)
		query.FromBlock = big.NewInt(int64(fromBlock))
		query.ToBlock = big.NewInt(int64(toBlock))
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Error().Uint64("FromBlock", fromBlock).
				Uint64("ToBlock", toBlock).Err(err).
				Msg("[EvmClient] [RecoverAllRedeemSessions] Sleep for a while then retry")
			return nil, err
		} else {
			log.Info().Str("Chain", c.EvmConfig.ID).
				Uint64("FromBlock", fromBlock).
				Uint64("ToBlock", toBlock).
				Int("Logs found", len(logs)).
				Msg("[EvmClient] [RecoverAllRedeemSessions]")
		}
		//Loop from the last log to the first log
		for i := len(logs) - 1; i >= 0; i-- {
			topic := logs[i].Topics[0].String()
			if topic == switchPhaseEvent.ID.String() {
				if switchedPhaseCounter == 2 {
					//We need only 2 last switch phase events
					continue
				}
				switchedPhase, err := parseSwitchPhaseEvent(logs[i])
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [RecoverAllRedeemSessions] failed to parse event %s", switchPhaseEvent.Name)
					continue
				}
				groupUid := hex.EncodeToString(switchedPhase.CustodianGroupId[:])
				if _, ok := allGroups[groupUid]; !ok {
					log.Warn().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverAllRedeemSessions] unexpected groupUid")
					continue
				}
				log.Info().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverAllRedeemSessions] found preparing event")
				switchedPhaseCounter = chainRedeemSessions.AppendSwitchPhaseEvent(groupUid, switchedPhase)
			} else if topic == redeemTokenEvent.ID.String() {
				redeemEvent, err := parseRedeemTokenEvent(logs[i])
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [RecoverAllRedeemSessions] failed to parse event %s", redeemTokenEvent.Name)
					continue
				}
				groupUid := hex.EncodeToString(redeemEvent.CustodianGroupUID[:])
				if _, ok := allGroups[groupUid]; !ok {
					log.Warn().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverAllRedeemSessions] unexpected groupUid")
					continue
				}
				log.Info().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverAllRedeemSessions] found redeemToken event")
				redeemToken, err := c.RedeemTokenEvent2Model(redeemEvent)
				if err != nil {
					log.Error().Err(err).Msg("[EvmClient] [RecoverAllRedeemSessions] failed to convert IScalarGatewayRedeemToken to ContractCallWithToken")
					continue
				}
				maxSequence := maxSequences[groupUid]
				if maxSequence <= redeemEvent.Sequence {
					//Add to buffer only redeem event of the latest sequence
					maxSequences[groupUid] = redeemEvent.Sequence
					chainRedeemSessions.AppendRedeemTokenEvent(groupUid, redeemEvent)
				}
				if redeemEvent.Sequence < maxSequence {
					redeemToken.Status = chainsModel.ContractCallStatusSuccess
				}
				redeemTokenChannel <- &redeemToken
			}
		}
		toBlock = fromBlock
		fromBlock = fromBlock - recoverRange + 1
		if fromBlock < c.EvmConfig.StartBlock && toBlock >= c.EvmConfig.StartBlock {
			fromBlock = c.EvmConfig.StartBlock
		}
	}
	return chainRedeemSessions, nil
}

/*
For each evm chain, we need to get 2 last switch phase events
so we have to case, [Preparing, Executing], [Executing, Preparing] or [Preparing]
*/
func (c *EvmClient) RecoverRedeemSessions(groups []*covExported.CustodianGroup) (*pkgTypes.ChainRedeemSessions, error) {
	log.Info().Str("Chain", c.EvmConfig.ID).
		Msgf("[EvmClient] [RecoverRedeemSessions] start recovering redeem sessions")
	currentBlockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %w", err)
	}
	if currentBlockNumber < c.EvmConfig.StartBlock {
		return nil, fmt.Errorf("invalid config: start block %d is greater than current block number %d", c.EvmConfig.StartBlock, currentBlockNumber)
	}
	allGroups := map[string]string{}
	expectingGroups := map[string]string{}
	for _, group := range groups {
		groupUid := strings.TrimPrefix(group.UID.Hex(), "0x")
		allGroups[groupUid] = group.Name
		expectingGroups[groupUid] = group.Name
	}
	switchPhaseEvent := scalarGatewayAbi.Events[events.EVENT_EVM_SWITCHED_PHASE]
	redeemTokenEvent := scalarGatewayAbi.Events[events.EVENT_EVM_REDEEM_TOKEN]
	chainRedeemSessions := &pkgTypes.ChainRedeemSessions{
		SwitchPhaseEvents: map[string][]*contracts.IScalarGatewaySwitchPhase{},
		RedeemTokenEvents: map[string][]*contracts.IScalarGatewayRedeemToken{},
	}
	query := ethereum.FilterQuery{
		Addresses: []common.Address{c.GatewayAddress},
		Topics:    [][]common.Hash{{switchPhaseEvent.ID, redeemTokenEvent.ID}},
	}
	recoverRange := uint64(100000)
	if c.EvmConfig.RecoverRange > 0 && c.EvmConfig.RecoverRange < 100000 {
		recoverRange = c.EvmConfig.RecoverRange
	}
	fromBlock := currentBlockNumber - recoverRange + 1
	if fromBlock < c.EvmConfig.StartBlock {
		fromBlock = c.EvmConfig.StartBlock
	}
	toBlock := currentBlockNumber
	for len(expectingGroups) > 0 {
		query.FromBlock = big.NewInt(int64(fromBlock))
		query.ToBlock = big.NewInt(int64(toBlock))
		logs, err := c.Client.FilterLogs(context.Background(), query)
		if err != nil {
			return nil, fmt.Errorf("failed to filter logs: %w", err)
		}
		log.Info().Str("Chain", c.EvmConfig.ID).Msgf("[EvmClient] [RecoverRedeemSessions] found %d logs, fromBlock: %d, toBlock: %d", len(logs), fromBlock, toBlock)
		//Loop from the last log to the first log
		for i := len(logs) - 1; i >= 0; i-- {
			topic := logs[i].Topics[0].String()
			if topic == switchPhaseEvent.ID.String() {
				switchedPhase, err := parseSwitchPhaseEvent(logs[i])
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [RecoverRedeemSessions] failed to parse event %s", switchPhaseEvent.Name)
					continue
				}
				groupUid := hex.EncodeToString(switchedPhase.CustodianGroupId[:])
				if _, ok := allGroups[groupUid]; !ok {
					log.Warn().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverRedeemSessions] unexpected groupUid")
					continue
				}
				log.Info().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverRedeemSessions] found preparing event")
				counter := chainRedeemSessions.AppendSwitchPhaseEvent(groupUid, switchedPhase)
				if counter == 2 || switchedPhase.To == uint8(covExported.Preparing) && switchedPhase.Sequence == 1 {
					//Stop get logs if we have 2 switch phase events or hit the fist event from sequence 1
					delete(expectingGroups, groupUid)
				}
			} else if topic == redeemTokenEvent.ID.String() {
				redeemToken, err := parseRedeemTokenEvent(logs[i])
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [RecoverRedeemSessions] failed to parse event %s", redeemTokenEvent.Name)
					continue
				}
				groupUid := hex.EncodeToString(redeemToken.CustodianGroupUID[:])
				if _, ok := allGroups[groupUid]; !ok {
					log.Warn().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverRedeemSessions] unexpected groupUid")
					continue
				}
				log.Info().Str("Chain", c.EvmConfig.ID).Str("groupUid", groupUid).Msg("[EvmClient] [RecoverRedeemSessions] found redeemToken event")
				chainRedeemSessions.AppendRedeemTokenEvent(groupUid, redeemToken)
			}
		}
		if fromBlock <= c.EvmConfig.StartBlock {
			break
		}
		toBlock = fromBlock
		if fromBlock < recoverRange+c.EvmConfig.StartBlock {
			fromBlock = c.EvmConfig.StartBlock
		} else {
			fromBlock = fromBlock - recoverRange + 1
		}
	}
	if len(expectingGroups) > 0 {
		log.Warn().Msgf("[EvmClient] [RecoverRedeemSessions] some groups are not recovered: %v", expectingGroups)
		return nil, fmt.Errorf("some groups are not found: %v", expectingGroups)
	}
	return chainRedeemSessions, nil
}

func parseSwitchPhaseEvent(log types.Log) (*contracts.IScalarGatewaySwitchPhase, error) {
	eventName := events.EVENT_EVM_SWITCHED_PHASE
	switchedPhase := &contracts.IScalarGatewaySwitchPhase{
		Raw: log,
	}
	err := parser.ParseEventData(&log, eventName, switchedPhase)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event %s: %w", eventName, err)
	}
	return switchedPhase, nil
}

func parseRedeemTokenEvent(log types.Log) (*contracts.IScalarGatewayRedeemToken, error) {
	eventName := events.EVENT_EVM_REDEEM_TOKEN
	redeemToken := &contracts.IScalarGatewayRedeemToken{
		Raw: log,
	}
	err := parser.ParseEventData(&log, eventName, redeemToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event %s: %w", eventName, err)
	}
	return redeemToken, nil
}
