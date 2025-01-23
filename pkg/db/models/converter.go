package models

import (
	"encoding/hex"
	"strings"

	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

func EventTokenSent2Model(event *types.EventTokenSent) scalarnet.TokenSentApproved {
	eventId := strings.TrimPrefix(string(event.EventID), "0x")
	commandId := strings.TrimPrefix(string(event.CommandID), "0x")
	model := scalarnet.TokenSentApproved{
		EventID:            eventId,
		CommandId:          commandId,
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
