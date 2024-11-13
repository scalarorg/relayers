package evm

import (
	evm_clients "github.com/scalarorg/relayers/pkg/clients/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

// type EvmListener struct {
// 	client         *ethclient.Client
// 	chainName      string
// 	finalityBlocks int
// 	gatewayAddress common.Address
// 	gateway        *contracts.IAxelarGateway
// 	maxRetry       int
// 	retryDelay     time.Duration
// 	privateKey     *ecdsa.PrivateKey
// 	auth           *bind.TransactOpts
// }

// func NewEvmListeners() ([]*EvmListener, error) {
// 	if len(config.GlobalConfig.EvmNetworks) == 0 {
// 		return nil, fmt.Errorf("no EVM networks configured")
// 	}

// 	listeners := make([]*EvmListener, 0, len(config.GlobalConfig.EvmNetworks))

// 	for _, network := range config.GlobalConfig.EvmNetworks {
// 		client, err := ethclient.Dial(network.RPCUrl)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to connect to EVM network %s: %w", network.Name, err)
// 		}

// 		// Initialize private key (wallet equivalent)
// 		privateKey, err := crypto.HexToECDSA(network.PrivateKey)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse private key for network %s: %w", network.Name, err)
// 		}

// 		chainID, ok := new(big.Int).SetString(network.ChainID, 10)
// 		if !ok {
// 			return nil, fmt.Errorf("invalid chain ID for network %s", network.Name)
// 		}

// 		// Create auth for transactions
// 		auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create auth for network %s: %w", network.Name, err)
// 		}

// 		gatewayAddr := common.HexToAddress(network.Gateway)
// 		if gatewayAddr == (common.Address{}) {
// 			return nil, fmt.Errorf("invalid gateway address for network %s", network.Name)
// 		}

// 		// Initialize gateway contract
// 		gateway, err := contracts.NewIAxelarGateway(gatewayAddr, client)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", network.Name, err)
// 		}

// 		maxRetries := viper.GetInt("MAX_RETRY")
// 		retryDelay := time.Duration(viper.GetInt("RETRY_DELAY")) * time.Millisecond

// 		listener := &EvmListener{
// 			client:         client,
// 			chainName:      network.Name,
// 			finalityBlocks: network.Finality,
// 			gatewayAddress: gatewayAddr,
// 			gateway:        gateway,
// 			maxRetry:       maxRetries,
// 			retryDelay:     retryDelay,
// 			privateKey:     privateKey,
// 			auth:           auth,
// 		}
// 		listeners = append(listeners, listener)
// 	}

// 	return listeners, nil
// }

type EvmAdapter struct {
	BusEventChan         chan *types.EventEnvelope
	BusEventReceiverChan chan *types.EventEnvelope
	evmClients           []*evm_clients.EvmClient
}

// // Rename from NewEvmListeners02 to NewEvmAdapter
// func NewEvmAdapter(configs []config.EvmNetworkConfig, busEventChan chan *types.EventEnvelope, receiverChanBufSize int) (*EvmAdapter, error) {
// 	evmClients, err := evm_clients.NewEvmClients(configs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	adapter := &EvmAdapter{
// 		BusEventChan:         busEventChan,
// 		BusEventReceiverChan: make(chan *types.EventEnvelope, receiverChanBufSize),
// 		evmClients:           evmClients,
// 	}

// 	return adapter, nil
// }

// func (ea *EvmAdapter) Close() {
// 	for _, evmClient := range ea.evmClients {
// 		if evmClient.Client != nil {
// 			evmClient.Client.Close()
// 		}
// 	}
// }

// func (ea *EvmAdapter) PollEventsFromAllEvmNetworks(pollInterval time.Duration) {
// 	// Create ticker for each EVM client
// 	for _, evmClient := range ea.evmClients {
// 		go ea.pollEvmNetwork(evmClient, pollInterval)
// 	}

// 	// Keep main routine running
// 	select {}
// }

// // New helper function to poll individual network
// func (ea *EvmAdapter) pollEvmNetwork(evmClient *evm_clients.EvmClient, pollInterval time.Duration) {
// 	ticker := time.NewTicker(pollInterval)
// 	defer ticker.Stop()

// 	for {
// 		// query last block from db
// 		lastBlock, err := db.DbAdapter.GetLastBlock(evmClient.ChainName)
// 		if err != nil {
// 			log.Error().Err(err).Msg("[EvmAdapter] Failed to get last block")
// 			continue
// 		}

// 		query := ethereum.FilterQuery{
// 			FromBlock: lastBlock,
// 			ToBlock:   nil,
// 			Addresses: []common.Address{
// 				evmClient.GatewayAddress,
// 			},
// 		}

// 		logs, err := evmClient.Client.FilterLogs(context.Background(), query)
// 		if err != nil {
// 			log.Error().Err(err).Msgf("[EvmAdapter] Failed to filter logs for %s", evmClient.ChainName)
// 			continue
// 		}

// 		if len(logs) > 0 {
// 			// Process logs
// 			for _, log := range logs {
// 				envelope, err := parseEvmEventToEnvelope(evmClient.ChainName, log)
// 				if err != nil {
// 					continue
// 				}

// 				ea.SendEvent(&envelope)
// 			}

// 			// Update last block to db
// 			err = db.DbAdapter.UpdateLastBlock(evmClient.ChainName, big.NewInt(int64(logs[len(logs)-1].BlockNumber)))
// 			if err != nil {
// 				log.Error().Err(err).Msgf("[EvmAdapter] Failed to update last block for %s", evmClient.ChainName)
// 			} else {
// 				log.Info().Msgf("[EvmAdapter] Updated last block for %s to %s", evmClient.ChainName, big.NewInt(int64(logs[len(logs)-1].BlockNumber)).String())
// 			}
// 		}

// 		// Wait for next tick
// 		<-ticker.C
// 	}
// }

// func (ea *EvmAdapter) ListenEventsFromBusChannel() {
// 	for event := range ea.BusEventReceiverChan {
// 		switch event.Component {
// 		case "EvmAdapter":
// 			fmt.Printf("Received event in EvmAdapter: %+v\n", event)
// 			ea.handleEvmEvent(*event)
// 		default:
// 			// Pass the event that not belong to DbAdapter
// 		}
// 	}
// }

// func (ea *EvmAdapter) SendEvent(event *types.EventEnvelope) {
// 	ea.BusEventChan <- event
// 	log.Debug().Msgf("[EvmAdapter] Sent event to bus channel: %v", *event)
// }

// func (ea *EvmAdapter) handleEvmEvent(eventEnvelope types.EventEnvelope) {
// 	evmClientName := eventEnvelope.ReceiverClientName
// 	var evmClient *evm_clients.EvmClient
// 	for _, client := range ea.evmClients {
// 		if client.ChainName == evmClientName {
// 			evmClient = client
// 			break
// 		}
// 	}
// 	switch eventEnvelope.Handler {
// 	case "handleCosmosToEvmCallContractCompleteEvent":
// 		results, err := ea.handleCosmosToEvmCallContractCompleteEvent(evmClient, eventEnvelope.Data.(types.HandleCosmosToEvmCallContractCompleteEventData))
// 		if err != nil {
// 			log.Error().Err(err).Msg("[EvmAdapter] Failed to handle event")
// 		}

// 		// Send to DbAdapter to update Status
// 		for _, result := range results {
// 			ea.SendEvent(&types.EventEnvelope{
// 				Component:        "DbAdapter",
// 				SenderClientName: evmClientName,
// 				Handler:          "UpdateEventStatus",
// 				Data:             result,
// 			})
// 		}
// 	case "waitForTransaction":
// 		hash := eventEnvelope.Data.(types.WaitForTransactionData).Hash
// 		event := eventEnvelope.Data.(types.WaitForTransactionData).Event
// 		ea.handleWaitForTransaction(evmClient, hash)
// 		ea.SendEvent(&types.EventEnvelope{
// 			Component:        "AxelarAdapter",
// 			SenderClientName: evmClientName,
// 			Handler:          "handleEvmToCosmosEvent",
// 			Data:             event,
// 		})
// 	}
// }
