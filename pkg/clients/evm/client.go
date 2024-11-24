package evm

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
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

func (c *EvmClient) getLastCheckpoint() *models.EventCheckPoint {
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(c.evmConfig.GetId(), events.EVENT_EVM_CONTRACT_CALL)
	if err != nil {
		log.Warn().Str("chainId", c.evmConfig.GetId()).
			Str("eventName", events.EVENT_EVM_CONTRACT_CALL).
			Msg("[EvmClient] [getLastCheckpoint] using default value")
	}
	return lastCheckpoint
}

// Todo: [WIP] try to recover missing events from the last checkpoint block number to the current block number
func watchForEvent(c *EvmClient, eventName string) error {
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
	//This case should not happen
	//It can only happen if the db is set to a future block by debugger
	if lastCheckpoint.BlockNumber > uint64(blockNumber) {
		return nil
	}
	//recover missing events from the last checkpoint block number to the current block number
	//missingEvents := c.Client.GetEvents(lastCheckpoint.BlockNumber, uint64(blockNumber))
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

	watchOpts := bind.WatchOpts{Start: &c.evmConfig.LastBlock, Context: ctx}
	//Listen to the gateway ContractCallEvent
	//This event is initiated by user
	//1. User call protocol's smart contract on the evm
	//2. Protocol sm call Scalar gateway contract for emitting ContractCallEvent
	err := c.watchContractCall(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch ContractCallEvent: %w", err)
	}
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
