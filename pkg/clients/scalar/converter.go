package scalar

import (
	"encoding/hex"
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

func CreateCallContractApprovedFromScalarEvent(event *types.ContractCallApproved) scalarnet.ContractCallApproved {
	e := scalarnet.ContractCallApproved{}
	e.EventID = string(event.EventID)
	e.SourceChain = string(event.Chain)
	e.CommandID = hex.EncodeToString(event.CommandID[:])
	e.Sender = event.Sender
	e.DestinationChain = string(event.DestinationChain)
	e.ContractAddress = event.ContractAddress
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
	return e
}
func CreateCallContractApprovedFromEvmEvent(destinationChain string, event *contracts.IScalarGatewayContractCallApproved) scalarnet.ContractCallApproved {
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	e := scalarnet.ContractCallApproved{}
	e.EventID = eventId
	e.SourceChain = string(event.SourceChain)
	e.CommandID = hex.EncodeToString(event.CommandId[:])
	e.Sender = event.SourceAddress
	e.DestinationChain = string(destinationChain)
	e.ContractAddress = string(event.SourceAddress)
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
	return e
}

func CreateCallContractApprovedWithMintFromScalarEvent(event *types.EventContractCallWithMintApproved) scalarnet.ContractCallApprovedWithMint {
	e := scalarnet.ContractCallApprovedWithMint{}
	e.EventID = string(event.EventID)
	e.SourceChain = string(event.Chain)
	e.CommandID = hex.EncodeToString(event.CommandID[:])
	e.Sender = event.Sender
	e.DestinationChain = string(event.DestinationChain)
	e.ContractAddress = event.ContractAddress
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
	e.Amount = event.Asset.Amount.Uint64()
	e.Symbol = event.Asset.Denom
	return e
}

func CreateCallContractApprovedWithMintFromEvmEvent(destinationChain string, event *contracts.IScalarGatewayContractCallApprovedWithMint) scalarnet.ContractCallApprovedWithMint {
	e := scalarnet.ContractCallApprovedWithMint{}
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	e.EventID = eventId
	e.SourceChain = string(event.SourceChain)
	e.CommandID = hex.EncodeToString(event.CommandId[:])
	e.Sender = event.SourceAddress
	e.DestinationChain = string(destinationChain)
	e.ContractAddress = string(event.SourceAddress)
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
	return e
}

func CreateMintCommandFromScalarEvent(event *types.MintCommand) chains.MintCommand {
	e := chains.MintCommand{}
	e.TransferID = uint64(event.TransferID)
	e.CommandID = hex.EncodeToString(event.CommandID[:])
	e.SourceChain = string(event.Chain)
	e.DestinationChain = string(event.DestinationChain)
	e.Recipient = event.DestinationAddress
	e.Amount = event.Asset.Amount.Int64()
	e.Symbol = event.Asset.Denom
	return e
}
