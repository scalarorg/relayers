package evm_clients

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
)

type EvmClient struct {
	Client         *ethclient.Client
	ChainName      string
	GatewayAddress common.Address
	Gateway        *contracts.IAxelarGateway
	auth           *bind.TransactOpts
	config         config.EvmNetworkConfig
}

func NewEvmClient(config config.EvmNetworkConfig) (*EvmClient, error) {
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

func NewEvmClients(configs []config.EvmNetworkConfig) ([]*EvmClient, error) {
	evmClients := make([]*EvmClient, 0, len(configs))
	for _, config := range configs {
		client, err := NewEvmClient(config)
		if err != nil {
			return nil, err
		}
		evmClients = append(evmClients, client)
	}

	return evmClients, nil
}
