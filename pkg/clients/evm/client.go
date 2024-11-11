package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
)

type EvmClient struct {
	Client         *ethclient.Client
	ChainName      string
	GatewayAddress common.Address
	Gateway        *contracts.IAxelarGateway
	auth           *bind.TransactOpts
	config         EvmNetworkConfig
	dbAdapter      db.DatabaseAdapter
	eventBus       *events.EventBus
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
		privateKey, err := config.GetEvmPrivateKey(configs[i].ID)
		if err != nil {
			return nil, err
		}
		configs[i].PrivateKey = privateKey
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
