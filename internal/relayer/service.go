package relayer

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/adapters/evm"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/spf13/viper"
)

type Service struct {
	EvmAdapter   *evm.EvmAdapter
	BusEventChan chan *types.EventEnvelope
}

func NewService(config *config.Config, eventChan chan *types.EventEnvelope, receiverChanBufSize int) (*Service, error) {
	var err error

	// // Initialize Axelar service
	// err = axelar.InitAxelarAdapter()
	// if err != nil {
	// 	return nil, err
	// }

	// Initialize EVM service
	evmAdapter, err := evm.NewEvmAdapter(config.EvmNetworks, eventChan, receiverChanBufSize)
	if err != nil {
		return nil, err
	}

	// // Initialize BTC service
	// err = btc.NewBtcClients(config.GlobalConfig.BtcNetworks)
	// if err != nil {
	// 	return nil, err
	// }

	// // Initialize RabbitMQ client
	// err = rabbitmq.InitRabbitMQClient()
	// if err != nil {
	// 	return nil, err
	// }

	return &Service{
		EvmAdapter: evmAdapter,
	}, nil
}

func (s *Service) Start() error {
	go func() {
		for event := range s.BusEventChan {
			log.Info().Msgf("Received event from: %+v to Component: %+v", event.SenderClientName, event.Component)
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
