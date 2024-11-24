package parser

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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

func GetScalarGatewayAbi() (*abi.ABI, error) {
	if scalarGatewayAbi == nil {
		var err error
		scalarGatewayAbi, err = contracts.IAxelarGatewayMetaData.GetAbi()
		if err != nil {
			return nil, err
		}
	}
	return scalarGatewayAbi, nil
}

func ParseLogs(receiptLogs []*eth_types.Log) (AllEvmEvents, error) {
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
		if eventArgs, err := parseExecute(reciptLog); err == nil {
			events.Executed = createEvmEventFromArgs(eventArgs, reciptLog)
			continue
		}
	}
	return events, nil
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

func parseContractCallApproved(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayContractCallApproved, error) {
	eventData := struct {
		CommandId        [32]byte
		SourceChain      string
		SourceAddress    string
		ContractAddress  common.Address
		PayloadHash      [32]byte
		SourceTxHash     [32]byte
		SourceEventIndex *big.Int
	}{}

	abi, err := GetScalarGatewayAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	if abi.Events["ContractCallApproved"].ID != receiptLog.Topics[0] {
		return nil, fmt.Errorf("receipt log topic 0 does not match ContractCallApproved event id")
	}
	if err := abi.UnpackIntoInterface(&eventData, "ContractCallApproved", receiptLog.Data); err != nil {
		return nil, fmt.Errorf("failed to unpack event: %w", err)
	}
	//log.Debug().Any("parsedEventData", eventData).Msg("[EvmClient] [parseContractCallApproved]")
	eventArgs := contracts.IAxelarGatewayContractCallApproved{
		SourceChain:      eventData.SourceChain,
		SourceAddress:    eventData.SourceAddress,
		SourceTxHash:     eventData.SourceTxHash,
		SourceEventIndex: eventData.SourceEventIndex,
		Raw:              *receiptLog,
	}
	if len(receiptLog.Topics) >= 1 {
		eventArgs.CommandId = receiptLog.Topics[0]
	}
	if len(receiptLog.Topics) >= 2 {
		eventArgs.ContractAddress = common.BytesToAddress(receiptLog.Topics[1][:])
	}
	if len(receiptLog.Topics) >= 3 {
		eventArgs.PayloadHash = receiptLog.Topics[2]
	}

	log.Debug().Any("parsedEvent", eventArgs).Msg("[EvmClient] [parseContractCallApproved]")

	return &eventArgs, nil
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

func parseContractCall(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayContractCall, error) {
	eventData := struct {
		Sender                     common.Address
		DestinationChain           string
		DestinationContractAddress string
		PayloadHash                [32]byte
		Payload                    []byte
	}{}

	abi, err := GetScalarGatewayAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	event, ok := abi.Events["ContractCall"]
	if !ok {
		return nil, fmt.Errorf("failed to find Executed event in ABI")
	}
	if event.ID != receiptLog.Topics[0] {
		return nil, fmt.Errorf("command id mismatch")
	}
	if len(receiptLog.Topics) < 2 {
		return nil, fmt.Errorf("missing second topic, used as commandId")
	}
	if err := abi.UnpackIntoInterface(&event, "ContractCall", receiptLog.Data); err != nil {
		return nil, fmt.Errorf("failed to unpack event: %w", err)
	}

	var eventArgs contracts.IAxelarGatewayContractCall = contracts.IAxelarGatewayContractCall{
		Sender:                     eventData.Sender,
		DestinationChain:           eventData.DestinationChain,
		DestinationContractAddress: eventData.DestinationContractAddress,
		PayloadHash:                eventData.PayloadHash,
		Payload:                    eventData.Payload,
		Raw:                        *receiptLog,
	}

	log.Debug().Any("parsedEvent", eventArgs).Msg("[EvmClient] [parseContractCall]")

	return &eventArgs, nil
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

func parseExecute(
	receiptLog *eth_types.Log,
) (*contracts.IAxelarGatewayExecuted, error) {
	abi, err := GetScalarGatewayAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	executedEvent, ok := abi.Events["Executed"]
	if !ok {
		return nil, fmt.Errorf("failed to find Executed event in ABI")
	}
	if executedEvent.ID != receiptLog.Topics[0] {
		return nil, fmt.Errorf("command id mismatch")
	}
	if len(receiptLog.Topics) < 2 {
		return nil, fmt.Errorf("missing second topic, used as commandId")
	}

	var eventArgs contracts.IAxelarGatewayExecuted = contracts.IAxelarGatewayExecuted{
		CommandId: receiptLog.Topics[1],
		Raw:       *receiptLog,
	}

	log.Debug().Any("parsedEvent", eventArgs).Msg("[EvmClient] [parseExecute]")

	return &eventArgs, nil
}
