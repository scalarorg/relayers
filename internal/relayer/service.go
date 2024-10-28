package relayer

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/services/axelar"
	"github.com/scalarorg/relayers/pkg/services/rabbitmq"
)

type Service struct {
	// other fields...
}

func NewService() (*Service, error) {
	var err error

	// Initialize Axelar service
	err = axelar.InitAxelarService()
	if err != nil {
		return nil, err
	}

	// Initialize RabbitMQ client
	err = rabbitmq.InitRabbitMQClient()
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
