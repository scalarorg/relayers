package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/spf13/viper"
)

type EvmListener struct {
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

func NewEvmListeners() ([]*EvmListener, error) {
	if len(config.GlobalConfig.EvmNetworks) == 0 {
		return nil, fmt.Errorf("no EVM networks configured")
	}

	listeners := make([]*EvmListener, 0, len(config.GlobalConfig.EvmNetworks))

	for _, network := range config.GlobalConfig.EvmNetworks {
		client, err := ethclient.Dial(network.RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to EVM network %s: %w", network.Name, err)
		}

		// Initialize private key (wallet equivalent)
		privateKey, err := crypto.HexToECDSA(network.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key for network %s: %w", network.Name, err)
		}

		chainID, ok := new(big.Int).SetString(network.ChainID, 10)
		if !ok {
			return nil, fmt.Errorf("invalid chain ID for network %s", network.Name)
		}

		// Create auth for transactions
		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create auth for network %s: %w", network.Name, err)
		}

		gatewayAddr := common.HexToAddress(network.Gateway)
		if gatewayAddr == (common.Address{}) {
			return nil, fmt.Errorf("invalid gateway address for network %s", network.Name)
		}

		// Initialize gateway contract
		gateway, err := contracts.NewIAxelarGateway(gatewayAddr, client)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", network.Name, err)
		}

		maxRetries := viper.GetInt("MAX_RETRY")
		retryDelay := time.Duration(viper.GetInt("RETRY_DELAY")) * time.Millisecond

		listener := &EvmListener{
			client:         client,
			chainName:      network.Name,
			finalityBlocks: network.Finality,
			gatewayAddress: gatewayAddr,
			gateway:        gateway,
			maxRetry:       maxRetries,
			retryDelay:     retryDelay,
			privateKey:     privateKey,
			auth:           auth,
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// Listen monitors events from the gateway contract and sends them to the provided channel
func (l *EvmListener) Listen(eventName string, eventChan chan<- *EvmEvent) error {
	log.Info().Msgf(
		"[EVMListener] [%s] Listening to %s event from gateway contract %s",
		l.chainName,
		eventName,
		l.gatewayAddress.Hex(),
	)

	// Get current block number
	currentBlock, err := l.client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}

	// Create event filter
	filterOpts := &bind.FilterOpts{
		Start: currentBlock,
	}

	// Watch for events
	eventCh := make(chan *contracts.IAxelarGatewayContractCall)
	sub, err := l.gateway.WatchContractCall(filterOpts, eventCh)
	if err != nil {
		return fmt.Errorf("failed to watch for events: %w", err)
	}

	go func() {
		defer sub.Unsubscribe()

		for {
			select {
			case err := <-sub.Err():
				log.Error().Msgf("[EVMListener] [%s] Subscription error: %v", l.chainName, err)
				return

			case event := <-eventCh:
				if event.Raw.BlockNumber <= currentBlock {
					continue
				}

				// Parse the event into EvmEvent struct
				evmEvent, err := parseAnyEvent[*contracts.IAxelarGatewayContractCall](
					l.chainName,
					l.client,
					event.Raw,
					uint64(l.finalityBlocks),
				)
				if err != nil {
					log.Error().Msgf("[EVMListener] [%s] Error while parsing event: %v", l.chainName, err)
					continue
				}

				log.Debug().Msgf(
					"[EVMListener] [%s] [%s] Parsed event: %+v",
					l.chainName,
					eventName,
					mEvent,
				)

				// Send event to channel
				eventChan <- evmEvent
			}
		}
	}()

	return nil
}