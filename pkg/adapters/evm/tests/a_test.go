package evm_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestA(t *testing.T) {
	TEST_RPC_ENDPOINT := "https://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx"
	TEST_CONTRACT_ADDRESS := "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D"

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 29*time.Minute)
	defer cancel()

	// Connect to a test network
	client, err := ethclient.Dial(TEST_RPC_ENDPOINT)
	require.NoError(t, err)
	defer client.Close()

	// Initialize the contract
	contractAddress := common.HexToAddress(TEST_CONTRACT_ADDRESS)
	gateway, err := contracts.NewIAxelarGateway(contractAddress, client)
	require.NoError(t, err)

	// Create channels for events and errors
	eventCh := make(chan *contracts.IAxelarGatewayContractCallApproved)
	errorCh := make(chan error)

	// Start listening for ContractCallApproved events
	sub, err := gateway.WatchContractCallApproved(
		&bind.WatchOpts{Context: ctx},
		eventCh,
		nil, // Empty slice for all command IDs
		nil, // Empty slice for all contract addresses
		nil, // Empty slice for all payloads
	)
	t.Log("Subscribed to ContractCallApproved events")
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Add logging for initial setup
	t.Logf("Connected to network at: %s", TEST_RPC_ENDPOINT)
	t.Logf("Watching contract at address: %s", TEST_CONTRACT_ADDRESS)

	// Add error channel handling from subscription
	go func() {
		for err := range sub.Err() {
			errorCh <- fmt.Errorf("subscription error: %v", err)
		}
	}()

	// Add a shorter timeout for debugging
	timeout := time.After(5 * time.Minute) // Reduced from 30 minutes for faster debugging

	// Add periodic health check
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	eventCount := 0

	for {
		select {
		case err := <-errorCh:
			t.Fatalf("Error during test: %v", err)
		case event := <-eventCh:
			eventCount++
			t.Logf("Received event %d: CommandID=%x, SourceChain=%s, SourceAddress=%s, ContractAddress=%s",
				eventCount, event.CommandId, event.SourceChain, event.SourceAddress, event.ContractAddress)
			// Add assertions about the event
			assert.NotEmpty(t, event.CommandId)
			assert.NotEmpty(t, event.SourceChain)
			assert.NotEmpty(t, event.SourceAddress)
			assert.NotEmpty(t, event.ContractAddress)
		case <-ticker.C:
			// Periodic health check
			number, err := client.BlockNumber(ctx)
			if err != nil {
				t.Logf("Failed to get block number: %v", err)
			} else {
				t.Logf("Current block number: %d", number)
			}

			// Correct way to check subscription status
			select {
			case err := <-sub.Err():
				if err != nil {
					t.Logf("Subscription error: %v", err)
				}
			default:
				t.Log("Subscription is healthy")
			}
		case <-timeout:
			// Add more context when timing out
			number, _ := client.BlockNumber(ctx)
			t.Logf("Test timed out. Final block number: %d, Total events: %d", number, eventCount)
			return
		case <-ctx.Done():
			t.Log("Context cancelled")
			return
		}
	}
}
