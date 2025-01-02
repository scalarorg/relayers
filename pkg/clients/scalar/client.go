package scalar

import (
	"context"
	"fmt"
	"math"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/pkg/clients/cosmos"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/scalar-core/utils"
	"github.com/scalarorg/scalar-core/x/chains/types"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	nexus "github.com/scalarorg/scalar-core/x/nexus/exported"

	//tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	txConfig := tx.NewTxConfig(codec.GetProtoCodec(), tx.DefaultSignModes)
	subscriberName := fmt.Sprintf("subscriber-%s", config.ID)
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
	// if config.GrpcAddress != "" {
	// 	log.Info().Msgf("Create Grpc client to address: %s", config.GrpcAddress)
	// 	dialOpts := []grpc.DialOption{
	// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
	// 	}
	// 	grpcConn, err := grpc.NewClient(config.GrpcAddress, dialOpts...)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	// 	}
	queryClient := cosmos.NewQueryClient(clientCtx)
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
	receiver := c.eventBus.Subscribe(events.SCALAR_NETWORK_NAME)
	go func() {
		for event := range receiver {
			go func() {
				err := c.handleEventBusMessage(event)
				if err != nil {
					log.Error().Msgf("[ScalarClient] [Start] error: %v", err)
				}
			}()
		}
	}()
	go func() {
		log.Info().Msg("[ScalarClient] Start ProcessSignCommands process")
		c.ProcessSignCommands(ctx)
	}()
	go func() {
		c.subscribeWithHeatBeat(ctx)
	}()
	log.Info().Msg("Scalar client started")
	return nil
}
func (c *Client) subscribeWithHeatBeat(ctx context.Context) {
	retryInterval := time.Millisecond * time.Duration(c.networkConfig.RetryInterval)
	deadCount := 0
	for {
		cancelCtx, cancelFunc := context.WithCancel(ctx)
		//Start rpc client
		log.Debug().Msg("[ScalarClient] [Start] Try to start scalar connection")
		tmclient, err := c.network.Start()
		if err != nil {
			deadCount += 1
			if deadCount >= 10 {
				log.Debug().Msgf("[ScalarClient] [Start] Connect to the scalar network failed, sleep for %ds then retry", int64(retryInterval.Seconds()))
			}
			c.network.RemoveRpcClient()
			time.Sleep(retryInterval)
			continue
		}
		log.Info().Msgf("[ScalarClient] [Start] Start rpc client success. Subscribing for events...")

		err = subscribeTokenSentEvent(cancelCtx, c.network, c.handleTokenSentEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [subscribeTokenSentEvent] error: %v", err)
		}

		err = subscribeDestCallApprovedEvent(cancelCtx, c.network, c.handleDestCallApprovedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [subscribeDestCallApprovedEvent] error: %v", err)
		}
		// Todo: findout if this event is emitted by the ScalarNetwork
		// err = subscribeSignCommandsEvent(cancelCtx, c.network, c.handleSignCommandsEvents)
		// if err != nil {
		// 	log.Error().Msgf("[ScalarClient] [subscribeSignCommandsEvent] error: %v", err)
		// }
		err = subscribeEVMCompletedEvent(cancelCtx, c.network, c.handleEVMCompletedEvents)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] error: %v", err)
		}
		// err = subscribeAllNewBlockEvent(cancelCtx, c.network, c.handleAnyEvents)
		// if err != nil {
		// 	log.Error().Msgf("[ScalarClient] [subscribeAllEvent] Failed: %v", err)
		// }
		// For debug purpose, subscribe to all tx events, findout if there is sign commands event
		err = subscribeAllTxEvent(cancelCtx, c.network)
		if err != nil {
			log.Error().Msgf("[ScalarClient] [subscribeAllTxEvent] Failed: %v", err)
		}
		//HeatBeat
		aliveCount := 0
		for {
			_, err := tmclient.Health(ctx)
			if err != nil {
				// clean all subscriber then retry
				log.Info().Msgf("[ScalarClient] ScalarNode is dead. Perform reconnecting")
				c.network.RemoveRpcClient()
				break
			} else {
				aliveCount += 1
				if aliveCount >= 100 {
					log.Debug().Msgf("[ScalarClient] ScalarNode is alive")
					aliveCount = 0
				}
			}
			time.Sleep(retryInterval)
		}
		cancelFunc()
	}
}

func subscribeTokenSentEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.EventTokenSent]) error) error {
	if _, err := Subscribe(ctx, network, TokenSentEvent,
		func(events []IBCEvent[*types.EventTokenSent]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [DestCallApprovedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeDestCallApprovedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeDestCallApprovedEvent] success")
	}
	return nil
}

func subscribeDestCallApprovedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.DestCallApproved]) error) error {
	if _, err := Subscribe(ctx, network, DestCallApprovedEvent,
		func(events []IBCEvent[*types.DestCallApproved]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [DestCallApprovedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeDestCallApprovedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeDestCallApprovedEvent] success")
	}
	return nil
}

// func subscribeSignCommandsEvent(ctx context.Context, network *cosmos.NetworkClient,
// 	callback func(ctx context.Context, events []IBCEvent[SignCommands]) error) error {
// 	if _, err := Subscribe(ctx, network, SignCommandsEvent,
// 		func(events []IBCEvent[SignCommands]) {
// 			err := callback(ctx, events)
// 			if err != nil {
// 				log.Error().Msgf("[ScalarClient] [SignCommandsHandler] callback error: %v", err)
// 			}

// 		}); err != nil {
// 		log.Debug().Msgf("[ScalarClient] [subscribeSignCommandsEvent] Failed: %v", err)
// 		return err
// 	} else {
// 		log.Debug().Msgf("[ScalarClient] [subscribeSignCommandsEvent] success")
// 	}
// 	return nil
// }

func subscribeEVMCompletedEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[*types.ChainEventCompleted]) error) error {
	if _, err := Subscribe(ctx, network, EVMCompletedEvent,
		func(events []IBCEvent[*types.ChainEventCompleted]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [EVMCompletedHandler] callback error: %v", err)
			}
		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeEVMCompletedEvent] success")
	}
	return nil
}
func subscribeAllNewBlockEvent(ctx context.Context, network *cosmos.NetworkClient,
	callback func(ctx context.Context, events []IBCEvent[proto.Message]) error) error {
	//Subscribe to all events for debug purpose
	if _, err := Subscribe(ctx, network, AllNewBlockEvent,
		func(events []IBCEvent[proto.Message]) {
			err := callback(ctx, events)
			if err != nil {
				log.Error().Msgf("[ScalarClient] [AllNewBlockHandler] callback error: %v", err)
			}

		}); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllNewBlockEvent] success")
	}
	return nil
}

func subscribeAllTxEvent(ctx context.Context, network *cosmos.NetworkClient) error {
	//Subscribe to all events for debug purpose
	TxEvent := ListenerEvent[proto.Message]{
		Type: "Tx",
		//TopicId: "tm.event='Tx'",
		TopicId: "tm.event='*'",
		Parser: func(events map[string][]string) ([]IBCEvent[proto.Message], error) {
			log.Debug().Msgf("[ScalarClient] [AllTxHandler] events: %v", events)
			return nil, nil
		},
	}
	callback := func(events []IBCEvent[proto.Message]) {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] events: %v", events)
	}
	if _, err := Subscribe(ctx, network, TxEvent, callback); err != nil {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] Failed: %v", err)
		return err
	} else {
		log.Debug().Msgf("[ScalarClient] [subscribeAllTxEvent] success")
	}
	return nil
}

// https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L159
func Subscribe[T proto.Message](ctx context.Context,
	network *cosmos.NetworkClient,
	event ListenerEvent[T],
	callback EventHandlerCallBack[T]) (string, error) {
	eventCh, err := network.Subscribe(ctx, event.Type, event.TopicId)
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to Event: %v, %w", event, err)
	}
	//Handle the event in a separate goroutine
	go func() {
		for {
			select {
			case evt := <-eventCh:
				if evt.Query != event.TopicId {
					log.Debug().Msgf("[ScalarClient] [Subscribe] Event query is not match query: %v, topicId: %s", evt.Query, event.TopicId)
				} else {
					//Extract the data from the event
					log.Debug().Str("Topic", evt.Query).Msg("Received new event")
					// var args T
					// msgType := reflect.TypeOf(args).Elem()
					// msg := reflect.New(msgType).Interface().(T)
					events, err := ParseIBCEvent[T](evt.Events)
					if err != nil {
						log.Debug().Msgf("[ScalarClient] [Subscribe] parser event with query: %v error: %v", evt.Query, err)
					}
					callback(events)

					// fmt.Printf("Result %v", result)
					// //data, err := event.Parser(evt.Events)
					// if err != nil {
					// 	log.Debug().Msgf("[ScalarClient] [Subscribe] parser event with query: %v error: %v", evt.Query, err)
					// } else if len(data) == 0 {
					// 	log.Debug().Msgf("[ScalarClient] [Subscribe] Empty event with query: %v", evt.Query)
					// } else {
					// 	callback(result)
					// }
				}
			case <-ctx.Done():
				log.Debug().Msgf("[ScalarClient] [Subscribe] timed out waiting for event, the transaction could have already been included or wasn't yet included")
				network.UnSubscribeAll(context.Background(), event.Type) //nolint:errcheck // ignore
				return
			}
		}
	}()
	return event.Type, nil
}

func (c *Client) ConfirmTxs(ctx context.Context, chainName string, txIds []string) (*sdk.TxResponse, error) {
	//1. Create Confirm message request
	nexusChain := nexus.ChainName(utils.NormalizeString(chainName))
	log.Debug().Msgf("[ScalarClient] [ConfirmTxs] Broadcast for confirmation txs from chain %s: %v", nexusChain, txIds)
	txHashs := make([]chainstypes.Hash, len(txIds))
	for i, txId := range txIds {
		txHashs[i] = chainstypes.Hash(common.HexToHash(txId))
	}
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, txHashs)
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
	txHash := chainstypes.Hash(common.HexToHash(txId))
	msg := chainstypes.NewConfirmSourceTxsRequest(c.network.GetAddress(), nexusChain, []chainstypes.Hash{txHash})

	//2. Sign and broadcast the payload using the network client, which has the private key
	confirmTx, err := c.network.ConfirmEvmTx(ctx, msg)
	if err != nil {
		return nil, err
	}

	return confirmTx, nil
}

func (c *Client) GetQueryClient() *cosmos.QueryClient {
	return c.queryClient
}
