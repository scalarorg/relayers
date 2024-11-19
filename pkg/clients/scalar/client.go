package scalar

import (
	"context"
	"fmt"
	"math"

	"github.com/axelarnetwork/axelar-core/utils"
	emvtypes "github.com/axelarnetwork/axelar-core/x/evm/types"
	nexus "github.com/axelarnetwork/axelar-core/x/nexus/exported"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"

	//tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
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
	scalarCfgPath := fmt.Sprintf("%s/scalar.json", globalConfig.ConfigPath)
	scalarConfig, err := config.ReadJsonConfig[cosmos.CosmosNetworkConfig](scalarCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scalar config from file: %s, %w", scalarCfgPath, err)
	}
	if globalConfig.ScalarMnemonic == "" {
		return nil, fmt.Errorf("scalar mnemonic is not set")
	}
	scalarConfig.Mnemonic = globalConfig.ScalarMnemonic
	//Set default max retries is 3
	if scalarConfig.MaxRetries == 0 {
		scalarConfig.MaxRetries = 3
	}
	//Set default retry interval is 1000ms
	if scalarConfig.RetryInterval == 0 {
		scalarConfig.RetryInterval = 1000
	}
	return NewClientFromConfig(globalConfig, scalarConfig, dbAdapter, eventBus)
}

func NewClientFromConfig(globalConfig *config.Config, config *cosmos.CosmosNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*Client, error) {
	txConfig := tx.NewTxConfig(codec.GetProtoCodec(), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})
	subscriberName := fmt.Sprintf("subscriber-%s", config.ChainID)
	//Set default broadcast mode is sync
	if config.BroadcastMode == "" {
		config.BroadcastMode = "sync"
	}
	//Check if float gas price is 0, set default value is 0.0125
	if math.Abs(config.GasPrice*1000) < 0.000001 {
		config.GasPrice = 0.0125
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
	receiver := c.eventBus.Subscribe(SCALAR_NETWORK_NAME)
	go func() {
		for event := range receiver {
			err := c.handleEventBusMessage(event)
			if err != nil {
				log.Error().Msgf("Failed to handle event bus message: %v", err)
			}
		}
	}()
	//Start rpc client
	if err := c.network.Start(); err != nil {
		return fmt.Errorf("failed to start rpc client: %w", err)
	} else {
		log.Debug().Msgf("Start rpc client success")
	}
	if _, err := Subscribe(ctx, c.network, ContractCallApprovedEvent,
		func(event *IBCEvent[ContractCallApproved], err error) {
			if err != nil {
				log.Error().Msgf("Failed to handle received event ContractCallApprovedEvent: %v", err)
			}
			go c.handleContractCallApprovedEvent(ctx, event)
		}); err != nil {
		log.Debug().Msgf("Failed to subscribe to ContractCallApprovedEvent: %v", err)
	} else {
		log.Debug().Msgf("Subscribe to ContractCallApprovedEvent success")
	}

	if _, err := Subscribe(ctx, c.network, EVMCompletedEvent,
		func(event *IBCEvent[EVMEventCompleted], err error) {
			if err != nil {
				log.Error().Msgf("Failed to handle received event EVMCompletedEvent: %v", err)
			}
			go c.handleEVMCompletedEvent(ctx, event)
		}); err != nil {
		log.Debug().Msgf("Failed to subscribe to EVMCompletedEvent: %v", err)
	} else {
		log.Debug().Msgf("Subscribe to EVMCompletedEvent success")
	}

	return nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L159
func Subscribe[T any](ctx context.Context,
	network *cosmos.NetworkClient,
	event ListenerEvent[T],
	callback EventHandlerCallBack[T]) (string, error) {
	eventCh, err := network.Subscribe(ctx, event.Type, event.TopicId)
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to Event: %v, %w", event, err)
	}
	defer network.UnSubscribeAll(context.Background(), event.Type) //nolint:errcheck // ignore
	select {
	case evt := <-eventCh:
		log.Debug().Msgf("Received event: %v", evt)
		if evt.Query != event.TopicId {
			callback(nil, fmt.Errorf("event query is not match"))
		} else {
			//Extract the data from the event
			data, err := event.Parser(evt.Events)
			if err != nil {
				callback(nil, err)
			} else {
				callback(data, nil)
			}
		}
	case <-ctx.Done():
		return "", errors.ErrLogic.Wrapf("timed out waiting for event, the transaction could have already been included or wasn't yet included")
	}
	return event.Type, nil
}

func (c *Client) ConfirmTxs(ctx context.Context, chainName string, txIds []string) (*sdk.TxResponse, error) {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[ScalarClient] [ConfirmTxs] Broadcast for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]emvtypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = emvtypes.Hash(common.HexToHash(txId))
	}
	msg := emvtypes.NewConfirmGatewayTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
	//2. Sign and broadcast the payload using the network client, which has the private key
	confirmTx, err := c.network.SignAndBroadcastMsgs(ctx, msg)
	if err != nil {
		log.Error().Msgf("[ScalarClient] [ConfirmTxs] error from network client: %v", err)
		return nil, err
	}
	if confirmTx != nil && confirmTx.Code != 0 {
		log.Error().Msgf("[ScalarClient] [ConfirmTxs] error from network client: %v", confirmTx.RawLog)
		return nil, fmt.Errorf("error from network client: %v", confirmTx.RawLog)
	} else {
		log.Debug().Msgf("[ScalarClient] [ConfirmTxs] success broadcast confirmation txs with tx hash: %s", confirmTx.TxHash)
		return confirmTx, nil
	}
}

func (c *Client) ConfirmBtcTx(ctx context.Context, chainName string, txId string) (*sdk.TxResponse, error) {
	return c.ConfirmEvmTx(ctx, chainName, txId)
}

// Relayer call this function for request Scalar network to confirm the transaction on the source chain
func (c *Client) ConfirmEvmTx(ctx context.Context, chainName string, txId string) (*sdk.TxResponse, error) {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	txHash := emvtypes.Hash(common.HexToHash(txId))
	msg := emvtypes.NewConfirmGatewayTxsRequest(c.network.GetAddress(), nexusChain, []emvtypes.Hash{txHash})

	//2. Sign and broadcast the payload using the network client, which has the private key
	confirmTx, err := c.network.ConfirmEvmTx(ctx, msg)
	if err != nil {
		return nil, err
	}

	return confirmTx, nil
}
