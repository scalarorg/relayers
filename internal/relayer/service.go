package relayer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/client/axelar"
	"github.com/scalarorg/relayers/pkg/client/rabbitmq"
)

type Service struct {
	// other fields...
}

func NewService() (*Service, error) {
	var err error

	err = rabbitmq.InitRabbitMQClient()
	if err != nil {
		return nil, err
	}

	// Initialize Axelar client
	err = axelar.InitAxelarClient()
	if err != nil {
		return nil, err
	}

	return &Service{
		// ... initialize other fields if needed ...
	}, nil
}

func (s *Service) Start() error {
	return rabbitmq.RabbitMQClient.Consume()
}

func (s *Service) Stop() {
	rabbitmq.RabbitMQClient.Close()
	log.Info().Msg("Relayer service stopped")
}

func (s *Service) Run(ctx context.Context) error {
	// Implementation...
	return nil
}
