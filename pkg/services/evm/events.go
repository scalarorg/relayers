package evm

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
)

// EvmListenerEvent represents a generic event listener interface
type EvmListenerEvent[T interface{}, E types.Log] struct {
	Name            string
	GetEventFilter  func(gateway *contracts.IAxelarGateway) interface{}
	IsAcceptedChain func(allowedChainIds []string, event T) bool
	ParseEvent      func(currentChainName string, provider *ethclient.Client, event E, finalityBlocks uint64) (*EvmEvent[T], error)
}

// EvmEvent represents a generic event from an EVM chain with type parameter T
type EvmEvent[T any] struct {
	Hash             string
	BlockNumber      uint64
	LogIndex         uint
	SourceChain      string
	DestinationChain string
	WaitForFinality  func() (*types.Receipt, error)
	Args             T
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
