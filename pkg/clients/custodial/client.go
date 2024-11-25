package custodial

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type Client struct {
	globalConfig   *config.Config
	networkConfig  *cosmos.CosmosNetworkConfig
	txConfig       client.TxConfig
	network        *cosmos.NetworkClient
	queryClient    *cosmos.QueryClient
	dbAdapter      *db.DatabaseAdapter
	eventBus       *events.EventBus
	subscriberName string //Use as subscriber for networkClient
	// Add other necessary fields like chain ID, gas prices, etc.
}

func NewClient(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*Client, error) {
	// Read Scalar config from JSON file
	custodialCfgPath := fmt.Sprintf("%s/custodial.json", globalConfig.ConfigPath)
	custodialConfig, err := config.ReadJsonConfig[cosmos.CosmosNetworkConfig](custodialCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read custodial config from file: %s, %w", custodialCfgPath, err)
	}
	if globalConfig.ScalarMnemonic == "" {
		return nil, fmt.Errorf("scalar mnemonic is not set")
	}
	custodialConfig.Mnemonic = globalConfig.ScalarMnemonic
	//Set default max retries is 3
	if custodialConfig.MaxRetries == 0 {
		custodialConfig.MaxRetries = 3
	}
	//Set default retry interval is 1000ms
	if custodialConfig.RetryInterval == 0 {
		custodialConfig.RetryInterval = 1000
	}
	return NewClientFromConfig(globalConfig, custodialConfig, dbAdapter, eventBus)
}

func NewClientFromConfig(globalConfig *config.Config, config *cosmos.CosmosNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*Client, error) {
	txConfig := tx.NewTxConfig(codec.GetProtoCodec(), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	subscriberName := fmt.Sprintf("subscriber-%d", config.ChainID)
	//Set default broadcast mode is sync
	if config.BroadcastMode == "" {
		config.BroadcastMode = "sync"
	}
	clientCtx, err := cosmos.CreateClientContext(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client context: %w", err)
	}
	var queryClient *cosmos.QueryClient
	// if config.GrpcAddress != "" {
	// 	log.Info().Msgf("Create Grpc client to address: %s", config.GrpcAddress)
	// 	dialOpts := []grpc.DialOption{
	// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
	// 	}
	// 	grpcConn, err := grpc.NewClient(config.GrpcAddress, dialOpts...)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	// 	}
	queryClient = cosmos.NewQueryClient(clientCtx)
	networkClient, err := cosmos.NewNetworkClient(config, queryClient, txConfig)
	if err != nil {
		return nil, err
	}
	client := &Client{
		globalConfig:   globalConfig,
		networkConfig:  config,
		txConfig:       txConfig,
		network:        networkClient,
		queryClient:    queryClient,
		subscriberName: subscriberName,
		dbAdapter:      dbAdapter,
		eventBus:       eventBus,
	}
	return client, nil
}

func (c *Client) Start(ctx context.Context) error {
	receiver := c.eventBus.Subscribe(events.CUSTODIAL_NETWORK_NAME)
	go func() {
		for event := range receiver {
			err := c.handleEventBusMessage(event)
			if err != nil {
				log.Error().Msgf("Failed to handle event bus message: %v", err)
			}
		}
	}()
	//Start rpc client
	return nil
}
