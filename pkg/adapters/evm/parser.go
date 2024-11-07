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
	*contracts.IAxelarGatewayContractCallApproved |
		*contracts.IAxelarGatewayContractCall |
		*contracts.IAxelarGatewayExecuted
}

func parseEvmEventToEnvelope(
	currentChainName string,
	log eth_types.Log,
) (types.EventEnvelope, error) {
	// Try parse the log into different events
	eventArgs, err := parseLogIntoEventArgs(log)
	if err != nil {
		return types.EventEnvelope{}, err
	}

	envelope, err := parseEventIntoEnvelope(currentChainName, eventArgs, log)
	if err != nil {
		return types.EventEnvelope{}, err
	}

	return envelope, nil
}

func parseLogIntoEventArgs(log eth_types.Log) (any, error) {
	// Try parsing as ContractCallApproved
	if eventArgs, err := parseContractCallApproved(log); err == nil {
		return eventArgs, nil
	}

	// Try parsing as ContractCall
	if eventArgs, err := parseContractCall(log); err == nil {
		return eventArgs, nil
	}

	// Try parsing as Execute
	if eventArgs, err := parseExecute(log); err == nil {
		return eventArgs, nil
	}

	return nil, fmt.Errorf("failed to parse log into any known event type")
}

func parseEventIntoEnvelope(currentChainName string, eventArgs any, log eth_types.Log) (types.EventEnvelope, error) {
	switch args := eventArgs.(type) {
	case *contracts.IAxelarGatewayContractCallApproved:
		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayContractCallApproved](args, currentChainName, log)
		if err != nil {
			return types.EventEnvelope{}, err
		}
		return types.EventEnvelope{
			Component:        "DbAdapter",
			SenderClientName: currentChainName,
			Handler:          "FindCosmosToEvmCallContractApproved",
			Data:             event,
		}, nil

	case *contracts.IAxelarGatewayContractCall:
		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayContractCall](args, currentChainName, log)
		if err != nil {
			return types.EventEnvelope{}, err
		}
		return types.EventEnvelope{
			Component:        "DbAdapter",
			SenderClientName: currentChainName,
			Handler:          "CreateEvmCallContractEvent",
			Data:             event,
		}, nil

	case *contracts.IAxelarGatewayExecuted:
		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayExecuted](args, currentChainName, log)
		if err != nil {
			return types.EventEnvelope{}, err
		}
		return types.EventEnvelope{
			Component:        "DbAdapter",
			SenderClientName: currentChainName,
			Handler:          "CreateEvmExecutedEvent",
			Data:             event,
		}, nil

	default:
		return types.EventEnvelope{}, fmt.Errorf("unknown event type: %T", eventArgs)
	}
}

func parseEventArgsIntoEvent[T ValidEvmEvent](eventArgs T, currentChainName string, log eth_types.Log) (*types.EvmEvent[T], error) {
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

// parseAnyEvent is a generic function that parses any EVM event into a standardized EvmEvent structure
func parseEvmEventContractCallApproved[T *contracts.IAxelarGatewayContractCallApproved](
	currentChainName string,
	log eth_types.Log,
) (*types.EvmEvent[T], error) {
	eventArgs, err := parseContractCallApproved(log)
	if err != nil {
		return nil, err
	}

	event, err := parseEventArgsIntoEvent[T](eventArgs, currentChainName, log)
	if err != nil {
		return nil, err
	}

	return event, nil
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

func parseEvmEventContractCall[T *contracts.IAxelarGatewayContractCall](
	currentChainName string,
	log eth_types.Log,
) (*types.EvmEvent[T], error) {
	eventArgs, err := parseContractCall(log)
	if err != nil {
		return nil, err
	}

	event, err := parseEventArgsIntoEvent[T](eventArgs, currentChainName, log)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func parseContractCall(
	log eth_types.Log,
) (*contracts.IAxelarGatewayContractCall, error) {
	event := struct {
		Sender                     common.Address
		DestinationChain           string
		DestinationContractAddress string
		PayloadHash                [32]byte
		Payload                    []byte
	}{}

	contractAbi := contracts_abi.GetEventABI("ContractCall")
	parsedAbi, err := abi.JSON(strings.NewReader(contractAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	if err := parsedAbi.UnpackIntoInterface(&event, "ContractCall", log.Data); err != nil {
		return nil, fmt.Errorf("failed to unpack event: %w", err)
	}

	// Add validation for required fields
	if len(event.DestinationChain) == 0 || !isValidUTF8(event.DestinationChain) {
		return nil, fmt.Errorf("invalid destination chain value")
	}

	if len(event.DestinationContractAddress) == 0 || !isValidUTF8(event.DestinationContractAddress) {
		return nil, fmt.Errorf("invalid destination address value")
	}

	var eventArgs contracts.IAxelarGatewayContractCall = contracts.IAxelarGatewayContractCall{
		Sender:                     event.Sender,
		DestinationChain:           event.DestinationChain,
		DestinationContractAddress: event.DestinationContractAddress,
		PayloadHash:                event.PayloadHash,
		Payload:                    event.Payload,
		Raw:                        log,
	}

	fmt.Printf("[EVMListener] [parseContractCall] eventArgs: %+v\n", eventArgs)

	return &eventArgs, nil
}

func parseEvmEventExecute[T *contracts.IAxelarGatewayExecuted](
	currentChainName string,
	log eth_types.Log,
) (*types.EvmEvent[T], error) {
	eventArgs, err := parseExecute(log)
	if err != nil {
		return nil, err
	}

	event, err := parseEventArgsIntoEvent[T](eventArgs, currentChainName, log)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func parseExecute(
	log eth_types.Log,
) (*contracts.IAxelarGatewayExecuted, error) {
	event := struct {
		CommandId [32]byte
	}{}

	contractAbi := contracts_abi.GetEventABI("Executed")
	parsedAbi, err := abi.JSON(strings.NewReader(contractAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Check if log data size matches exactly what we expect (32 bytes for CommandId)
	if len(log.Data) != 32 {
		return nil, fmt.Errorf("unexpected log data size: got %d bytes, want 32 bytes", len(log.Data))
	}

	if err := parsedAbi.UnpackIntoInterface(&event, "Executed", log.Data); err != nil {
		return nil, fmt.Errorf("failed to unpack event: %w", err)
	}

	// Add validation for required fields
	if len(event.CommandId) == 0 {
		return nil, fmt.Errorf("invalid command id value")
	}

	var eventArgs contracts.IAxelarGatewayExecuted = contracts.IAxelarGatewayExecuted{
		CommandId: event.CommandId,
		Raw:       log,
	}

	fmt.Printf("[EVMListener] [parseExecute] eventArgs: %+v\n", eventArgs)

	return &eventArgs, nil
}

// Add helper function to validate UTF-8 strings
func isValidUTF8(s string) bool {
	return strings.ToValidUTF8(s, "") == s
}