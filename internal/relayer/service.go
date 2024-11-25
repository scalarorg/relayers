package relayer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/btc"
	"github.com/scalarorg/relayers/pkg/clients/custodial"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type Service struct {
	DbAdapter       *db.DatabaseAdapter
	EventBus        *events.EventBus
	ScalarClient    *scalar.Client
	CustodialClient *custodial.Client
	Electrs         []*electrs.Client
	EvmClients      []*evm.EvmClient
	BtcClient       []*btc.BtcClient
}

func NewService(config *config.Config, dbAdapter *db.DatabaseAdapter,
	eventBus *events.EventBus) (*Service, error) {
	var err error

	// Initialize Scalar client
	scalarClient, err := scalar.NewClient(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create scalar client: %w", err)
	}
	// Initialize Custodial client
	custodialClient, err := custodial.NewClient(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create custodial client: %w", err)
	}
	// Initialize Electrs clients
	electrsClients, err := electrs.NewElectrumClients(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create electrum clients: %w", err)
	}
	// Initialize EVM clients
	evmClients, err := evm.NewEvmClients(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm clients: %w", err)
	}

	// Initialize BTC service
	btcClients, err := btc.NewBtcClients(config, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create btc clients: %w", err)
	}

	return &Service{
		DbAdapter:       dbAdapter,
		EventBus:        eventBus,
		ScalarClient:    scalarClient,
		CustodialClient: custodialClient,
		Electrs:         electrsClients,
		EvmClients:      evmClients,
		BtcClient:       btcClients,
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
	if s.CustodialClient != nil {
		go func() {
			err := s.CustodialClient.Start(ctx)
			if err != nil {
				log.Error().Msgf("Start custodial client with error %+v", err)
			}
		}()
	} else {
		log.Warn().Msg("[Relayer] [Start] custodial client is undefined")
	}
	//Start electrum clients
	for _, client := range s.Electrs {
		go client.Start(ctx)
	}
	//Start evm clients
	for _, client := range s.EvmClients {
		go client.Start(ctx)
	}
	//Start btc clients
	for _, client := range s.BtcClient {
		go client.Start(ctx)
	}
	return nil
}

func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
}
