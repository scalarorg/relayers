package relayer

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type Service struct {
	DbAdapter    *db.DatabaseAdapter
	EventBus     *events.EventBus
	Electrs      []*electrs.Client
	EvmClients   []*evm.EvmClient
	ScalarClient *scalar.Client
}

func NewService(config *config.Config, dbAdapter *db.DatabaseAdapter,
	eventBus *events.EventBus) (*Service, error) {
	var err error

	// Initialize Scalar client
	scalarClient, err := scalar.NewClient(config.ConfigPath, dbAdapter, eventBus)
	if err != nil {
		return nil, err
	}
	// Initialize Electrs clients
	electrsClients, err := electrs.NewElectrumClients(config.ConfigPath, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create electrum clients: %w", err)
	}
	// Initialize EVM clients
	evmClients, err := evm.NewEvmClients(config.ConfigPath, dbAdapter, eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create evm clients: %w", err)
	}

	// // Initialize BTC service
	// err = btc.NewBtcClients(config.GlobalConfig.BtcNetworks)
	// if err != nil {
	// 	return nil, err
	// }

	return &Service{
		DbAdapter:    dbAdapter,
		EventBus:     eventBus,
		Electrs:      electrsClients,
		EvmClients:   evmClients,
		ScalarClient: scalarClient,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {

	//Start electrum clients
	for _, client := range s.Electrs {
		go client.Start(ctx)
	}
	//Start evm clients
	for _, client := range s.EvmClients {
		go client.Start(ctx)
	}
	//Start scalar client
	go s.ScalarClient.Start(ctx)
	return nil
}

func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
}
