package parser

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	eth_types "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
)

type ValidEvmEvent interface {
	*contracts.IAxelarGatewayContractCallApproved |
		*contracts.IAxelarGatewayContractCall |
		*contracts.IAxelarGatewayExecuted
}

var (
	scalarGatewayAbi *abi.ABI
)

func init() {
	var err error
	scalarGatewayAbi, err = contracts.IAxelarGatewayMetaData.GetAbi()
	if err != nil {
		log.Error().Msgf("failed to get scalar gateway abi: %v", err)
	}
}

func GetScalarGatewayAbi() *abi.ABI {
	return scalarGatewayAbi
}

func getEventIndexedArguments(eventName string) abi.Arguments {
	gatewayAbi := GetScalarGatewayAbi()
	var args abi.Arguments
	if event, ok := gatewayAbi.Events[eventName]; ok {
		for _, arg := range event.Inputs {
			if arg.Indexed {
				//Cast to non-indexed
				args = append(args, abi.Argument{
					Name: arg.Name,
					Type: arg.Type,
				})
			}
		}
	}
	return args
}

func parseEventData(receiptLog *eth_types.Log, eventName string, eventData any) error {
	gatewayAbi := GetScalarGatewayAbi()
	if gatewayAbi.Events[eventName].ID != receiptLog.Topics[0] {
		return fmt.Errorf("receipt log topic 0 does not match %s event id", eventName)
	}
	// Unpack non-indexed arguments
	if err := gatewayAbi.UnpackIntoInterface(eventData, eventName, receiptLog.Data); err != nil {
		return fmt.Errorf("failed to unpack event: %w", err)
	}
	// Unpack indexed arguments
	// concat all topic data from second element into single buffer
	var buffer []byte
	for i := 1; i < len(receiptLog.Topics); i++ {
		buffer = append(buffer, receiptLog.Topics[i].Bytes()...)
	}
	indexedArgs := getEventIndexedArguments(eventName)
	if len(buffer) > 0 && len(indexedArgs) > 0 {
		unpacked, err := indexedArgs.Unpack(buffer)
		if err == nil {
			indexedArgs.Copy(eventData, unpacked)
		}
	}
	return nil
}

func createEvmEventFromArgs[T ValidEvmEvent](eventArgs T, log *eth_types.Log) *EvmEvent[T] {
	// Get the value of eventArgs using reflection
	// v := reflect.ValueOf(eventArgs).Elem()
	// sourceChain := currentChainName
	// if f := v.FieldByName("SourceChain"); f.IsValid() {
	// 	sourceChain = f.String()
	// }
	// destinationChain := currentChainName
	// if f := v.FieldByName("DestinationChain"); f.IsValid() {
	// 	destinationChain = f.String()
	// }

	return &EvmEvent[T]{
		Hash:        log.TxHash.Hex(),
		BlockNumber: log.BlockNumber,
		LogIndex:    log.Index,
		Args:        eventArgs,
	}
}

func ParseLogs(receiptLogs []*eth_types.Log) AllEvmEvents {
	events := AllEvmEvents{}
	for _, reciptLog := range receiptLogs {
		// Try parsing as ContractCallApproved
		if eventArgs, err := parseContractCallApproved(reciptLog); err == nil {
			events.ContractCallApproved = createEvmEventFromArgs(eventArgs, reciptLog)
			continue
		}

		// Try parsing as ContractCall
		if eventArgs, err := parseContractCall(reciptLog); err == nil {
			events.ContractCall = createEvmEventFromArgs(eventArgs, reciptLog)
			continue
		}

		// Try parsing as Execute
		if eventArgs, err := parseExecute(reciptLog); err == nil && eventArgs != nil {
			events.Executed = createEvmEventFromArgs(eventArgs, reciptLog)
			continue
		}
	}
	return events
}
func parseContractCall(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayContractCall, error) {
	var eventArgs contracts.IAxelarGatewayContractCall = contracts.IAxelarGatewayContractCall{
		Raw: *receiptLog,
	}
	if err := parseEventData(receiptLog, "ContractCall", &eventArgs); err != nil {
		return nil, err
	}

	log.Debug().Any("parsedEvent", eventArgs).Msg("[EvmClient] [parseContractCall]")

	return &eventArgs, nil
}

func parseContractCallApproved(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayContractCallApproved, error) {
	eventArgs := contracts.IAxelarGatewayContractCallApproved{
		Raw: *receiptLog,
	}
	if err := parseEventData(receiptLog, "ContractCallApproved", &eventArgs); err != nil {
		return nil, err
	}

	return &eventArgs, nil
}

func parseExecute(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayExecuted, error) {

	var eventArgs contracts.IAxelarGatewayExecuted = contracts.IAxelarGatewayExecuted{
		Raw: *receiptLog,
	}
	if err := parseEventData(receiptLog, "Executed", &eventArgs); err != nil {
		return nil, err
	}

	return &eventArgs, nil
}

// Todo: Check if this is the correct way to extract the destination chain
// Maybe add destination chain to the event.Log
//
//	func extractDestChainFromEvmGwContractCallApproved(event *contracts.IAxelarGatewayContractCallApproved) string {
//		return event.SourceChain
//	}
func parseLogIntoEventArgs(log *eth_types.Log) (any, error) {
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

// func parseEventIntoEnvelope(currentChainName string, eventArgs any, log eth_types.Log) (types.EventEnvelope, error) {
// 	switch args := eventArgs.(type) {
// 	case *contracts.IAxelarGatewayContractCallApproved:
// 		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayContractCallApproved](args, currentChainName, log)
// 		if err != nil {
// 			return types.EventEnvelope{}, err
// 		}
// 		return types.EventEnvelope{
// 			Component:        "DbAdapter",
// 			SenderClientName: currentChainName,
// 			Handler:          "FindCosmosToEvmCallContractApproved",
// 			Data:             event,
// 		}, nil

// 	case *contracts.IAxelarGatewayContractCall:
// 		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayContractCall](args, currentChainName, log)
// 		if err != nil {
// 			return types.EventEnvelope{}, err
// 		}
// 		return types.EventEnvelope{
// 			Component:        "DbAdapter",
// 			SenderClientName: currentChainName,
// 			Handler:          "CreateEvmCallContractEvent",
// 			Data:             event,
// 		}, nil

// 	case *contracts.IAxelarGatewayExecuted:
// 		event, err := parseEventArgsIntoEvent[*contracts.IAxelarGatewayExecuted](args, currentChainName, log)
// 		if err != nil {
// 			return types.EventEnvelope{}, err
// 		}
// 		return types.EventEnvelope{
// 			Component:        "DbAdapter",
// 			SenderClientName: currentChainName,
// 			Handler:          "CreateEvmExecutedEvent",
// 			Data:             event,
// 		}, nil

// 	default:
// 		return types.EventEnvelope{}, fmt.Errorf("unknown event type: %T", eventArgs)
// 	}
// }

// parseAnyEvent is a generic function that parses any EVM event into a standardized EvmEvent structure
func parseEvmEventContractCallApproved[T *contracts.IAxelarGatewayContractCallApproved](
	currentChainName string,
	log *eth_types.Log,
) (*EvmEvent[T], error) {
	eventArgs, err := parseContractCallApproved(log)
	if err != nil {
		return nil, err
	}

	event := createEvmEventFromArgs[T](eventArgs, log)

	return event, nil
}

func parseEvmEventContractCall[T *contracts.IAxelarGatewayContractCall](
	currentChainName string,
	log *eth_types.Log,
) (*EvmEvent[T], error) {
	eventArgs, err := parseContractCall(log)
	if err != nil {
		return nil, err
	}

	event := createEvmEventFromArgs[T](eventArgs, log)

	return event, nil
}

func parseEvmEventExecute[T *contracts.IAxelarGatewayExecuted](
	log *eth_types.Log,
) (*EvmEvent[T], error) {
	eventArgs, err := parseExecute(log)
	if err != nil {
		return nil, err
	}

	event := createEvmEventFromArgs[T](eventArgs, log)
	return event, nil
}
