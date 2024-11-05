package evm

/*
// EvmListenerEvent represents a generic event listener interface
type EvmListenerEvent[T interface{}, E types.Log] struct {
	Name            string
	GetEventFilter  func(gateway *contracts.IAxelarGateway) interface{}
	IsAcceptedChain func(allowedChainIds []string, event T) bool
	ParseEvent      func(currentChainName string, provider *ethclient.Client, event E, finalityBlocks uint64) (*EvmEvent[T], error)
}

// ContractCallEvent implementation
var EvmContractCallEvent = EvmListenerEvent[*contracts.IAxelarGatewayContractCall, types.Log]{
	Name: "ContractCall",
	GetEventFilter: func(gateway *contracts.IAxelarGateway) interface{} {
		return bind.FilterOpts{}
	},
	IsAcceptedChain: func(allowedDestChainIds []string, event *contracts.IAxelarGatewayContractCall) bool {
		return contains(allowedDestChainIds, strings.ToLower(event.DestinationChain))
	},
	ParseEvent: parseAnyEvent[*contracts.IAxelarGatewayContractCall],
}

// ContractCallWithTokenEvent implementation
var EvmContractCallWithTokenEvent = EvmListenerEvent[*contracts.IAxelarGatewayContractCallWithToken, types.Log]{
	Name: "ContractCallWithToken",
	GetEventFilter: func(gateway *contracts.IAxelarGateway) interface{} {
		return bind.FilterOpts{}
	},
	IsAcceptedChain: func(allowedDestChainIds []string, event *contracts.IAxelarGatewayContractCallWithToken) bool {
		return contains(allowedDestChainIds, strings.ToLower(event.DestinationChain))
	},
	ParseEvent: parseAnyEvent[*contracts.IAxelarGatewayContractCallWithToken],
}

// ContractCallApprovedEvent implementation
var EvmContractCallApprovedEvent = EvmListenerEvent[*contracts.IAxelarGatewayContractCallApproved, types.Log]{
	Name: "ContractCallApproved",
	GetEventFilter: func(gateway *contracts.IAxelarGateway) interface{} {
		return bind.FilterOpts{}
	},
	IsAcceptedChain: func(allowedSrcChainIds []string, event *contracts.IAxelarGatewayContractCallApproved) bool {
		return contains(allowedSrcChainIds, strings.ToLower(event.SourceChain))
	},
	ParseEvent: parseAnyEvent[*contracts.IAxelarGatewayContractCallApproved],
}

// ContractCallWithTokenApprovedEvent implementation
var EvmContractCallWithTokenApprovedEvent = EvmListenerEvent[*contracts.IAxelarGatewayContractCallApprovedWithMint, types.Log]{
	Name: "ContractCallWithTokenApproved",
	GetEventFilter: func(gateway *contracts.IAxelarGateway) interface{} {
		return bind.FilterOpts{}
	},
	IsAcceptedChain: func(allowedSrcChainIds []string, event *contracts.IAxelarGatewayContractCallApprovedWithMint) bool {
		return contains(allowedSrcChainIds, strings.ToLower(event.SourceChain))
	},
	ParseEvent: parseAnyEvent[*contracts.IAxelarGatewayContractCallApprovedWithMint],
}

// ExecutedEvent implementation
var EvmExecutedEvent = EvmListenerEvent[*contracts.IAxelarGatewayExecuted, types.Log]{
	Name: "Executed",
	GetEventFilter: func(gateway *contracts.IAxelarGateway) interface{} {
		return bind.FilterOpts{}
	},
	IsAcceptedChain: func(allowedChainIds []string, event *contracts.IAxelarGatewayExecuted) bool {
		return true
	},
	ParseEvent: parseAnyEvent[*contracts.IAxelarGatewayExecuted],
}

// Helper function to check if a string exists in a slice
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

type EventSubject[T any] struct {
	subscribers []func(*EvmEvent[T])
}

func NewEventSubject[T any]() *EventSubject[T] {
	return &EventSubject[T]{
		subscribers: make([]func(*EvmEvent[T]), 0),
	}
}

func (s *EventSubject[T]) Subscribe(handler func(*EvmEvent[T])) {
	s.subscribers = append(s.subscribers, handler)
}

func (s *EventSubject[T]) Next(event *EvmEvent[T]) {
	for _, subscriber := range s.subscribers {
		subscriber(event)
	}
}

type ContractCallApproved struct {
	EventTypeName string
	Event         *contracts.IAxelarGatewayContractCallApproved
}
*/
