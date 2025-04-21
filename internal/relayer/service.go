package relayer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/btc"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	types "github.com/scalarorg/relayers/pkg/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
)

type Service struct {
	DbAdapter    *db.DatabaseAdapter
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

	// Initialize BTC service
	btcClients, err := btc.NewBtcClients(config, dbAdapter, eventBus, scalarClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create btc clients: %w", err)
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
	//Start btc clients
	for _, client := range s.BtcClient {
		go client.Start(ctx)
	}
	//Start electrum clients. This client can get all vault transactions from last checkpoint of begining if no checkpoint is found
	for _, client := range s.Electrs {
		go client.Start(ctx)
	}
	// Improvement recovery evm missing source events
	// 2025, March 10
	// Recover all swiched phase events from evm networks
	groups, err := s.ScalarClient.GetCovenantGroups(ctx)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot get covenant groups")
		panic(err)
	}
	//Perform recovery redeem session before recover other events
	err = s.RecoverEvmSessions(groups)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover sessions")
		panic(err)
	}
	//Recover all events
	for _, client := range s.EvmClients {
		go client.ProcessMissingLogs()
		go func() {
			//Todo: Handle the moment when recover just finished and listner has not started yet. It around 1 second
			err := client.RecoverAllEvents(ctx, groups)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover events for evm client %s", client.EvmConfig.GetId())
			} else {
				log.Info().Msgf("[Relayer] [Start] recovered missing events for evm client %s", client.EvmConfig.GetId())
				client.Start(ctx)
			}
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
func (s *Service) RecoverEvmSessions(groups []*covExported.CustodianGroup) error {
	wg := sync.WaitGroup{}
	recoverSessions := CustodiansRecoverRedeemSessions{}
	for _, client := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainRedeemSessions, err := client.RecoverRedeemSessions(groups)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [Start] cannot recover sessions for evm client %s", client.EvmConfig.GetId())
			}
			recoverSessions.AddRecoverSessions(client.EvmConfig.GetId(), chainRedeemSessions)

		}()
	}
	wg.Wait()
	recoverSessions.CleanUp()
	for groupUid, groupRedeemSessions := range recoverSessions.RecoverSessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info().Str("groupUid", groupUid).
				Any("redeemSession", groupRedeemSessions).
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
	return nil
}
func (s *Service) processRecoverExecutingPhase(groupUid string, groupRedeemSessions *types.GroupRedeemSessions) error {
	//1. Replay all switch to preparing phase event,
	wg := sync.WaitGroup{}
	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			switchPhaseEvents, ok := groupRedeemSessions.SwitchPhaseEvents[chainId]
			if !ok || len(switchPhaseEvents) == 0 {
				log.Warn().Msgf("[Relayer] [processRecoverExecutionPhase] cannot find redeem session for evm client %s", chainId)
				return
			}
			if switchPhaseEvents[0].To == uint8(covExported.Preparing) {
				err := evmClient.HandleSwitchPhase(switchPhaseEvents[0])
				if err != nil {
					log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot handle switch phase event for evm client %s", chainId)
				}
			}
		}()
	}
	wg.Wait()
	//2. wait for group's session switch to preparing then replay all redeem token events
	err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Preparing)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to preparing phase", groupUid)
		return err
	}
	//3. Wait for utxo snapshot
	err = s.ScalarClient.WaitForUtxoSnapshot(groupUid)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for utxo snapshot for group %s", groupUid)
		return err
	}
	//4. Replay all redeem transactions
	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			redeemTokenEvents, ok := groupRedeemSessions.RedeemTokenEvents[chainId]
			if !ok {
				log.Warn().Msgf("[Relayer] [processRecoverExecutionPhase] cannot find redeem token events for evm client %s", chainId)
				return
			}
			sourceTxs := map[string]bool{}
			//Call reserve utxo request
			for _, redeemTokenEvent := range redeemTokenEvents {
				err := s.ScalarClient.ReserveUtxo(chainId, redeemTokenEvent)
				if err != nil {
					log.Warn().
						Str("chainId", chainId).
						Any("redeemTokenEvent", redeemTokenEvent).
						Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] broadcast reserve utxo requests")
				} else {
					sourceTxs[redeemTokenEvent.Raw.TxHash.String()] = true
				}
			}

			for _, redeemTokenEvent := range redeemTokenEvents {
				err := evmClient.HandleRedeemToken(redeemTokenEvent)
				if err != nil {
					log.Warn().
						Str("chainId", chainId).
						Any("redeemTokenEvent", redeemTokenEvent).
						Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot handle redeem token event")
				} else {
					sourceTxs[redeemTokenEvent.Raw.TxHash.String()] = true
				}
			}
			//Wait for pending command
			err = s.ScalarClient.WaitForPendingCommands(chainId, sourceTxs)
			if err != nil {
				log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for pending commands for evm client %s", chainId)
			}
		}()
	}
	wg.Wait()
	//4. Replay all switch to executing phase events
	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			switchPhaseEvents, ok := groupRedeemSessions.SwitchPhaseEvents[chainId]
			if !ok || len(switchPhaseEvents) < 2 {
				log.Warn().Str("ChainId", chainId).Msgf("[Relayer] [processRecoverExecutionPhase] SwitchPhase to Executing is not found")
				return
			}
			if switchPhaseEvents[1].To == uint8(covExported.Executing) {
				err := evmClient.HandleSwitchPhase(switchPhaseEvents[1])
				if err != nil {
					log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot handle switch phase event for evm client %s", chainId)
				}
			}
		}()
	}
	wg.Wait()
	// 5. wait for group's session switch to executing then replay all switch phase events,
	// Maybe some evm dit not switch to executing phase yet.

	// err = s.ScalarClient.WaitForSwitchingToPhase(groupUid, covExported.Executing)
	// if err != nil {
	// 	log.Warn().Err(err).Msgf("[Relayer] [processRecoverExecutionPhase] cannot wait for group %s to switch to executing phase", groupUid)
	// 	return err
	// }
	return nil
}
func (s *Service) processRecoverPreparingPhase(groupUid string, groupRedeemSessions *types.GroupRedeemSessions) error {
	//1. For each evm chain, replay last switch event. It can be Preparing or executing from previous session
	wg := sync.WaitGroup{}
	var chainsInExecuting sync.Map
	var hasChainInExecuting atomic.Bool
	for _, evmClient := range s.EvmClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainId := evmClient.EvmConfig.GetId()
			switchPhaseEvents, ok := groupRedeemSessions.SwitchPhaseEvents[chainId]
			if !ok || len(switchPhaseEvents) == 0 {
				log.Warn().Msgf("[Relayer] [processRecoverPreparingPhase] cannot find redeem session for evm client %s", chainId)
				return
			}
			if len(switchPhaseEvents) == 1 {
				if switchPhaseEvents[0].To == uint8(covExported.Preparing) {
					chainsInExecuting.Store(chainId, true)
					hasChainInExecuting.Store(true)
				}
				err := evmClient.HandleSwitchPhase(switchPhaseEvents[0])
				if err != nil {
					log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot handle switch phase event for evm client %s", chainId)
				}
			} else if len(switchPhaseEvents) == 2 {
				log.Warn().Str("ChainId", chainId).Any("switchPhaseEvents", switchPhaseEvents).
					Msgf("[Relayer] [processRecoverPreparingPhase] found 2 switch phase events. Something not works as expected")
			}
		}()
	}
	wg.Wait()
	expectedPhase := covExported.Preparing
	if hasChainInExecuting.Load() {
		expectedPhase = covExported.Executing
	}
	//2. Waiting for group session switch to expected phase
	err := s.ScalarClient.WaitForSwitchingToPhase(groupUid, expectedPhase)
	if err != nil {
		log.Warn().Err(err).Msgf("[Relayer] [processRecoverPreparingPhase] cannot wait for group %s to switch to executing phase", groupUid)
		return err
	}
	//3. Wait for group redeem transaction confirmed from btc
	return nil
}

func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
}
