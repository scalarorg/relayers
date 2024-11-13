package evm

import (
	"encoding/hex"
	"fmt"
	"strings"

	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db/models"
)

func (c *EvmClient) ContractCallEvent2RelayData(event *contracts.IAxelarGatewayContractCall) (models.RelayData, error) {
	id := strings.ToLower(fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index))
	senderAddress := event.Sender.String()
	relayData := models.RelayData{
		ID:   id,
		From: c.config.ChainID,
		To:   event.DestinationChain,
		CallContract: &models.CallContract{
			TxHash:        event.Raw.TxHash.String(),
			BlockNumber:   event.Raw.BlockNumber,
			LogIndex:      event.Raw.Index,
			SenderAddress: &senderAddress,
			//3 follows field are used for query to get back payload, so need to convert to lower case
			ContractAddress: strings.ToLower(event.DestinationContractAddress),
			SourceAddress:   strings.ToLower(senderAddress),
			PayloadHash:     strings.ToLower(hex.EncodeToString(event.PayloadHash[:])),
			Payload:         event.Payload,
			//Use for bitcoin vault tx only
			StakerPublicKey: nil,
			Symbol:          "",
		},
	}
	return relayData, nil
}

func (c *EvmClient) ContractCallApprovedEvent2Model(event *contracts.IAxelarGatewayContractCallApproved) (models.CallContractApproved, error) {
	txHash := event.Raw.TxHash.String()
	id := strings.ToLower(fmt.Sprintf("%s-%d-%d", txHash, event.SourceEventIndex, event.Raw.Index))
	sourceEventIndex := uint64(0)
	if event.SourceEventIndex != nil && event.SourceEventIndex.IsUint64() {
		sourceEventIndex = event.SourceEventIndex.Uint64()
	}
	relayData := models.CallContractApproved{
		ID:               id,
		SourceChain:      event.SourceChain,
		DestinationChain: c.config.ChainID,
		TxHash:           strings.ToLower(txHash),
		BlockNumber:      event.Raw.BlockNumber,
		LogIndex:         event.Raw.Index,
		CommandId:        hex.EncodeToString(event.CommandId[:]),
		SourceAddress:    strings.ToLower(event.SourceAddress),
		ContractAddress:  strings.ToLower(event.ContractAddress.String()),
		PayloadHash:      strings.ToLower(hex.EncodeToString(event.PayloadHash[:])),
		SourceTxHash:     strings.ToLower(hex.EncodeToString(event.SourceTxHash[:])),
		SourceEventIndex: sourceEventIndex,
	}
	return relayData, nil
}

func (c *EvmClient) CommandExecutedEvent2Model(event *contracts.IAxelarGatewayExecuted) (models.CommandExecuted, error) {
	id := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	cmdExecuted := models.CommandExecuted{
		ID:               id,
		SourceChain:      c.config.ChainID,
		DestinationChain: c.config.ChainID,
		TxHash:           strings.ToLower(event.Raw.TxHash.String()),
		BlockNumber:      uint64(event.Raw.BlockNumber),
		LogIndex:         uint(event.Raw.Index),
		CommandId:        hex.EncodeToString(event.CommandId[:]),
		Status:           0,
		ReferenceTxHash:  nil,
	}
	return cmdExecuted, nil
}
