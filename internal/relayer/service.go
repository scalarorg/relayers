package relayer

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/spf13/viper"
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
	scalarClient, err := scalar.NewClient(config.ConfigPath)
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

func (s *Service) Start() error {
	go func() {
		for event := range s.BusEventChan {
			log.Info().Msgf("Received event: %v", event)
			if event.Component == "DbAdapter" {
				db.DbAdapter.BusEventReceiverChan <- event
			} else if event.Component == "EvmAdapter" {
				s.EvmAdapter.BusEventReceiverChan <- event
			}
		}
	}()

	go db.DbAdapter.ListenEventsFromBusChannel()

	go s.EvmAdapter.ListenEventsFromBusChannel()

	s.EvmAdapter.PollEventsFromAllEvmNetworks(time.Duration(viper.GetInt("POLL_INTERVAL")) * time.Second)

	return nil
}

func (s *Service) Stop() {
	log.Info().Msg("Relayer service stopped")
}
