package models

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/scalarorg/data-models/scalarnet"
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

func EventTokenSent2Model(event *types.EventTokenSent) scalarnet.TokenSentApproved {
	eventId := strings.TrimPrefix(string(event.EventID), "0x")
	model := scalarnet.TokenSentApproved{
		EventID:            eventId,
		CommandId:          event.CommandID,
		TransferID:         uint64(event.TransferID),
		SourceChain:        string(event.Chain),
		SourceAddress:      event.Sender,
		DestinationChain:   string(event.DestinationChain),
		DestinationAddress: event.DestinationAddress,
		Amount:             event.Asset.Amount.Uint64(),
		Symbol:             event.Asset.Denom,
	}
	return model
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
