package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/spf13/viper"
)

type EvmClient struct {
	client         *ethclient.Client
	chainName      string
	finalityBlocks int
	gatewayAddress common.Address
	gateway        *contracts.IAxelarGateway
	maxRetry       int
	retryDelay     time.Duration
	privateKey     *ecdsa.PrivateKey
	auth           *bind.TransactOpts
}

func NewEvmClients() ([]*EvmClient, error) {
	if len(config.GlobalConfig.EvmNetworks) == 0 {
		return nil, fmt.Errorf("no EVM networks configured")
	}

	clients := make([]*EvmClient, 0, len(config.GlobalConfig.EvmNetworks))

	for _, network := range config.GlobalConfig.EvmNetworks {
		client, err := ethclient.Dial(network.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to EVM network %s: %w", network.Name, err)
		}

		privateKey, err := crypto.HexToECDSA(network.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key for network %s: %w", network.Name, err)
		}

		chainID, ok := new(big.Int).SetString(network.ChainID, 10)
		if !ok {
			return nil, fmt.Errorf("invalid chain ID for network %s", network.Name)
		}

		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create auth for network %s: %w", network.Name, err)
		}

		gatewayAddr := common.HexToAddress(network.Gateway)
		if gatewayAddr == (common.Address{}) {
			return nil, fmt.Errorf("invalid gateway address for network %s", network.Name)
		}

		gateway, err := contracts.NewIAxelarGateway(gatewayAddr, client)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", network.Name, err)
		}

		evmClient := &EvmClient{
			client:         client,
			chainName:      network.Name,
			finalityBlocks: network.Finality,
			gatewayAddress: gatewayAddr,
			gateway:        gateway,
			maxRetry:       config.GlobalConfig.MaxRetry,
			retryDelay:     time.Duration(config.GlobalConfig.RetryDelay) * time.Millisecond,
			privateKey:     privateKey,
			auth:           auth,
		}
		clients = append(clients, evmClient)
	}

	return clients, nil
}

func (c *EvmClient) ChainName() string {
	return c.chainName
}

func (c *EvmClient) GetSenderAddress() string {
	return c.auth.From.Hex()
}

func (c *EvmClient) WaitForFinality(txHash string) (*types.Receipt, error) {
	ctx := context.Background()
	hash := common.HexToHash(txHash)
	return bind.WaitMined(ctx, c.client, &hash)
}

func (c *EvmClient) GatewayExecute(executeData []byte) (*types.Transaction, error) {
	ctx := context.Background()

	// Get latest nonce
	nonce, err := c.client.PendingNonceAt(ctx, c.auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		c.gatewayAddress,
		big.NewInt(0),                       // No ETH value being sent
		uint64(viper.GetInt64("GAS_LIMIT")), // Cast int64 to uint64
		gasPrice,
		executeData,
	)

	log.Debug().Msgf("[EvmClient.gatewayExecute] to: %s, data: %x, gasLimit: %d, gasPrice: %s",
		tx.To().Hex(), tx.Data(), tx.Gas(), gasPrice.String())

	// Sign the transaction
	chainID, err := c.client.NetworkID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Error().Msgf("[EvmClient.gatewayExecute] Failed: %v", err)
		return nil, err
	}

	return signedTx, nil
}

func (c *EvmClient) IsExecuted(commandId [32]byte) (bool, error) {
	return c.gateway.IsCommandExecuted(nil, commandId)
}

func (c *EvmClient) IsCallContractExecuted(
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	contractAddress common.Address,
	payloadHash [32]byte,
) (bool, error) {
	return c.gateway.IsContractCallApproved(nil, commandId, sourceChain, sourceAddress,
		contractAddress, payloadHash)
}

func (c *EvmClient) IsCallContractWithTokenExecuted(
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	contractAddress common.Address,
	payloadHash [32]byte,
	symbol string,
	amount *big.Int,
) (bool, error) {
	return c.gateway.IsContractCallAndMintApproved(nil, commandId, sourceChain, sourceAddress,
		contractAddress, payloadHash, symbol, amount)
}

func (c *EvmClient) Execute(
	contractAddress common.Address,
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	payload []byte,
) (*types.Transaction, error) {
	executable, err := contracts.NewIAxelarExecutable(contractAddress, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create executable contract: %w", err)
	}

	tx, err := executable.Execute(c.auth, commandId, sourceChain, sourceAddress, payload)
	if err != nil {
		log.Error().Msgf("[EvmClient.Execute] Failed: %v", err)
		return nil, err
	}

	return tx, nil
}

func (c *EvmClient) ExecuteWithToken(
	contractAddress common.Address,
	commandId [32]byte,
	sourceChain string,
	sourceAddress string,
	payload []byte,
	symbol string,
	amount *big.Int,
) (*types.Transaction, error) {
	executable, err := contracts.NewIAxelarExecutable(contractAddress, c.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create executable contract: %w", err)
	}

	tx, err := executable.ExecuteWithToken(c.auth, commandId, sourceChain, sourceAddress, payload, symbol, amount)
	if err != nil {
		log.Error().Msgf("[EvmClient.ExecuteWithToken] Failed: %v", err)
		return nil, err
	}

	return tx, nil
}
