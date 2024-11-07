package evm

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/types"
)

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
				fmt.Printf("Received log: %v\n", log)
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
				case ea.BusEventChan <- &eventEnvelope:
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
				case ea.BusEventChan <- &eventEnvelope:
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
				case ea.BusEventChan <- &eventEnvelope:
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
