package models

import (
	"encoding/hex"
	"fmt"

	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

func (e *ContractCallApproved) BindCallContractApprovedFromScalarEvent(event *types.ContractCallApproved) {
	e.EventID = string(event.EventID)
	e.SourceChain = string(event.Chain)
	e.CommandID = hex.EncodeToString(event.CommandID[:])
	e.Sender = event.Sender
	e.DestinationChain = string(event.DestinationChain)
	e.ContractAddress = event.ContractAddress
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
}
func (e *ContractCallApproved) BindCallContractApprovedFromEvmEvent(destinationChain string, event *contracts.IScalarGatewayContractCallApproved) {
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	e.EventID = eventId
	e.SourceChain = string(event.SourceChain)
	e.CommandID = hex.EncodeToString(event.CommandId[:])
	e.Sender = event.SourceAddress
	e.DestinationChain = string(destinationChain)
	e.ContractAddress = string(event.SourceAddress)
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
}

func (e *ContractCallApprovedWithMint) BindCallContractApprovedWithMintFromScalarEvent(event *types.EventContractCallWithMintApproved) {
	e.EventID = string(event.EventID)
	e.SourceChain = string(event.Chain)
	e.CommandID = hex.EncodeToString(event.CommandID[:])
	e.Sender = event.Sender
	e.DestinationChain = string(event.DestinationChain)
	e.ContractAddress = event.ContractAddress
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
	e.Amount = event.Asset.Amount.Uint64()
	e.Symbol = event.Asset.Denom
}

func (e *ContractCallApprovedWithMint) BindCallContractApprovedWithMintFromEvmEvent(destinationChain string, event *contracts.IScalarGatewayContractCallApprovedWithMint) {
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	e.EventID = eventId
	e.SourceChain = string(event.SourceChain)
	e.CommandID = hex.EncodeToString(event.CommandId[:])
	e.Sender = event.SourceAddress
	e.DestinationChain = string(destinationChain)
	e.ContractAddress = string(event.SourceAddress)
	e.PayloadHash = hex.EncodeToString(event.PayloadHash[:])
}

func (e *TokenSentApproved) BindTokenSentApprovedFromScalarEvent(event *types.EventTokenSent) {
	e.EventID = string(event.EventID)
	e.TransferID = uint64(event.TransferID)
	e.SourceChain = string(event.Chain)
	e.SourceAddress = event.Sender
	e.DestinationChain = string(event.DestinationChain)
	e.DestinationAddress = event.DestinationAddress
	e.Amount = event.Asset.Amount.Uint64()
	e.Symbol = event.Asset.Denom
}
func (cmd *MintCommand) BindMintCommandFromScalarEvent(event *types.MintCommand) {
	cmd.TransferID = uint64(event.TransferID)
	cmd.CommandID = hex.EncodeToString(event.CommandID[:])
	cmd.SourceChain = string(event.Chain)
	cmd.DestinationChain = string(event.DestinationChain)
	cmd.Recipient = event.DestinationAddress
	cmd.Amount = event.Asset.Amount.Int64()
	cmd.Symbol = event.Asset.Denom
}
