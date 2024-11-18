package evm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const (
	rpcURL         = "wss://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx"
	gatewayAddress = "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D"
)

// ContractCallEvent represents the event structure
type ContractCallEvent struct {
	Sender      common.Address
	DestChain   string
	DestAddr    string
	PayloadHash [32]byte
	Payload     []byte
}

func TestEvmSubscribe(t *testing.T) {
	fmt.Println("Test evm client")

	// Connect to Ethereum client
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to connect to the Ethereum client: %v", err)
	}

	// Get current block
	currentBlock, err := client.BlockNumber(context.Background())
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to get current block: %v", err)
	}
	fmt.Printf("Current block %d\n", currentBlock)

	// Create the event signature
	contractCallSig := []byte("ContractCall(address,string,string,bytes32,bytes)")

	// Create the filter query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(gatewayAddress)},
		Topics: [][]common.Hash{{
			crypto.Keccak256Hash(contractCallSig),
		}},
	}

	// Subscribe to events
	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to subscribe to logs: %v", err)
	}

	// Handle events in a separate goroutine
	go func() {
		for {
			select {
			case err := <-sub.Err():
				fmt.Printf("Received error: %v", err)
			case vLog := <-logs:
				fmt.Println("Log:", vLog)
			}
		}
	}()

	// Keep the program running
	select {}
}
