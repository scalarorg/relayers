package evm

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	contracts_abi "github.com/scalarorg/relayers/pkg/contracts-abi"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/types"
)

type ValidEvmEvent interface {
	*contracts.IAxelarGatewayContractCallApproved
}

// parseAnyEvent is a generic function that parses any EVM event into a standardized EvmEvent structure
func parseEvmEvent[T ValidEvmEvent](
	currentChainName string,
	log eth_types.Log,
) (*types.EvmEvent[T], error) {
	var eventArgs T

	eventArgs, err := parseContractCallApproved(log)
	if err != nil {
		return nil, err
	}

	// Get the value of eventArgs using reflection
	v := reflect.ValueOf(eventArgs).Elem()
	sourceChain := currentChainName
	if f := v.FieldByName("SourceChain"); f.IsValid() {
		sourceChain = f.String()
	}
	destinationChain := currentChainName
	if f := v.FieldByName("DestinationChain"); f.IsValid() {
		destinationChain = f.String()
	}

	return &types.EvmEvent[T]{
		Hash:             log.TxHash.Hex(),
		BlockNumber:      log.BlockNumber,
		LogIndex:         log.Index,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		Args:             eventArgs,
	}, nil
}

func parseContractCallApproved(
	log eth_types.Log,
) (*contracts.IAxelarGatewayContractCallApproved, error) {
	event := struct {
		CommandId        [32]byte
		SourceChain      string
		SourceAddress    string
		ContractAddress  common.Address
		PayloadHash      [32]byte
		SourceTxHash     [32]byte
		SourceEventIndex *big.Int
	}{}

	contractAbi := contracts_abi.GetEventABI("ContractCallApproved")
	parsedAbi, err := abi.JSON(strings.NewReader(contractAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	if err := parsedAbi.UnpackIntoInterface(&event, "ContractCallApproved", log.Data); err != nil {
		return nil, fmt.Errorf("failed to unpack event: %w", err)
	}

	// Add validation for required fields
	if len(event.SourceChain) == 0 || !isValidUTF8(event.SourceChain) {
		return nil, fmt.Errorf("invalid source chain value")
	}

	if len(event.SourceAddress) == 0 || !isValidUTF8(event.SourceAddress) {
		return nil, fmt.Errorf("invalid source address value")
	}

	var eventArgs contracts.IAxelarGatewayContractCallApproved = contracts.IAxelarGatewayContractCallApproved{
		CommandId:        event.CommandId,
		SourceChain:      event.SourceChain,
		SourceAddress:    event.SourceAddress,
		ContractAddress:  event.ContractAddress,
		PayloadHash:      event.PayloadHash,
		SourceTxHash:     event.SourceTxHash,
		SourceEventIndex: event.SourceEventIndex,
		Raw:              log,
	}

	fmt.Printf("[EVMListener] [parseContractCallApproved] eventArgs: %+v\n", eventArgs)

	return &eventArgs, nil
}

// Add helper function to validate UTF-8 strings
func isValidUTF8(s string) bool {
	return strings.ToValidUTF8(s, "") == s
}
