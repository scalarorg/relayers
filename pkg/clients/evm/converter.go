package evm

import (
	"encoding/hex"
	"fmt"
	"strings"

	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	relaydata "github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
)

func (c *EvmClient) ContractCallEvent2RelayData(event *contracts.IScalarGatewayContractCall) (models.RelayData, error) {
	//id := strings.ToLower(fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index))
	//Calculate eventId by Txhash-logIndex among logs in txreceipt (AxelarEvmModule)
	//https://github.com/scalarorg/scalar-core/blob/main/vald/evm/gateway_tx_confirmation.go#L73
	//Dec 30, use logIndex directly to avoid redundant request. This must aggrees with the scalar-core vald module
	// receipt, err := c.Client.TransactionReceipt(context.Background(), event.Raw.TxHash)
	// if err != nil {
	// 	return models.RelayData{}, fmt.Errorf("failed to get transaction receipt: %w", err)
	// }
	// var id string
	// for ind, log := range receipt.Logs {
	// 	if log.Index == event.Raw.Index {
	// 		id = fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), ind)
	// 		break
	// 	}
	// }
	id := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	senderAddress := event.Sender.String()
	relayData := models.RelayData{
		ID:   id,
		From: c.evmConfig.GetId(),
		To:   event.DestinationChain,
		CallContract: &models.CallContract{
			ID:          id,
			TxHash:      event.Raw.TxHash.String(),
			BlockNumber: event.Raw.BlockNumber,
			LogIndex:    event.Raw.Index,
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

func (c *EvmClient) TokenSentEvent2RelayData(event *contracts.IScalarGatewayTokenSent) (models.RelayData, error) {
	//id := strings.ToLower(fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index))
	//Calculate eventId by Txhash-logIndex among logs in txreceipt (AxelarEvmModule)
	//https://github.com/scalarorg/scalar-core/blob/main/vald/evm/gateway_tx_confirmation.go#L73
	//Dec 30, use logIndex directly to avoid redundant request. This must aggrees with the scalar-core vald module
	// receipt, err := c.Client.TransactionReceipt(context.Background(), event.Raw.TxHash)
	// if err != nil {
	// 	return models.RelayData{}, fmt.Errorf("failed to get transaction receipt: %w", err)
	// }
	// var id string
	// for ind, log := range receipt.Logs {
	// 	if log.Index == event.Raw.Index {
	// 		id = fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), ind)
	// 		break
	// 	}
	// }
	id := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	senderAddress := event.Sender.String()
	relayData := models.RelayData{
		ID:   id,
		From: c.evmConfig.GetId(),
		To:   event.DestinationChain,
		TokenSent: &models.TokenSent{
			ID:          id,
			TxHash:      event.Raw.TxHash.String(),
			BlockNumber: event.Raw.BlockNumber,
			LogIndex:    event.Raw.Index,
			//3 follows field are used for query to get back payload, so need to convert to lower case
			SourceAddress:      strings.ToLower(senderAddress),
			DestinationAddress: strings.ToLower(event.DestinationAddress),
			Symbol:             event.Symbol,
			Amount:             event.Amount.Uint64(),
		},
	}
	return relayData, nil
}

func (c *EvmClient) ContractCallApprovedEvent2Model(event *contracts.IScalarGatewayContractCallApproved) (models.CallContractApproved, error) {
	txHash := event.Raw.TxHash.String()
	id := strings.ToLower(fmt.Sprintf("%s-%d-%d", txHash, event.SourceEventIndex, event.Raw.Index))
	sourceEventIndex := uint64(0)
	if event.SourceEventIndex != nil && event.SourceEventIndex.IsUint64() {
		sourceEventIndex = event.SourceEventIndex.Uint64()
	}
	relayData := models.CallContractApproved{
		ID:               id,
		SourceChain:      event.SourceChain,
		DestinationChain: c.evmConfig.GetId(),
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

func (c *EvmClient) CommandExecutedEvent2Model(event *contracts.IScalarGatewayExecuted) models.CommandExecuted {
	id := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	cmdExecuted := models.CommandExecuted{
		ID:               id,
		SourceChain:      c.evmConfig.GetId(),
		DestinationChain: "", //TODO: need to get the destination chain from the db by the commandId
		TxHash:           strings.ToLower(event.Raw.TxHash.String()),
		BlockNumber:      uint64(event.Raw.BlockNumber),
		LogIndex:         uint(event.Raw.Index),
		CommandId:        hex.EncodeToString(event.CommandId[:]),
		Status:           int(relaydata.SUCCESS),
		ReferenceTxHash:  nil,
	}
	return cmdExecuted
}
