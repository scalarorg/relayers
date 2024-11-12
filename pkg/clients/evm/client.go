package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

const COMPONENT_NAME = "EvmClient"

type EvmClient struct {
	Client                  *ethclient.Client
	ChainName               string
	GatewayAddress          common.Address
	Gateway                 *contracts.IAxelarGateway
	auth                    *bind.TransactOpts
	config                  EvmNetworkConfig
	dbAdapter               *db.DatabaseAdapter
	eventBus                *events.EventBus
	subContractCall         event.Subscription
	subContractCallApproved event.Subscription
	subExecuted             event.Subscription
}

func NewEvmClient(config EvmNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*EvmClient, error) {
	// Setup
	ctx := context.Background()

	// Connect to a test network
	rpc, err := rpc.DialContext(ctx, config.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM network %s: %w", config.Name, err)
	}

	client := ethclient.NewClient(rpc)

	// Initialize the contract
	gatewayAddress := common.HexToAddress(config.Gateway)

	gateway, err := contracts.NewIAxelarGateway(gatewayAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", config.Name, err)
	}

	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key for network %s: %w", config.Name, err)
	}
	chainID, ok := new(big.Int).SetString(config.ChainID, 10)
	if !ok {
		return nil, fmt.Errorf("invalid chain ID for network %s", config.Name)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth for network %s: %w", config.Name, err)
	}

	evmClient := &EvmClient{
		Client:         client,
		ChainName:      config.Name,
		GatewayAddress: gatewayAddress,
		Gateway:        gateway,
		auth:           auth,
		config:         config,
	}

	return evmClient, nil
}

func NewEvmClients(configPath string, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) ([]*EvmClient, error) {
	evmCfgPath := fmt.Sprintf("%s/evm.json", configPath)
	configs, err := config.ReadJsonArrayConfig[EvmNetworkConfig](evmCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read electrum configs: %w", err)
	}
	//Inject evm private keys
	for i := range configs {
		preparePrivateKey(&configs[i])
	}
	evmClients := make([]*EvmClient, 0, len(configs))
	for _, config := range configs {
		client, err := NewEvmClient(config, dbAdapter, eventBus)
		if err != nil {
			return nil, err
		}
		evmClients = append(evmClients, client)
	}

	return evmClients, nil
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

func (c *EvmClient) watchContractCall(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayContractCall)

	subContractCall, err := c.Gateway.WatchContractCall(watchOpts, sink, nil, nil)
	if err != nil {
		return err
	}
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
	defer subContractCall.Unsubscribe()
	return nil
}

func (c *EvmClient) watchContractCallApproved(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayContractCallApproved)
	subContractCallApproved, err := c.Gateway.WatchContractCallApproved(watchOpts, sink, nil, nil, nil)
	if err != nil {
		return err
	}
	go func() {
		for event := range sink {
			log.Info().Msgf("Contract call approved: %v", event)
			c.handleContractCallApproved(event)
		}

	}()
	c.subContractCallApproved = subContractCallApproved
	defer subContractCallApproved.Unsubscribe()
	return nil
}

func (c *EvmClient) watchEVMExecuted(watchOpts *bind.WatchOpts) error {
	sink := make(chan *contracts.IAxelarGatewayExecuted)
	subExecuted, err := c.Gateway.WatchExecuted(watchOpts, sink, nil)
	if err != nil {
		return err
	}
	go func() {
		for event := range sink {
			log.Info().Msgf("EVM executed: %v", event)
			c.handleCommandExecuted(event)
		}
	}()
	c.subExecuted = subExecuted
	defer subExecuted.Unsubscribe()
	return nil
}

func (c *EvmClient) Start(ctx context.Context) error {
	watchOpts := bind.WatchOpts{Start: &c.config.LastBlock, Context: ctx}
	//Listen to the gateway ContractCallEvent
	//This event is initiated by user
	//1. User call protocol's smart contract on the evm
	//2. Protocol sm call Scalar gateway contract for emitting ContractCallEvent
	err := c.watchContractCall(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch ContractCallEvent: %w", err)
	}
	//Listen to the gateway ContractCallApprovedEvent
	err = c.watchContractCallApproved(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch ContractCallApprovedEvent: %w", err)
	}
	//Listen to the gateway EVMExecutedEvent
	err = c.watchEVMExecuted(&watchOpts)
	if err != nil {
		return fmt.Errorf("failed to watch EVMExecutedEvent: %w", err)
	}
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
