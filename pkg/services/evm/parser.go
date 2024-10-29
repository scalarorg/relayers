package evm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// parseAnyEvent is a generic function that parses any EVM event into a standardized EvmEvent structure
func parseAnyEvent[T any](
	currentChainName string,
	provider *ethclient.Client,
	event types.Log,
	finalityBlocks uint64,
) (*EvmEvent[T], error) {
	// Get the transaction receipt
	receipt, err := provider.TransactionReceipt(context.Background(), event.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Find the log index
	var logIndex uint
	for i, log := range receipt.Logs {
		if log.Index == event.Index {
			logIndex = uint(i)
			break
		}
	}

	// Log the event index
	fmt.Printf("[EVMListener] [%s] [parseAnyEvent] eventIndex: %d\n", currentChainName, logIndex)

	// Create a closure for the WaitForFinality function
	waitForFinality := func() (*types.Receipt, error) {
		return provider.TransactionReceipt(context.Background(), event.TxHash)
	}

	// Filter event arguments
	args := filterEventArgs(event)

	// Create and return the EvmEvent
	return &EvmEvent[T]{
		Hash:             event.TxHash.Hex(),
		BlockNumber:      event.BlockNumber,
		LogIndex:         logIndex,
		SourceChain:      args["sourceChain"].(string),
		DestinationChain: args["destinationChain"].(string),
		WaitForFinality:  waitForFinality,
		Args:             args,
	}, nil
}

// filterEventArgs filters the event arguments to exclude numeric keys
func filterEventArgs(event types.Log) map[string]interface{} {
	args := make(map[string]interface{})
	for key, value := range event.Topics {
		if _, err := strconv.Atoi(key); err != nil {
			args[key] = value
		}
	}
	return args
}
