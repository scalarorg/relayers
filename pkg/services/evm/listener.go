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
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	evm_clients "github.com/scalarorg/relayers/pkg/clients/evm"
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

type EvmAdapter struct {
	EventChan  chan *types.EventEnvelope
	evmClients []*evm_clients.EvmClient
}

// Rename from NewEvmListeners02 to NewEvmAdapter
func NewEvmAdapter(configs []config.EvmNetworkConfig, eventChan chan *types.EventEnvelope) (*EvmAdapter, error) {
	evmClients, err := evm_clients.NewEvmClients(configs)
	if err != nil {
		return nil, err
	}

	adapter := &EvmAdapter{
		EventChan:  eventChan,
		evmClients: evmClients,
	}

	return adapter, nil
}

// Rename from ListenForContractCallApproved to PollForContractCallApproved
func (ea *EvmAdapter) PollForContractCallApproved() error {
	// Create ticker for 1-minute intervals
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		for _, evmClient := range ea.evmClients {
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(6922244),
				ToBlock:   big.NewInt(6922244),
				Addresses: []common.Address{
					evmClient.GatewayAddress,
				},
			}

			logs, err := evmClient.Client.FilterLogs(context.Background(), query)
			if err != nil {
				fmt.Printf("Failed to filter logs: %v\n", err)
				// Continue instead of returning error to keep the loop running
				continue
			}

			// Process logs
			for _, log := range logs {
				event, err := parseEvmEventContractCallApproved[*contracts.IAxelarGatewayContractCallApproved](evmClient.ChainName, log)
				if err != nil {
					continue
				}

				// Create the event envelope
				eventEnvelope := types.EventEnvelope{
					Component:        "DbAdapter",
					SenderClientName: evmClient.ChainName,
					Handler:          "FindCosmosToEvmCallContractApproved",
					Data:             event,
				}

				// Send the envelope to the channel
				select {
				case ea.EventChan <- &eventEnvelope:
					// Event sent successfully
				default:
					fmt.Printf("Warning: Unable to send event to channel, might be full or closed\n")
				}
			}
		}

		// Wait for next tick
		<-ticker.C
	}
}

func (ea *EvmAdapter) PollForContractCall() error {
	// Create ticker for 1-minute intervals
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		for _, evmClient := range ea.evmClients {
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(6922208),
				ToBlock:   big.NewInt(6922208),
				Addresses: []common.Address{
					evmClient.GatewayAddress,
				},
			}

			logs, err := evmClient.Client.FilterLogs(context.Background(), query)
			if err != nil {
				fmt.Printf("Failed to filter logs: %v\n", err)
				// Continue instead of returning error to keep the loop running
				continue
			}

			// Process logs
			for _, log := range logs {
				fmt.Printf("Received log: %v\n", log)
				event, err := parseEvmEventContractCall[*contracts.IAxelarGatewayContractCall](evmClient.ChainName, log)
				if err != nil {
					continue
				}

				// Create the event envelope
				eventEnvelope := types.EventEnvelope{
					Component:        "DbAdapter",
					SenderClientName: evmClient.ChainName,
					Handler:          "CreateEvmCallContractEvent",
					Data:             event,
				}

				// Send the envelope to the channel
				select {
				case ea.EventChan <- &eventEnvelope:
					// Event sent successfully
				default:
					fmt.Printf("Warning: Unable to send event to channel, might be full or closed\n")
				}
			}
		}

		// Wait for next tick
		<-ticker.C
	}
}

func (ea *EvmAdapter) PollForExecute() error {
	// Create ticker for 1-minute intervals
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		for _, evmClient := range ea.evmClients {
			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(7007979),
				ToBlock:   big.NewInt(7007979),
				Addresses: []common.Address{
					evmClient.GatewayAddress,
				},
			}

			logs, err := evmClient.Client.FilterLogs(context.Background(), query)
			if err != nil {
				fmt.Printf("Failed to filter logs: %v\n", err)
				// Continue instead of returning error to keep the loop running
				continue
			}

			// Process logs
			for _, log := range logs {
				fmt.Printf("Received log: %v\n", log)
				event, err := parseEvmEventExecute[*contracts.IAxelarGatewayExecuted](evmClient.ChainName, log)
				if err != nil {
					continue
				}

				// Create the event envelope
				eventEnvelope := types.EventEnvelope{
					Component:        "DbAdapter",
					SenderClientName: evmClient.ChainName,
					Handler:          "CreateEvmExecutedEvent",
					Data:             event,
				}

				// Send the envelope to the channel
				select {
				case ea.EventChan <- &eventEnvelope:
					// Event sent successfully
				default:
					fmt.Printf("Warning: Unable to send event to channel, might be full or closed\n")
				}
			}
		}

		// Wait for next tick
		<-ticker.C
	}
}

func (ea *EvmAdapter) Close() {
	for _, evmClient := range ea.evmClients {
		if evmClient.Client != nil {
			evmClient.Client.Close()
		}
	}
}

// Add getter method for the channel
func (ea *EvmAdapter) GetEventChannel() <-chan *types.EventEnvelope {
	return ea.EventChan
}

func (ea *EvmAdapter) ListenEvents() {
	for event := range ea.EventChan {
		switch event.Component {
		case "EvmAdapter":
			fmt.Printf("Received event in EvmAdapter: %+v\n", event)
			ea.handleEvmEvent(*event)
		default:
			// Pass the event that not belong to DbAdapter
		}
	}
}

func (ea *EvmAdapter) SendEvent(event *types.EventEnvelope) {
	ea.EventChan <- event
	log.Debug().Msgf("[EvmAdapter] Sent event to channel: %v", event.Handler)
}

func (ea *EvmAdapter) handleEvmEvent(eventEnvelope types.EventEnvelope) {
	evmClientName := eventEnvelope.ReceiverClientName
	var evmClient *evm_clients.EvmClient
	for _, client := range ea.evmClients {
		if client.ChainName == evmClientName {
			evmClient = client
			break
		}
	}
	switch eventEnvelope.Handler {
	case "handleCosmosToEvmCallContractCompleteEvent":
		results, err := ea.handleCosmosToEvmCallContractCompleteEvent(evmClient, eventEnvelope.Data.(types.HandleCosmosToEvmCallContractCompleteEventData))
		if err != nil {
			log.Error().Err(err).Msg("[EvmAdapter] Failed to handle event")
		}

		// Send to DbAdapter to update Status
		for _, result := range results {
			ea.SendEvent(&types.EventEnvelope{
				Component:        "DbAdapter",
				SenderClientName: evmClientName,
				Handler:          "UpdateEventStatus",
				Data:             result,
			})
		}
	case "waitForTransaction":
		hash := eventEnvelope.Data.(types.WaitForTransactionData).Hash
		event := eventEnvelope.Data.(types.WaitForTransactionData).Event
		ea.handleWaitForTransaction(evmClient, hash)
		ea.SendEvent(&types.EventEnvelope{
			Component:        "AxelarAdapter",
			SenderClientName: evmClientName,
			Handler:          "handleEvmToCosmosEvent",
			Data:             event,
		})
	}
}
