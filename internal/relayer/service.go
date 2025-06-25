package relayer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/btc"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/indexer"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	types "github.com/scalarorg/relayers/pkg/types"
	chainExported "github.com/scalarorg/scalar-core/x/chains/exported"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covTypes "github.com/scalarorg/scalar-core/x/covenant/types"
)

type Service struct {
	DbAdapter    *db.DatabaseAdapter
	Indexer      *indexer.EVMIndexer
	EventBus     *events.EventBus
	ScalarClient *scalar.Client
	//CustodialClient *custodial.Client
	Electrs    []*electrs.Client
	EvmClients []*evm.EvmClient
	BtcClient  []*btc.BtcClient
}

func NewService(config *config.Config, dbAdapter *db.DatabaseAdapter,
	eventBus *events.EventBus) (*Service, error) {
	var err error

	// Initialize Scalar client
	scalarClient, err := scalar.NewClient(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar client: %w", err)
	}
	// Initialize BTC service
	btcClients, err := btc.NewBtcClients(config, dbAdapter, eventBus, scalarClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create btc clients: %w", err)
	}
	// Initialize Electrs clients
	electrsClients, err := electrs.NewElectrumClients(config, dbAdapter, eventBus, scalarClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create electrum clients: %w", err)
	}
	// Initialize EVM clients
	evmClients, err := evm.NewEvmClients(config, dbAdapter, eventBus, scalarClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm clients: %w", err)
	}

	return &Service{
		DbAdapter:    dbAdapter,
		EventBus:     eventBus,
		ScalarClient: scalarClient,
		Electrs:      electrsClients,
		EvmClients:   evmClients,
		BtcClient:    btcClients,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	//Start scalar client
	if s.ScalarClient != nil {
		go func() {
			err := s.ScalarClient.Start(ctx)
			if err != nil {
				log.Error().Msgf("Start scalar client with error %+v", err)
			}
		}()
	} else {
		log.Warn().Msg("[Relayer] [Start] scalar client is undefined")
	}
	// Improvement recovery evm missing source events
	// 2025, March 10
	// Recover all swiched phase events from evm networks
	groups, err := s.ScalarClient.GetCovenantGroups(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot get covenant groups")
		panic(err)
	}
	groupUids := make([]chainExported.Hash, len(groups))
	for ind, group := range groups {
		groupUids[ind] = group.UID
	}
	//Perform recovery redeem session before recover other events
	err = s.RecoverEvmSessions(groupUids)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover sessions")
		panic(err)
	}
	//Start btc clients
	for _, client := range s.BtcClient {
		go client.Start(ctx)
	}
	//Start electrum clients. This client can get all vault transactions from last checkpoint of begining if no checkpoint is found
	for _, client := range s.Electrs {
		go client.Start(ctx)
	}
	//Recover all events
	for _, client := range s.EvmClients {
		go client.ProcessMissingLogs()
		go func() {
			err := client.RecoverAllEvents(ctx, groups)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover events for evm client %s", client.EvmConfig.GetId())
			}
		}()
		go func() {
			//Todo: Handle the moment when recover just finished and listner has not started yet. It around 1 second
			// err := client.RecoverAllEvents(ctx, groups)
			// if err != nil {
			// 	log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover events for evm client %s", client.EvmConfig.GetId())
			// } else {
			// 	log.Info().Msgf("[Relayer] [Start] recovered missing events for evm client %s", client.EvmConfig.GetId())
			// 	client.Start(ctx)
			// }
			client.Start(ctx)
		}()
	}

	// Recovery evm missing source events
	// for _, client := range s.EvmClients {
	// 	err := client.RecoverInitiatedEvents(ctx)
	// 	if err != nil {
	// 		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover initiated events for evm client %s", client.EvmConfig.GetId())
	// 	}
	// }
	// for _, client := range s.EvmClients {
	// 	err := client.RecoverApprovedEvents(ctx)
	// 	if err != nil {
	// 		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover initiated events for evm client %s", client.EvmConfig.GetId())
	// 	}
	// }
	// for _, client := range s.EvmClients {
	// 	err := client.RecoverExecutedEvents(ctx)
	// 	if err != nil {
	// 		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover initiated events for evm client %s", client.EvmConfig.GetId())
	// 	}
	// }
	// //Start evm clients
	// for _, client := range s.EvmClients {
	// 	go client.Start(ctx)
	// }

	return nil
}
func (s *Service) RecoverEvmSessions(groups []chainExported.Hash) error {
	wg := sync.WaitGroup{}
	recoverSessions := CustodiansRecoverRedeemSessions{}
	redeemTokenChannel := make(chan *chains.EvmRedeemTx)
	//Store all redeem events from all evm chains into db
	go func() {
		for redeemToken := range redeemTokenChannel {
			err := s.DbAdapter.CreateEvmRedeemToken(redeemToken, nil)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot save redeem token to db")
			}
		}
	}()
	for _, client := range s.EvmClients {
		wg.Add(1)
		log.Info().Str("chainId", client.EvmConfig.GetId()).
			Msg("[Relayer] [RecoverEvmSessions] start goroutine for recovering redeem sessions")
		go func() {
			defer wg.Done()
			chainRedeemSessions, err := client.RecoverAllRedeemSessions(groups, redeemTokenChannel)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover sessions for evm client %s", client.EvmConfig.GetId())
			}
			if chainRedeemSessions != nil {
				log.Info().Str("chainId", client.EvmConfig.GetId()).
					//Any("chainRedeemSessions", chainRedeemSessions).
					Msg("[Relayer] [RecoverEvmSessions] add evm session")
				recoverSessions.AddRecoverSessions(client.EvmConfig.GetId(), chainRedeemSessions)
			} else {
				panic(fmt.Sprintf("[Relayer] [RecoverEvmSessions] cannot recover sessions for evm client %s", client.EvmConfig.GetId()))
			}
		}()
	}
	wg.Wait()
	close(redeemTokenChannel)
	log.Info().Msgf("[Relayer] [RecoverEvmSessions] finished get SwitchPhase And redeemTx from evm chains")
	recoverSessions.ConstructSessions()
	for groupUid, groupRedeemSessions := range recoverSessions.RecoverSessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info().Str("groupUid", groupUid).
				Any("maxSession", groupRedeemSessions.MaxSession).
				Any("minSession", groupRedeemSessions.MinSession).
				//Any("switchPhaseEvents", groupRedeemSessions.SwitchPhaseEvents).
				//Any("redeemTokenEvents", groupRedeemSessions.RedeemTokenEvents).
				Msg("[Relayer] [RecoverEvmSessions] recovered redeem session for each group")
			if groupRedeemSessions.MaxSession.Phase == uint8(covExported.Executing) {
				err := s.processRecoverExecutingPhase(groupUid, groupRedeemSessions)
				if err != nil {
					log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot process recover executing phase for group %s", groupUid)
				}
			} else if groupRedeemSessions.MaxSession.Phase == uint8(covExported.Preparing) {
				err := s.processRecoverPreparingPhase(groupUid, groupRedeemSessions)
				if err != nil {
					log.Warn().Err(err).Msgf("[Relayer] [RecoverEvmSessions] cannot process recover preparing phase for group %s", groupUid)
				}
			}
		}()
	}
	wg.Wait()
	log.Info().Msgf("[Relayer] [RecoverEvmSessions] finished RecoverEvmSessions")
	return nil
}
func (s *Service) processRecoverExecutingPhase(groupUid string, groupRedeemSessions *types.GroupRedeemSessions) error {
	log.Info().Str("groupUid", groupUid).
		Msg("[Relayer] [RecoverEvmSessions] processRecoverExecutingPhase")
	//0. Check if the redeem session is broadcasted to bitcoin network
	isBroadcasted, err := s.isRedeemSessionBroadcasted(groupRedeemSessions.RedeemTokenEvents)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutingPhase] cannot check if the redeem session is broadcasted to bitcoin network")
		return err
	}
	if !isBroadcasted {
		log.Info().Msgf("[Relayer] [processRecoverExecutingPhase] redeem session is not broadcasted to bitcoin network")

		//1. Replay all switch to preparing phase event,
		expectedPhase, evmCounter, hasDifferentPhase := s.replaySwitchPhaseEvents(groupRedeemSessions.SwitchPhaseEvents, 0)
		log.Info().Int32("evmCounter", evmCounter).
			Any("ExpectedPhase", expectedPhase).
			Bool("hasDifferentPhase", hasDifferentPhase).
			Msg("[Relayer] [processRecoverExecutingPhase] first events")
		if hasDifferentPhase {
			panic("[Relayer] [processRecoverExecutingPhase] cannot recover all evm switch phase events to the same phase")
		}
		if evmCounter != int32(len(s.EvmClients)) {
			panic(fmt.Sprintf("[Relayer] [processRecoverExecutingPhase] cannot recover all evm switch phase events, evm counter is %d", evmCounter))
		}
		if expectedPhase != int32(covExported.Preparing) {
			panic("[Relayer] [processRecoverExecutingPhase] by design, recover first event switch to Preparing for all evm chains")
		}
		//2. wait for group's session switch to preparing then replay all redeem token events
		err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Preparing)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to preparing phase", groupUid)
			return err
		}

		mapTxHashes, err := s.replayRedeemTransactions(groupUid, groupRedeemSessions.RedeemTokenEvents)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot replay redeem transactions")
			return err
		}
		log.Info().Any("mapTxHashes", mapTxHashes).Msg("[Relayer] [processRecoverExecutionPhase] finished replay redeem transactions")
		s.waitingForPendingCommands(mapTxHashes)
		log.Info().Msgf("[Relayer] [processRecoverExecutionPhase] all pending commands are ready")
	}
	//5. Replay all switch to executing phase events
	expectedPhase, evmCounter, hasDifferentPhase := s.replaySwitchPhaseEvents(groupRedeemSessions.SwitchPhaseEvents, 1)
	log.Info().Int32("evmCounter", evmCounter).
		Any("ExpectedPhase", expectedPhase).
		Bool("hasDifferentPhase", hasDifferentPhase).
		Msg("[Relayer] [processRecoverExecutionPhase] second events")
	if hasDifferentPhase {
		panic("[Relayer] [processRecoverExecutionPhase] cannot recover all evm switch phase events")
	}
	if expectedPhase != int32(covExported.Executing) {
		panic(fmt.Sprintf("[Relayer] [processRecoverExecutionPhase] cannot recover all evm switch phase events, expected phase is %d", expectedPhase))
	}

	if evmCounter == int32(len(s.EvmClients)) {
		log.Info().Int32("evmCounter", evmCounter).Msg("[Relayer] [processRecoverExecutionPhase] all evm chains a in executing phase")
		//All evm chains are in executing phase
		err = s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Executing)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to executing phase", groupUid)
			return err
		}
		err = s.replayBtcRedeemTxs(groupUid)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot replay btc redeem transactions")
			return err
		}
	} else {
		log.Warn().Int32("evmCounter", evmCounter).Msg("[Relayer] [processRecoverExecutionPhase] not all evm chains are in executing phase")
	}
	return nil
}
func (s *Service) processRecoverPreparingPhase(groupUid string, groupRedeemSessions *types.GroupRedeemSessions) error {
	log.Info().Str("groupUid", groupUid).
		Msg("[Relayer] [RecoverEvmSessions] processRecoverPreparingPhase")
	//1. For each evm chain, replay last switch event. It can be Preparing or executing from previous session
	expectedPhase, evmCounter, hasDifferentPhase := s.replaySwitchPhaseEvents(groupRedeemSessions.SwitchPhaseEvents, 0)
	if hasDifferentPhase {
		panic("[Relayer] [processRecoverPreparingPhase] cannot recover all evm switch phase events")
	}
	if evmCounter != int32(len(s.EvmClients)) {
		panic(fmt.Sprintf("[Relayer] [processRecoverPreparingPhase] cannot recover all evm switch phase events, evm counter is %d", evmCounter))
	}
	//2. Waiting for group session switch to expected phase
	err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Phase(expectedPhase))
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot wait for group %s to switch to executing phase", groupUid)
		return err
	}

	if expectedPhase == int32(covExported.Preparing) {
		//3. Replay all redeem transactions
		mapTxHashes, err := s.replayRedeemTransactions(groupUid, groupRedeemSessions.RedeemTokenEvents)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot replay redeem transactions")
			return err
		}
		log.Info().Any("mapTxHashes", mapTxHashes).Msg("[Relayer] [processRecoverPreparingPhase] finished replay redeem transactions")
	} else if expectedPhase == int32(covExported.Executing) {
		err := s.replayBtcRedeemTxs(groupUid)
		if err != nil {
			log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot replay btc redeem transactions")
			return err
		}
	}
	return nil
}

// Find and replay btc redeem tx
func (s *Service) replayBtcRedeemTxs(groupUid string) error {
	log.Info().Str("groupUid", groupUid).Msgf("[Relayer] [processRecoverPreparingPhase] replay btc redeem transactions")
	if s.ScalarClient == nil {
		return fmt.Errorf("[Relayer] [processRecoverPreparingPhase] scalar client is undefined")
	}
	redeemSession, err := s.ScalarClient.GetRedeemSession(groupUid)
	if err != nil {
		return fmt.Errorf("[Relayer] [processRecoverPreparingPhase] cannot get redeem session for group %s", groupUid)
	}
	redeemTxs := s.ScalarClient.PickCacheRedeemTx(groupUid, redeemSession.Session.Sequence)
	log.Info().Any("redeemTxs", redeemTxs).Msgf("[Relayer] [replayBtcRedeemTxs] redeem txs in cache")
	for chainId, redeemTxs := range redeemTxs {
		err := s.ScalarClient.BroadcastRedeemTxsConfirmRequest(chainId, groupUid, redeemTxs)
		if err != nil {
			return fmt.Errorf("[Relayer] [processRecoverPreparingPhase] cannot broadcast redeem txs confirm request for group %s", groupUid)
		}
	}
	return nil
}

func (s *Service) waitingForPendingCommands(mapTxHashes map[string][]string) {
	wg := sync.WaitGroup{}
	for chainId, txHashes := range mapTxHashes {
		wg.Add(1)
		go func(chainId string, txHashes []string) {
			defer wg.Done()
			err := s.ScalarClient.WaitForPendingCommands(chainId, txHashes)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for pending commands for evm client %s", chainId)
			} else {
				log.Info().Str("ChainId", chainId).Msgf("[Relayer] [processRecoverExecutionPhase] finished waiting for pending commands")
			}
		}(chainId, txHashes)
	}
	wg.Wait()

}
func (s *Service) replaySwitchPhaseEvents(mapSwitchPhaseEvents map[string][]*contracts.IScalarGatewaySwitchPhase, index int) (int32, int32, bool) {
	wg := sync.WaitGroup{}
	var hasDifferentPhase atomic.Bool
	var expectedPhase atomic.Int32
	var evmCounter atomic.Int32
	expectedPhase.Store(-1)

	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			switchPhaseEvents, ok := mapSwitchPhaseEvents[chainId]
			if !ok || len(switchPhaseEvents) == 0 {
				log.Warn().Msgf("[Relayer] [processRecoverPreparingPhase] cannot find redeem session for evm client %s", chainId)
				return
			}
			if index >= len(switchPhaseEvents) {
				log.Warn().Str("chainId", chainId).
					Int("index", index).
					Msgf("[Relayer] [processRecoverPreparingPhase] Switchphase event not found")
				return
			}
			switchPhaseEvent := switchPhaseEvents[index]
			expectedPhaseValue := expectedPhase.Load()
			if expectedPhaseValue == -1 {
				expectedPhase.Store(int32(switchPhaseEvent.To))
			} else if expectedPhaseValue != int32(switchPhaseEvent.To) {
				log.Warn().Msgf("[Relayer] [processRecoverPreparingPhase] found switch phase event with different phase")
				hasDifferentPhase.Store(true)
				return
			}
			err := evmClient.HandleSwitchPhase(switchPhaseEvent)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot handle switch phase event for evm client %s", chainId)
			} else {
				evmCounter.Add(1)
			}
		}()
	}
	wg.Wait()
	return expectedPhase.Load(), evmCounter.Load(), hasDifferentPhase.Load()
}

/*
 * check if the current redeem session is broadcasted to bitcoin network by checking if the first input utxo is present in the bitcoin network
 */
func (s *Service) isRedeemSessionBroadcasted(mapRedeemTokenEvents map[string][]*contracts.IScalarGatewayRedeemToken) (bool, error) {
	log.Info().Msgf("[Relayer] [isRedeemSessionBroadcasted] checking if the current redeem session is broadcasted to bitcoin network")
	if s.BtcClient == nil {
		return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] btc client is undefined")
	}
	var firstRedeemTokenEvent *contracts.IScalarGatewayRedeemToken
	for _, redeemTxs := range mapRedeemTokenEvents {
		if len(redeemTxs) > 0 {
			firstRedeemTokenEvent = redeemTxs[0]
			break
		}
	}
	params := covTypes.RedeemTokenPayloadWithType{}
	err := params.AbiUnpack(firstRedeemTokenEvent.Payload)
	if err != nil {
		return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] cannot unpack redeem token payload: %s", err)
	}
	log.Info().Any("First EvmRedeemTx", params).Msg("[Service] check if redeem session is broadcasted")
	if len(params.Utxos) == 0 {
		return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] no utxos found in redeem token payload")
	}
	for _, btcClient := range s.BtcClient {
		if btcClient.Config().ID == firstRedeemTokenEvent.DestinationChain {
			txId := strings.TrimPrefix(params.Utxos[0].TxID.Hex(), "0x")
			outspend, err := btcClient.GetOutSpend(txId, params.Utxos[0].Vout)
			// outResult, err := btcClient.GetTxOut(params.Utxos[0].TxID.Hex(), params.Utxos[0].Vout)
			if err != nil {
				return false, fmt.Errorf("[Relayer] [isRedeemSessionBroadcasted] cannot get outspend for redeem token event: %s", err)
			}
			log.Info().Any("OutSpend", outspend).Msg("Successfully get outspend from mempool")
			// if outResult == nil {
			// 	return true, nil
			// }
			return outspend.Spent, nil
		}
	}
	return false, nil
}

func (s *Service) replayRedeemTransactions(groupUid string, mapRedeemTokenEvents map[string][]*contracts.IScalarGatewayRedeemToken) (map[string][]string, error) {
	if s.ScalarClient == nil {
		return nil, fmt.Errorf("[Relayer] [processRecoverExecutionPhase] scalar client is undefined")
	}
	//Waiting for utxo snapshot initialied
	err := s.ScalarClient.WaitForUtxoSnapshot(groupUid)
	if err != nil {
		return nil, fmt.Errorf("[Relayer] [processRecoverExecutionPhase] cannot wait for utxo snapshot for group %s", groupUid)
	}
	mapTxHashes := sync.Map{}
	wg := sync.WaitGroup{}
	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			redeemTokenEvents, ok := mapRedeemTokenEvents[chainId]
			if !ok {
				log.Warn().Str("ChainId", chainId).Msgf("[Relayer] [processRecoverExecutionPhase] no redeemToken event for repaylaying")
				return
			}
			// Scalar network will utxoSnapshot request on each confirm RedeemToken event
			for _, redeemTokenEvent := range redeemTokenEvents {
				err := evmClient.HandleRedeemToken(redeemTokenEvent)
				if err != nil {
					log.Warn().
						Str("chainId", chainId).
						Any("redeemTokenEvent", redeemTokenEvent).
						Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot handle redeem token event")
				} else {
					value, loaded := mapTxHashes.LoadOrStore(redeemTokenEvent.DestinationChain, []string{redeemTokenEvent.Raw.TxHash.Hex()})
					if loaded {
						mapTxHashes.Store(redeemTokenEvent.DestinationChain, append(value.([]string), redeemTokenEvent.Raw.TxHash.Hex()))
					}
				}
			}
			log.Info().Str("ChainId", chainId).Int("RedeemTx count", len(redeemTokenEvents)).
				Msgf("[Relayer] [processRecoverExecutionPhase] finished handle redeem token events")
		}()
	}
	wg.Wait()
	result := map[string][]string{}
	mapTxHashes.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.([]string)
		return true
	})
	return result, nil
}
func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
	for _, client := range s.Electrs {
		go client.Electrs.Close()
	}
}
