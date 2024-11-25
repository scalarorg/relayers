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
	Gateway                 *contracts.IAxelarGateway
	auth                    *bind.TransactOpts
	dbAdapter               *db.DatabaseAdapter
	eventBus                *events.EventBus
	pendingTxs              pending.PendingTxs //Transactions sent to Gateway for approval, waiting for event from EVM chain.
	subContractCall         event.Subscription
	subContractCallApproved event.Subscription
	subExecuted             event.Subscription
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
	gateway, gatewayAddress, err := CreateGateway(evmConfig, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway for network %s: %w", evmConfig.Name, err)
	}
	auth, err := createEvmAuth(evmConfig)
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
	}

	return evmClient, nil
}
func CreateGateway(evmConfig *EvmNetworkConfig, client *ethclient.Client) (*contracts.IAxelarGateway, *common.Address, error) {
	if evmConfig.Gateway == "" {
		return nil, nil, fmt.Errorf("gateway address is not set for network %s", evmConfig.Name)
	}
	gatewayAddress := common.HexToAddress(evmConfig.Gateway)
	gateway, err := contracts.NewIAxelarGateway(gatewayAddress, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", evmConfig.Name, err)
	}
	return gateway, &gatewayAddress, nil
}
func createEvmAuth(evmConfig *EvmNetworkConfig) (*bind.TransactOpts, error) {
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

// Todo: [WIP] try to recover missing events from the last checkpoint block number to the current block number
func RecoverThenWatchForEvent[T ValidEvmEvent](c *EvmClient, ctx context.Context, eventName string, fnCreateEventData func(types.Log) T) error {
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
	//This case should not happen
	//It can only happen if the db is set to a future block by debugger
	if lastCheckpoint.BlockNumber > uint64(blockNumber) {
		return nil
	}

	//recover missing events from the last checkpoint block number to the current block number
	for {
		missingEvents, err := GetMissingEvents[T](c, eventName, lastCheckpoint, fnCreateEventData)
		if err != nil {
			return fmt.Errorf("failed to get missing events: %w", err)
		}
		if len(missingEvents) > 0 {
			//Process the missing events
			for _, event := range missingEvents {
				log.Debug().Str("eventName", eventName).
					Str("txHash", event.Hash).
					Msg("[EvmClient] [RecoverThenWatchForEvent] start handling missing event")
				err := c.handleEvent(event.Args)
				//Update the last checkpoint value for next iteration
				lastCheckpoint.BlockNumber = blockNumber
				lastCheckpoint.TxIndex = event.TxIndex
				lastCheckpoint.TxHash = event.Hash
				//If handleEvent success, the last checkpoint is updated within the function
				//So we need to update the last checkpoint only if handleEvent failed
				if err != nil {
					log.Error().Err(err).Msg("[EvmClient] [RecoverThenWatchForEvent] failed to handle event")
					err = c.dbAdapter.UpdateLastEventCheckPoint(lastCheckpoint)
					if err != nil {
						log.Error().Err(err).Msg("[EvmClient] [RecoverThenWatchForEvent] update last checkpoint failed")
					}
				}
			}
		} else {
			//Watch for new events
			log.Info().Msg("[EvmClient] [watchForEvent] no missing events")
			break
		}
	}
	watchOpts := bind.WatchOpts{Start: &lastCheckpoint.BlockNumber, Context: ctx}
	//Watch for new event
	err = watchForEvent(c, eventName, &watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch for event: %w", err)
	}
	return nil
}
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
	log.Info().Any("query", query).Msg("[EvmClient] [GetMissingEvents]")
	// // Fetch the logs
	logs, err := c.Client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	log.Info().Int("eventCount", len(logs)).Msg("[EvmClient] [GetMissingEvents]")
	result := []*parser.EvmEvent[T]{}
	// Parse the logs
	for _, receiptLog := range logs {
		if receiptLog.BlockNumber < lastCheckpoint.BlockNumber ||
			(receiptLog.BlockNumber == lastCheckpoint.BlockNumber && receiptLog.TxIndex < lastCheckpoint.TxIndex) {
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
	return result, nil
}
func (c *EvmClient) handleEvent(event any) error {
	switch e := event.(type) {
	case *contracts.IAxelarGatewayContractCall:
		return c.handleContractCall(e)
	case *contracts.IAxelarGatewayContractCallApproved:
		return c.HandleContractCallApproved(e)
	case *contracts.IAxelarGatewayExecuted:
		return c.HandleCommandExecuted(e)
	}
	return nil
}
func watchForEvent(c *EvmClient, eventName string, watchOpts *bind.WatchOpts) error {
	log.Info().Str("eventName", eventName).Msg("[EvmClient] [watchForEvent] started watching for new events")
	switch eventName {
	case events.EVENT_EVM_CONTRACT_CALL:
		return c.watchContractCall(watchOpts)
	case events.EVENT_EVM_CONTRACT_CALL_APPROVED:
		return c.watchContractCallApproved(watchOpts)
	case events.EVENT_EVM_COMMAND_EXECUTED:
		return c.watchEVMExecuted(watchOpts)
	}
	return nil
}
func (c *EvmClient) watchContractCall(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayContractCall)

	subContractCall, err := c.Gateway.WatchContractCall(watchOpts, sink, nil, nil)
	if err != nil {
		return err
	}
	log.Info().Msgf("[EvmClient] [watchContractCall] success. Listening to ContractCallEvent")
	go func() {
		for event := range sink {
			log.Info().Msgf("Contract call: %v", event)
			err := c.handleContractCall(event)
			if err != nil {
				log.Error().Msgf("Failed to handle ContractCallEvent: %v", err)
			}
		}
	}()
	c.subContractCall = subContractCall
	//defer subContractCall.Unsubscribe()
	return nil
}

func (c *EvmClient) watchContractCallApproved(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayContractCallApproved)
	subContractCallApproved, err := c.Gateway.WatchContractCallApproved(watchOpts, sink, nil, nil, nil)
	if err != nil {
		return err
	}
	log.Info().Msgf("[EvmClient] [watchContractCallApproved] success. Listening to ContractCallApprovedEvent")
	go func() {
		for event := range sink {
			log.Info().Any("event", event).Msg("[EvmClient] [ContractCallApprovedHandler]")
			c.HandleContractCallApproved(event)
		}

	}()
	c.subContractCallApproved = subContractCallApproved
	// defer subContractCallApproved.Unsubscribe()
	return nil
}

func (c *EvmClient) watchEVMExecuted(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayExecuted)
	subExecuted, err := c.Gateway.WatchExecuted(watchOpts, sink, nil)
	if err != nil {
		return err
	}
	log.Info().Msgf("[EvmClient] [watchEVMExecuted] success. Listening to ExecutedEvent")
	go func() {
		for event := range sink {
			log.Info().Any("event", event).Msgf("EvmClient] [ExecutedHandler]")
			c.HandleCommandExecuted(event)
		}
	}()
	c.subExecuted = subExecuted
	// defer subExecuted.Unsubscribe()
	return nil
}

func (c *EvmClient) Start(ctx context.Context) error {
	//Subscribe to the event bus
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
	}
	var err error
	err = RecoverThenWatchForEvent[*contracts.IAxelarGatewayContractCall](c, ctx,
		events.EVENT_EVM_CONTRACT_CALL, func(log types.Log) *contracts.IAxelarGatewayContractCall {
			return &contracts.IAxelarGatewayContractCall{
				Raw: log,
			}
		})
	if err != nil {
		log.Error().Err(err).Str("eventName", events.EVENT_EVM_CONTRACT_CALL).Msg("failed to watch event")
	}

	watchOpts := bind.WatchOpts{Start: &c.evmConfig.LastBlock, Context: ctx}
	//Listen to the gateway ContractCallEvent
	//This event is initiated by user
	//1. User call protocol's smart contract on the evm
	//2. Protocol sm call Scalar gateway contract for emitting ContractCallEvent
	// err = c.watchContractCall(&watchOpts)
	// if err != nil {
	// 	return fmt.Errorf("failed to watch ContractCallEvent: %w", err)
	// }
	// Listen to the gateway ContractCallApprovedEvent
	// This event is emitted by the ScalarGateway contract when the executeData is broadcast to the Gateway by call method
	// ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	// Received this event relayer find payload from the db and call the execute method on the protocol's smart contract
	err = c.watchContractCallApproved(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch ContractCallApprovedEvent: %w", err)
	}
	// Listen to the gateway ExecutedEvent
	// This event is emitted by the gateway contract when the executeData if broadcast to the Gateway by call method
	// ec.Gateway.Execute(ec.auth, decodedExecuteData.Input)
	// Receiverd this event, the relayer store the executed data to the db for scanner
	err = c.watchEVMExecuted(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch EVMExecutedEvent: %w", err)
	}
	// Watch pending transactions, call to the evm network to check if the transaction is included in a block
	c.WatchPendingTxs()
	return nil
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
