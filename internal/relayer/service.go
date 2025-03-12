package relayer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/btc"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
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
	for _, client := range s.EvmClients {
		go client.ProcessMissingLogs()
		go func() {
			//Todo: Handle the moment when recover just finished and listner has not started yet. It around 1 second
			err := client.RecoverAllEvents(ctx)
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

func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
}
