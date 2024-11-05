package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/types"
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

// Define the event structure

type EvmAdapter struct {
	client         *ethclient.Client
	chainName      string
	gatewayAddress common.Address
	gateway        *contracts.IAxelarGateway
	eventChan      chan *types.EventEnvelope
	auth           *bind.TransactOpts
	config         config.EvmNetworkConfig
}

// Rename from NewEvmListeners02 to NewEvmAdapter
func NewEvmAdapter(config config.EvmNetworkConfig, eventChan chan *types.EventEnvelope) (*EvmAdapter, error) {
	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 29*time.Minute)
	defer cancel()

	// Connect to a test network
	rpc, err := rpc.DialContext(ctx, config.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM network %s: %w", config.Name, err)
	}

	client := ethclient.NewClient(rpc)
	if err != nil {
		rpc.Close()
		return nil, fmt.Errorf("failed to create client for EVM network %s: %w", config.Name, err)
	}

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

	adapter := &EvmAdapter{
		client:         client,
		chainName:      config.Name,
		gatewayAddress: gatewayAddress,
		gateway:        gateway,
		eventChan:      eventChan,
		auth:           auth,
		config:         config,
	}

	return adapter, nil
}

// Rename from ListenForContractCallApproved to PollForContractCallApproved
func (ea *EvmAdapter) PollForContractCallApproved() error {
	// Create ticker for 1-minute intervals
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(6922244),
			ToBlock:   big.NewInt(6922244),
			Addresses: []common.Address{
				ea.gatewayAddress,
			},
		}

		logs, err := ea.client.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Printf("Failed to filter logs: %v\n", err)
			// Continue instead of returning error to keep the loop running
			continue
		}

		// Process logs
		for _, log := range logs {
			event, err := parseEvmEvent[*contracts.IAxelarGatewayContractCallApproved](ea.chainName, log)
			if err != nil {
				continue
			}

			// Create the event envelope
			eventEnvelope := types.EventEnvelope{
				Component: "DbAdapter",
				Handler:   "FindCosmosToEvmCallContractApproved",
				Data:      event,
			}

			// Send the envelope to the channel
			select {
			case ea.eventChan <- &eventEnvelope:
				// Event sent successfully
			default:
				fmt.Printf("Warning: Unable to send event to channel, might be full or closed\n")
			}
		}

		// Wait for next tick
		<-ticker.C
	}
}

func (ea *EvmAdapter) Close() {
	if ea.client != nil {
		ea.client.Close()
	}
}

// Add getter method for the channel
func (ea *EvmAdapter) GetEventChannel() <-chan *types.EventEnvelope {
	return ea.eventChan
}

func (ea *EvmAdapter) listenEvents() {
	for event := range ea.eventChan {
		switch event.Component {
		case "EvmAdapter":
			ea.handleEvmEvent(*event)
		default:
			// Pass the event that not belong to DbAdapter
		}
	}
}

func (ea *EvmAdapter) SendEvent(event *types.EventEnvelope) {
	ea.eventChan <- event
	log.Debug().Msgf("[EvmAdapter] Sent event to channel: %v", event.Handler)
}

func (ea *EvmAdapter) handleEvmEvent(eventEnvelope types.EventEnvelope) {
	switch eventEnvelope.Handler {
	case "handleCosmosToEvmCallContractCompleteEvent":
		results, err := ea.handleCosmosToEvmCallContractCompleteEvent(eventEnvelope.Data.(types.HandleCosmosToEvmCallContractCompleteEventData))
		if err != nil {
			log.Error().Err(err).Msg("[EvmAdapter] Failed to handle event")
		}

		// Send to DbAdapter to update Status
		for _, result := range results {
			ea.SendEvent(&types.EventEnvelope{
				Component: "DbAdapter",
				Handler:   "UpdateEventStatus",
				Data:      result,
			})
		}
	}
}
