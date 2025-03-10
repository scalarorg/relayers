package evm

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/clients/evm/pending"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type EvmClient struct {
	globalConfig   *config.Config
	EvmConfig      *EvmNetworkConfig
	Client         *ethclient.Client
	ScalarClient   *scalar.Client
	ChainName      string
	GatewayAddress common.Address
	Gateway        *contracts.IScalarGateway
	auth           *bind.TransactOpts
	dbAdapter      *db.DatabaseAdapter
	eventBus       *events.EventBus
	pendingTxs     pending.PendingTxs //Transactions sent to Gateway for approval, waiting for event from EVM chain.
	retryInterval  time.Duration
}

// This function is used to adjust the rpc url to the ws prefix
// format: ws:// -> http://
// format: wss:// -> https://
// Todo: Improve this implementation

func NewEvmClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) ([]*EvmClient, error) {
	if globalConfig == nil || globalConfig.ConfigPath == "" {
		return nil, fmt.Errorf("config path is not set")
	}
	evmCfgPath := fmt.Sprintf("%s/evm.json", globalConfig.ConfigPath)
	configs, err := config.ReadJsonArrayConfig[EvmNetworkConfig](evmCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read electrum configs: %w", err)
	}

	evmClients := make([]*EvmClient, 0, len(configs))
	for _, evmConfig := range configs {
		//Inject evm private keys
		preparePrivateKey(&evmConfig)
		//Set default value for block time if is not set
		if evmConfig.BlockTime == 0 {
			evmConfig.BlockTime = 12 * time.Second
		} else {
			evmConfig.BlockTime = evmConfig.BlockTime * time.Millisecond
		}
		//Set default gaslimit to 300000
		if evmConfig.GasLimit == 0 {
			evmConfig.GasLimit = 3000000
		}
		if evmConfig.MaxRecoverRange == 0 {
			evmConfig.MaxRecoverRange = 64000
		}
		client, err := NewEvmClient(globalConfig, &evmConfig, dbAdapter, eventBus, scalarClient)
		if err != nil {
			log.Warn().Msgf("Failed to create evm client for %s: %v", evmConfig.GetName(), err)
			continue
		}
		globalConfig.AddChainConfig(config.IChainConfig(&evmConfig))
		evmClients = append(evmClients, client)
	}

	return evmClients, nil
}

func NewEvmClient(globalConfig *config.Config, evmConfig *EvmNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) (*EvmClient, error) {
	// Setup
	ctx := context.Background()
	log.Info().Any("evmConfig", evmConfig).Msgf("[EvmClient] [NewEvmClient] connecting to EVM network")
	// Connect to a test network
	rpc, err := rpc.DialContext(ctx, evmConfig.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM network %s: %w", evmConfig.Name, err)
	}
	client := ethclient.NewClient(rpc)
	gateway, gatewayAddress, err := CreateGateway(evmConfig.Name, evmConfig.Gateway, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway for network %s: %w", evmConfig.Name, err)
	}
	auth, err := CreateEvmAuth(evmConfig)
	if err != nil {
		//Not fatal, we can still use the gateway without auth
		//auth is only used for sending transaction
		log.Warn().Msgf("[EvmClient] [NewEvmClient] failed to create auth for network %s: %v", evmConfig.Name, err)
	}
	evmClient := &EvmClient{
		globalConfig:   globalConfig,
		EvmConfig:      evmConfig,
		Client:         client,
		ScalarClient:   scalarClient,
		GatewayAddress: *gatewayAddress,
		Gateway:        gateway,
		auth:           auth,
		dbAdapter:      dbAdapter,
		eventBus:       eventBus,
		pendingTxs:     pending.PendingTxs{},
		retryInterval:  RETRY_INTERVAL,
	}

	return evmClient, nil
}
func CreateGateway(networName string, gwAddr string, client *ethclient.Client) (*contracts.IScalarGateway, *common.Address, error) {
	if gwAddr == "" {
		return nil, nil, fmt.Errorf("gateway address is not set for network %s", networName)
	}
	gatewayAddress := common.HexToAddress(gwAddr)
	gateway, err := contracts.NewIScalarGateway(gatewayAddress, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", networName, err)
	}
	return gateway, &gatewayAddress, nil
}
func CreateEvmAuth(evmConfig *EvmNetworkConfig) (*bind.TransactOpts, error) {
	if evmConfig.PrivateKey == "" {
		return nil, fmt.Errorf("private key is not set for network %s", evmConfig.Name)
	}
	privateKey, err := crypto.HexToECDSA(evmConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key for network %s: %w", evmConfig.Name, err)
	}
	chainID := big.NewInt(int64(evmConfig.ChainID))
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth for network %s: %w", evmConfig.Name, err)
	}
	auth.GasLimit = evmConfig.GasLimit
	return auth, nil
}

func preparePrivateKey(evmCfg *EvmNetworkConfig) error {
	if evmCfg.PrivateKey == "" {
		if config.GlobalConfig.EvmPrivateKey == "" {
			return fmt.Errorf("private key is not set")
		}
		evmCfg.PrivateKey = config.GlobalConfig.EvmPrivateKey
		// if evmCfg.Mnemonic != nil || evmCfg.WalletIndex != nil {
		// 	wallet, err := hdwallet.NewFromMnemonic(*networkConfig.Mnemonic)
		// 	if err != nil {
		// 		return "", fmt.Errorf("failed to create wallet from mnemonic: %w", err)
		// 	}

		// 	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%s", *networkConfig.WalletIndex))
		// 	account, err := wallet.Derive(path, true)
		// 	if err != nil {
		// 		return "", fmt.Errorf("failed to derive account: %w", err)
		// 	}

		// 	privateKeyECDSA, err := wallet.PrivateKey(account)
		// 	if err != nil {
		// 		return "", fmt.Errorf("failed to get private key: %w", err)
		// 	}

		// 	privateKeyBytes := crypto.FromECDSA(privateKeyECDSA)
		// 	privateKey = hex.EncodeToString(privateKeyBytes)
		// 	return fmt.Errorf("private key and mnemonic/wallet index cannot be set at the same time")
		// }
	}
	return nil
}
func (c *EvmClient) SetAuth(auth *bind.TransactOpts) {
	c.auth = auth
}

func (c *EvmClient) Start(ctx context.Context) error {
	//Subscribe to the event bus
	c.subscribeEventBus()
	c.ConnectWithRetry(ctx)
	return fmt.Errorf("context cancelled")
}

func (c *EvmClient) ConnectWithRetry(ctx context.Context) {
	var retryInterval = time.Second * 12 // Initial retry interval
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	for {
		// Directly call recovery method in the service before starting listeners
		// err = c.RecoverInitiatedEvents(ctx)
		// if err != nil {
		// 	log.Printf("Error recovering missing events: %v", err)
		// }
		// err = c.RecoverApprovedEvents(ctx)
		// if err != nil {
		// 	log.Printf("Error recovering missing events: %v", err)
		// }
		// err = c.RecoverExecutedEvents(ctx)
		// if err != nil {
		// 	log.Printf("Error recovering missing events: %v", err)
		// }
		//Listen to new events
		err = c.ListenToEvents(ctx)
		if err != nil {
			log.Printf("Error listening to events: %v", err)
		}

		// If context is cancelled, stop retrying
		if ctx.Err() != nil {
			log.Warn().Msg("Context cancelled, stopping listener")
			return
		}

		// Wait before retrying
		log.Printf("Reconnecting in %v...\n", retryInterval)
		time.Sleep(retryInterval)

		// Exponential backoff with cap
		if retryInterval < time.Minute {
			retryInterval *= 2
		}
		//Wait for context cancel
		<-ctx.Done()
	}
}
func (c *EvmClient) VerifyDeployTokens(ctx context.Context) error {
	return nil
}

/*
 * Recover initiated events
 */
func (c *EvmClient) RecoverInitiatedEvents(ctx context.Context) error {
	//Recover ContractCall events
	if err := RecoverEvent[*contracts.IScalarGatewayContractCall](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL, func(log types.Log) *contracts.IScalarGatewayContractCall {
			return &contracts.IScalarGatewayContractCall{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL).Msg("failed to recover missing events")
	}
	if err := RecoverEvent[*contracts.IScalarGatewayContractCallWithToken](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN, func(log types.Log) *contracts.IScalarGatewayContractCallWithToken {
			return &contracts.IScalarGatewayContractCallWithToken{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN).Msg("failed to recover missing events")
	}
	if err := RecoverEvent[*contracts.IScalarGatewayTokenSent](c, ctx,
		events.EVENT_EVM_TOKEN_SENT, func(log types.Log) *contracts.IScalarGatewayTokenSent {
			return &contracts.IScalarGatewayTokenSent{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_TOKEN_SENT).Msg("failed to recover missing events")
	}
	return nil
}
func (c *EvmClient) RecoverApprovedEvents(ctx context.Context) error {
	err := RecoverEvent[*contracts.IScalarGatewayContractCallApproved](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED, func(log types.Log) *contracts.IScalarGatewayContractCallApproved {
			return &contracts.IScalarGatewayContractCallApproved{
				Raw: log,
			}
		})
	if err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL_APPROVED).Msg("failed to recover missing events")
	}
	return err
}

func (c *EvmClient) RecoverExecutedEvents(ctx context.Context) error {
	err := RecoverEvent[*contracts.IScalarGatewayExecuted](c, ctx,
		events.EVENT_EVM_COMMAND_EXECUTED, func(log types.Log) *contracts.IScalarGatewayExecuted {
			return &contracts.IScalarGatewayExecuted{
				Raw: log,
			}
		})
	if err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_COMMAND_EXECUTED).Msg("failed to recover missing events")
	}
	return err
}

// Try to recover missing events from the last checkpoint block number to the current block number
func RecoverEvent[T ValidEvmEvent](c *EvmClient, ctx context.Context, eventName string, fnCreateEventData func(types.Log) T) error {
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(c.EvmConfig.GetId(), eventName, c.EvmConfig.LastBlock)
	if err != nil {
		log.Warn().Str("chainId", c.EvmConfig.GetId()).
			Str("eventName", eventName).
			Msg("[EvmClient] [getLastCheckpoint] using default value")
	}
	//Get current block number
	blockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Str("Chain", c.EvmConfig.ID).Uint64("Current BlockNumber", blockNumber).Msg("[EvmClient] [RecoverThenWatchForEvent]")
	if blockNumber-lastCheckpoint.BlockNumber > c.EvmConfig.MaxRecoverRange {
		lastCheckpoint.BlockNumber = blockNumber - c.EvmConfig.MaxRecoverRange + 1 //We add extra 1 to make sure we don't cover over configured range
	}
	//recover missing events from the last checkpoint block number to the current block number
	// We already filtered received event (defined by block and logIndex)
	// So if no more missing events, then function GetMissionEvents returns an empty array

	for {
		missingEvents, err := GetMissingEvents[T](c, eventName, lastCheckpoint, fnCreateEventData)
		if err != nil {
			return fmt.Errorf("failed to get missing events: %w", err)
		}
		if len(missingEvents) == 0 {
			log.Info().Str("eventName", eventName).Msg("[EvmClient] [RecoverEvent] no more missing events")
			break
		}
		//Process the missing events
		for _, event := range missingEvents {
			log.Debug().Str("eventName", eventName).
				Str("txHash", event.Hash).
				Msg("[EvmClient] [RecoverEvent] start handling missing event")
			err := handleEvent(c, eventName, event.Args)
			//Update the last checkpoint value for next iteration
			lastCheckpoint.BlockNumber = event.BlockNumber
			lastCheckpoint.LogIndex = event.LogIndex
			lastCheckpoint.TxHash = event.Hash
			//If handleEvent success, the last checkpoint is updated within the function
			//So we need to update the last checkpoint only if handleEvent failed
			if err != nil {
				log.Error().Err(err).Msg("[EvmClient] [RecoverEvent] failed to handle event")
			}
			//Store the last checkpoint value into db,
			//this can be performed only once when we finish recover, but some thing can break recover process, so we store state imediatele
			err = c.dbAdapter.UpdateLastEventCheckPoint(lastCheckpoint)
			if err != nil {
				log.Error().Err(err).Msg("[EvmClient] [RecoverEvent] update last checkpoint failed")
			}

		}
		//Try to get more missing events in the next iteration
	}
	return nil
}

// Get missing events from the last checkpoint block number to the current block number
// In query we filter out the event with index equal to the last checkpoint log index
func GetMissingEvents[T ValidEvmEvent](c *EvmClient, eventName string, lastCheckpoint *scalarnet.EventCheckPoint, fnCreateEventData func(types.Log) T) ([]*parser.EvmEvent[T], error) {
	event, ok := scalarGatewayAbi.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event %s not found", eventName)
	}

	// Set up a query for logs
	// Todo: config default last checkpoint block number to make sure we don't recover too old events
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(lastCheckpoint.BlockNumber)),
		Addresses: []common.Address{c.GatewayAddress},
		Topics:    [][]common.Hash{{event.ID}}, // Filter by event signature
	}
	// // Fetch the logs
	logs, err := c.Client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	result := []*parser.EvmEvent[T]{}
	// Parse the logs
	for _, receiptLog := range logs {
		if receiptLog.BlockNumber < lastCheckpoint.BlockNumber ||
			(receiptLog.BlockNumber == lastCheckpoint.BlockNumber && receiptLog.Index <= lastCheckpoint.LogIndex) {
			log.Info().Uint64("receiptLogBlockNumber", receiptLog.BlockNumber).
				Uint("receiptLogIndex", receiptLog.Index).
				Msg("[EvmClient] [GetMissingEvents] skip log")
			continue
		}
		var eventData = fnCreateEventData(receiptLog)
		err := parser.ParseEventData(&receiptLog, eventName, eventData)
		if err == nil {
			evmEvent := parser.CreateEvmEventFromArgs[T](eventData, &receiptLog)
			result = append(result, evmEvent)
			log.Info().
				Str("eventName", eventName).
				Uint64("blockNumber", evmEvent.BlockNumber).
				Str("txHash", evmEvent.Hash).
				Uint("txIndex", evmEvent.TxIndex).
				Uint("logIndex", evmEvent.LogIndex).
				Msg("[EvmClient] [GetMissingEvents] parsing successfully.")
		} else {
			log.Error().Err(err).Msg("[EvmClient] [GetMissingEvents] failed to unpack log data")
		}
	}
	log.Info().Int("eventCount", len(logs)).
		Str("eventName", eventName).
		Any("query", query).
		Any("lastCheckpoint", lastCheckpoint).
		Any("missingEventsCount", len(result)).
		Msg("[EvmClient] [GetMissingEvents]")
	return result, nil
}

func (c *EvmClient) ListenToEvents(ctx context.Context) error {
	c.retryInterval = RETRY_INTERVAL

	events := []struct {
		name  string
		watch func(context.Context) error
	}{
		{events.EVENT_EVM_TOKEN_SENT, func(ctx context.Context) error {
			return WatchForEvent[*contracts.IScalarGatewayTokenSent](c, ctx, events.EVENT_EVM_TOKEN_SENT)
		}},
		{events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN, func(ctx context.Context) error {
			return WatchForEvent[*contracts.IScalarGatewayContractCallWithToken](c, ctx, events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN)
		}},
		{events.EVENT_EVM_CONTRACT_CALL_APPROVED, func(ctx context.Context) error {
			return WatchForEvent[*contracts.IScalarGatewayContractCallApproved](c, ctx, events.EVENT_EVM_CONTRACT_CALL_APPROVED)
		}},
		{events.EVENT_EVM_COMMAND_EXECUTED, func(ctx context.Context) error {
			return WatchForEvent[*contracts.IScalarGatewayExecuted](c, ctx, events.EVENT_EVM_COMMAND_EXECUTED)
		}},
	}

	for _, event := range events {
		go func(e struct {
			name  string
			watch func(context.Context) error
		}) {
			if err := e.watch(ctx); err != nil {
				log.Error().Err(err).Any("Config", c.EvmConfig).Msgf("[EvmClient] [ListenToEvents] failed to watch for event: %s", e.name)
			}
		}(event)
	}

	<-ctx.Done()
	return nil
}

type ValidWatchEvent interface {
	*contracts.IScalarGatewayTokenSent |
		*contracts.IScalarGatewayContractCallWithToken |
		// *contracts.IScalarGatewayContractCall |
		*contracts.IScalarGatewayContractCallApproved |
		*contracts.IScalarGatewayExecuted
}

const (
	baseDelay    = 5 * time.Second
	maxDelay     = 2 * time.Minute
	maxAttempts  = math.MaxUint64
	jitterFactor = 0.2 // 20% jitter
)

func WatchForEvent[T ValidWatchEvent](c *EvmClient, ctx context.Context, eventName string) error {

	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(c.EvmConfig.GetId(), eventName, c.EvmConfig.LastBlock)
	if err != nil {
		log.Warn().Str("chainId", c.EvmConfig.GetId()).
			Str("eventName", eventName).
			Msg("[EvmClient] [getLastCheckpoint] using default value")
	}

	watchOpts := bind.WatchOpts{Start: &lastCheckpoint.BlockNumber, Context: ctx}

	sink := make(chan T)

	subscription, err := setupSubscription(c, &watchOpts, sink, eventName)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	log.Info().Msgf("[EvmClient] [watchEVMTokenSent] success. Listening to %s", eventName)

	for {
		select {
		case err := <-subscription.Err():
			log.Error().Err(err).Msgf("[EvmClient] [WatchForEvent] error with subscription for %s, attempting reconnect", eventName)

			subscription, err = reconnectWithBackoff(c, &watchOpts, sink, eventName)
			if err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}

		case <-watchOpts.Context.Done():
			return nil

		case event := <-sink:
			err := handleEvent(c, eventName, event)
			if err != nil {
				log.Error().Err(err).Msgf("[EvmClient] [WatchForEvent] error handling %s event", eventName)
			} else {
				log.Info().Any("event", event).Msgf("[EvmClient] [WatchForEvent] handled %s event", eventName)
			}
		}
	}
}

func reconnectWithBackoff[T ValidWatchEvent](c *EvmClient, watchOpts *bind.WatchOpts, sink chan T, eventName string) (ethereum.Subscription, error) {
	delay := baseDelay
	for attempt := uint64(1); attempt <= maxAttempts; attempt++ {
		jitter := time.Duration(rand.Float64() * float64(delay) * jitterFactor)
		retryDelay := delay + jitter

		log.Info().
			Uint64("attempt", attempt).
			Dur("delay", retryDelay).
			Msgf("[EvmClient] [WatchForEvent] attempting reconnection for %s", eventName)

		select {
		case <-watchOpts.Context.Done():
			return nil, watchOpts.Context.Err()
		case <-time.After(retryDelay):
			subscription, err := setupSubscription(c, watchOpts, sink, eventName)
			if err == nil {
				log.Info().Msgf("[EvmClient] [WatchForEvent] successfully reconnected for %s", eventName)
				return subscription, nil
			}

			log.Error().
				Err(err).
				Uint64("attempt", attempt).
				Msgf("[EvmClient] [WatchForEvent] reconnection failed for %s", eventName)

			delay = time.Duration(float64(delay) * 2)
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	return nil, fmt.Errorf("failed to reconnect after %d attempts", uint64(maxAttempts))
}

func setupSubscription[T ValidWatchEvent](c *EvmClient, watchOpts *bind.WatchOpts, sink chan T, eventName string) (ethereum.Subscription, error) {
	sinkInterface := any(sink)

	switch eventName {
	case events.EVENT_EVM_TOKEN_SENT:
		return c.Gateway.WatchTokenSent(watchOpts, sinkInterface.(chan *contracts.IScalarGatewayTokenSent), nil)
	case events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN:
		return c.Gateway.WatchContractCallWithToken(watchOpts, sinkInterface.(chan *contracts.IScalarGatewayContractCallWithToken), nil, nil)
	case events.EVENT_EVM_CONTRACT_CALL_APPROVED:
		return c.Gateway.WatchContractCallApproved(watchOpts, sinkInterface.(chan *contracts.IScalarGatewayContractCallApproved), nil, nil, nil)
	case events.EVENT_EVM_COMMAND_EXECUTED:
		return c.Gateway.WatchExecuted(watchOpts, sinkInterface.(chan *contracts.IScalarGatewayExecuted), nil)
	default:
		return nil, fmt.Errorf("[EvmClient] [setupSubscription] unsupported event type for %s, event: %v", eventName, (*T)(nil))
	}
}

func handleEvent(c *EvmClient, eventName string, event any) error {
	switch eventName {
	case events.EVENT_EVM_TOKEN_SENT:
		if evt, ok := event.(*contracts.IScalarGatewayTokenSent); ok {
			return c.HandleTokenSent(evt)
		}
		return fmt.Errorf("cannot parse event %s: %T to %T", eventName, event, (*contracts.IScalarGatewayTokenSent)(nil))
	case events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN:
		if evt, ok := event.(*contracts.IScalarGatewayContractCallWithToken); ok {
			return c.HandleContractCallWithToken(evt)
		}
		return fmt.Errorf("cannot parse event %s: %T to %T", eventName, event, (*contracts.IScalarGatewayContractCallWithToken)(nil))
	case events.EVENT_EVM_CONTRACT_CALL_APPROVED:
		if evt, ok := event.(*contracts.IScalarGatewayContractCallApproved); ok {
			return c.HandleContractCallApproved(evt)
		}
		return fmt.Errorf("cannot parse event %s: %T to %T", eventName, event, (*contracts.IScalarGatewayContractCallApproved)(nil))
	case events.EVENT_EVM_COMMAND_EXECUTED:
		if evt, ok := event.(*contracts.IScalarGatewayExecuted); ok {
			return c.HandleCommandExecuted(evt)
		}
		return fmt.Errorf("cannot parse event %s: %T to %T", eventName, event, (*contracts.IScalarGatewayExecuted)(nil))
	}
	return fmt.Errorf("invalid event type for %s: %T", eventName, event)
}

func (c *EvmClient) subscribeEventBus() {
	if c.eventBus != nil {
		log.Debug().Msgf("[EvmClient] [Start] subscribe to the event bus %s", c.EvmConfig.GetId())
		receiver := c.eventBus.Subscribe(c.EvmConfig.GetId())
		go func() {
			for event := range receiver {
				err := c.handleEventBusMessage(event)
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [EventBusHandler]")
				}
			}
		}()
	} else {
		log.Warn().Msgf("[EvmClient] [subscribeEventBus] event bus is not set")
	}
}
