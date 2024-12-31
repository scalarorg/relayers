package evm

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/clients/evm/pending"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
)

type EvmClient struct {
	globalConfig            *config.Config
	evmConfig               *EvmNetworkConfig
	Client                  *ethclient.Client
	ChainName               string
	GatewayAddress          common.Address
	Gateway                 *contracts.IScalarGateway
	auth                    *bind.TransactOpts
	dbAdapter               *db.DatabaseAdapter
	eventBus                *events.EventBus
	pendingTxs              pending.PendingTxs //Transactions sent to Gateway for approval, waiting for event from EVM chain.
	retryInterval           time.Duration
	subContractCall         event.Subscription
	subContractCallApproved event.Subscription
	subExecuted             event.Subscription
	subTokenSent            event.Subscription
}

// This function is used to adjust the rpc url to the ws prefix
// format: ws:// -> http://
// format: wss:// -> https://
// Todo: Improve this implementation

func adjustRpcUrl(rpcUrl string) string {
	if strings.HasPrefix(rpcUrl, "http") {
		return strings.Replace(rpcUrl, "http", "ws", 1)
	}
	return rpcUrl
}

func NewEvmClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) ([]*EvmClient, error) {
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
			evmConfig.GasLimit = 300000
		}
		client, err := NewEvmClient(globalConfig, &evmConfig, dbAdapter, eventBus)
		if err != nil {
			log.Warn().Msgf("Failed to create evm client for %s: %v", evmConfig.GetName(), err)
			continue
		}
		globalConfig.AddChainConfig(config.IChainConfig(&evmConfig))
		evmClients = append(evmClients, client)
	}

	return evmClients, nil
}

func NewEvmClient(globalConfig *config.Config, evmConfig *EvmNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*EvmClient, error) {
	// Setup
	ctx := context.Background()

	// Connect to a test network
	rpcUrl := adjustRpcUrl(evmConfig.RPCUrl)
	rpc, err := rpc.DialContext(ctx, rpcUrl)
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
		evmConfig:      evmConfig,
		Client:         client,
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
		err = c.RecoverMissingEvents(ctx)
		if err != nil {
			log.Printf("Error recovering missing events: %v", err)
		}
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
func (c *EvmClient) RecoverMissingEvents(ctx context.Context) error {
	//Recover ContractCall events
	if err := RecoverEvent[*contracts.IScalarGatewayContractCall](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL, func(log types.Log) *contracts.IScalarGatewayContractCall {
			return &contracts.IScalarGatewayContractCall{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL_APPROVED).Msg("failed to recover missing events")
		return err
	}
	if err := RecoverEvent[*contracts.IScalarGatewayContractCallApproved](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL_APPROVED, func(log types.Log) *contracts.IScalarGatewayContractCallApproved {
			return &contracts.IScalarGatewayContractCallApproved{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL_APPROVED).Msg("failed to recover missing events")
		return err
	}
	if err := RecoverEvent[*contracts.IScalarGatewayTokenSent](c, ctx,
		events.EVENT_EVM_TOKEN_SENT, func(log types.Log) *contracts.IScalarGatewayTokenSent {
			return &contracts.IScalarGatewayTokenSent{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_TOKEN_SENT).Msg("failed to recover missing events")
		return err
	}
	if err := RecoverEvent[*contracts.IScalarGatewayExecuted](c, ctx,
		events.EVENT_EVM_COMMAND_EXECUTED, func(log types.Log) *contracts.IScalarGatewayExecuted {
			return &contracts.IScalarGatewayExecuted{
				Raw: log,
			}
		}); err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL_APPROVED).Msg("failed to recover missing events")
		return err
	}
	return nil
}

// Try to recover missing events from the last checkpoint block number to the current block number
func RecoverEvent[T ValidEvmEvent](c *EvmClient, ctx context.Context, eventName string, fnCreateEventData func(types.Log) T) error {
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(c.evmConfig.GetId(), eventName)
	if err != nil {
		log.Warn().Str("chainId", c.evmConfig.GetId()).
			Str("eventName", eventName).
			Msg("[EvmClient] [getLastCheckpoint] using default value")
	}
	//Get current block number
	blockNumber, err := c.Client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	log.Info().Uint64("Current BlockNumber", blockNumber).Msg("[EvmClient] [RecoverThenWatchForEvent]")

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
			err := c.handleEvent(event.Args)
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
func GetMissingEvents[T ValidEvmEvent](c *EvmClient, eventName string, lastCheckpoint *models.EventCheckPoint, fnCreateEventData func(types.Log) T) ([]*parser.EvmEvent[T], error) {
	event, ok := scalarGatewayAbi.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event %s not found", eventName)
	}

	// Set up a query for logs
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
	//Watch for new ContractCall events in one go routine
	var err error
	if c.subContractCall, err = watchForEvent(c, ctx, events.EVENT_EVM_CONTRACT_CALL); err != nil {
		return fmt.Errorf("failed to watch for event: %w", err)
	}
	//Watch for new ContractCallApproved events in one go routine
	if c.subContractCallApproved, err = watchForEvent(c, ctx, events.EVENT_EVM_CONTRACT_CALL_APPROVED); err != nil {
		return fmt.Errorf("failed to watch for event: %w", err)
	}
	//Watch for new Executed events in one go routine
	if c.subExecuted, err = watchForEvent(c, ctx, events.EVENT_EVM_COMMAND_EXECUTED); err != nil {
		return fmt.Errorf("failed to watch for event: %w", err)
	}
	//Watch for new TokenSent events in one go routine
	if c.subTokenSent, err = watchForEvent(c, ctx, events.EVENT_EVM_TOKEN_SENT); err != nil {
		return fmt.Errorf("failed to watch for event: %w", err)
	}
	c.retryInterval = RETRY_INTERVAL
	//Wait for context cancel
	<-ctx.Done()
	return nil
}

func (c *EvmClient) handleEvent(event any) error {
	switch e := event.(type) {
	case *contracts.IScalarGatewayContractCall:
		return c.handleContractCall(e)
	case *contracts.IScalarGatewayContractCallApproved:
		return c.HandleContractCallApproved(e)
	case *contracts.IScalarGatewayExecuted:
		return c.HandleCommandExecuted(e)
	case *contracts.IScalarGatewayTokenSent:
		return c.HandleTokenSent(e)
	}
	return nil
}
func watchForEvent(c *EvmClient, ctx context.Context, eventName string) (event.Subscription, error) {
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(c.evmConfig.GetId(), eventName)
	if err != nil {
		log.Warn().Str("chainId", c.evmConfig.GetId()).
			Str("eventName", eventName).
			Msg("[EvmClient] [getLastCheckpoint] using default value")
	}
	watchOpts := bind.WatchOpts{Start: &lastCheckpoint.BlockNumber, Context: ctx}
	log.Info().Str("eventName", eventName).Msg("[EvmClient] [watchForEvent] started watching for new events")
	switch eventName {
	case events.EVENT_EVM_CONTRACT_CALL:
		return c.watchContractCall(&watchOpts)
	case events.EVENT_EVM_CONTRACT_CALL_APPROVED:
		return c.watchContractCallApproved(&watchOpts)
	case events.EVENT_EVM_COMMAND_EXECUTED:
		return c.watchEVMExecuted(&watchOpts)
	case events.EVENT_EVM_TOKEN_SENT:
		return c.watchEVMTokenSent(&watchOpts)
	}
	return nil, nil
}
func (c *EvmClient) watchContractCall(watchOpts *bind.WatchOpts) (event.Subscription, error) {
	sink := make(chan *contracts.IScalarGatewayContractCall)
	errorCh := make(chan error)
	subContractCall, err := c.Gateway.WatchContractCall(watchOpts, sink, nil, nil)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("[EvmClient] [watchContractCall] success. Listening to ContractCallEvent")
	go func() {
		// timeout := time.After(30 * time.Minute)
		for {
			select {
			case err := <-errorCh:
				log.Error().Msgf("Failed to watch ContractCallEvent: %v", err)
			case event := <-sink:
				log.Info().Msg("[EvmClient] [watchContractCall] receved contract call")
				err := c.handleContractCall(event)
				if err != nil {
					log.Error().Msgf("Failed to handle ContractCallEvent: %v", err)
				}
			// case <-timeout:
			// 	log.Error().Msgf("[EvmClient] [watchContractCall] timeout")
			// 	return
			case <-watchOpts.Context.Done():
				log.Info().Msgf("[EvmClient] [watchContractCall] context done")
				return
			}
		}
	}()
	return subContractCall, nil
}

func (c *EvmClient) watchContractCallApproved(watchOpts *bind.WatchOpts) (event.Subscription, error) {
	sink := make(chan *contracts.IScalarGatewayContractCallApproved)
	subContractCallApproved, err := c.Gateway.WatchContractCallApproved(watchOpts, sink, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("[EvmClient] [watchContractCallApproved] success. Listening to ContractCallApprovedEvent")
	go func() {
		for event := range sink {
			log.Info().Any("event", event).Msg("[EvmClient] [ContractCallApprovedHandler]")
			c.HandleContractCallApproved(event)
		}

	}()
	return subContractCallApproved, nil
}

func (c *EvmClient) watchEVMExecuted(watchOpts *bind.WatchOpts) (event.Subscription, error) {
	sink := make(chan *contracts.IScalarGatewayExecuted)
	subExecuted, err := c.Gateway.WatchExecuted(watchOpts, sink, nil)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("[EvmClient] [watchEVMExecuted] success. Listening to ExecutedEvent")
	go func() {
		for event := range sink {
			log.Info().Any("event", event).Msgf("EvmClient] [ExecutedHandler]")
			c.HandleCommandExecuted(event)
		}
	}()
	return subExecuted, nil
}
func (c *EvmClient) watchEVMTokenSent(watchOpts *bind.WatchOpts) (event.Subscription, error) {
	sink := make(chan *contracts.IScalarGatewayTokenSent)
	subTokenSent, err := c.Gateway.WatchTokenSent(watchOpts, sink, nil)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("[EvmClient] [watchEVMTokenSent] success. Listening to TokenSent")
	go func() {
		for event := range sink {
			err := c.HandleTokenSent(event)
			if err != nil {
				log.Debug().Err(err).Msgf("[EvmClient] [watchEVMTokenSent] with error.")
			} else {
				log.Info().Any("event", event).Msgf("EvmClient] [watchEVMTokenSent]")
			}
		}
	}()
	return subTokenSent, nil
}
func (c *EvmClient) subscribeEventBus() {
	if c.eventBus != nil {
		log.Debug().Msgf("[EvmClient] [Start] subscribe to the event bus %s", c.evmConfig.GetId())
		receiver := c.eventBus.Subscribe(c.evmConfig.GetId())
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

func (c *EvmClient) Stop() {
	if c.subContractCall != nil {
		c.subContractCall.Unsubscribe()
	}
	if c.subContractCallApproved != nil {
		c.subContractCallApproved.Unsubscribe()
	}
	if c.subExecuted != nil {
		c.subExecuted.Unsubscribe()
	}
	c.Client.Close()
}
