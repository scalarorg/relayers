package evm

import (
	"encoding/hex"
	"fmt"
	"strings"

	chains "github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db/models"
)

func (c *EvmClient) ContractCallEvent2Model(event *contracts.IScalarGatewayContractCall) (chains.ContractCall, error) {
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
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	eventId = strings.TrimPrefix(eventId, "0x")
	senderAddress := event.Sender.String()
	contractCall := chains.ContractCall{
		EventId:     eventId,
		TxHash:      event.Raw.TxHash.String(),
		BlockNumber: event.Raw.BlockNumber,
		LogIndex:    event.Raw.Index,
		SourceChain: c.evmConfig.GetId(),
		//3 follows field are used for query to get back payload, so need to convert to lower case
		DestinationChain:    event.DestinationChain,
		DestContractAddress: strings.ToLower(event.DestinationContractAddress),
		SourceAddress:       strings.ToLower(senderAddress),
		PayloadHash:         strings.ToLower(hex.EncodeToString(event.PayloadHash[:])),
		Payload:             event.Payload,
	}
	return contractCall, nil
}

func (c *EvmClient) ContractCallWithToken2Model(event *contracts.IScalarGatewayContractCallWithToken) (chains.ContractCallWithToken, error) {
	eventId := strings.ToLower(fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index))
	senderAddress := event.Sender.String()
	callContract := chains.ContractCall{
		EventId:     eventId,
		TxHash:      event.Raw.TxHash.String(),
		BlockNumber: event.Raw.BlockNumber,
		LogIndex:    event.Raw.Index,
		SourceChain: c.evmConfig.GetId(),
		//3 follows field are used for query to get back payload, so need to convert to lower case
		DestinationChain:    event.DestinationChain,
		DestContractAddress: strings.ToLower(event.DestinationContractAddress),
		SourceAddress:       strings.ToLower(senderAddress),
		PayloadHash:         strings.ToLower(hex.EncodeToString(event.PayloadHash[:])),
		Payload:             event.Payload,
		//Use for bitcoin vault tx only
		StakerPublicKey: nil,
	}
	//Todo: get token contract address from scalar-core by source chain and token symbol
	tokenContractAddress := "" //strings.ToLower(event.TokenContractAddress.String())
	contractCallWithToken := chains.ContractCallWithToken{
		ContractCall:         callContract,
		TokenContractAddress: tokenContractAddress,
		Symbol:               event.Symbol,
		Amount:               event.Amount.Uint64(),
	}
	return contractCallWithToken, nil
}

func (c *EvmClient) TokenSentEvent2Model(event *contracts.IScalarGatewayTokenSent) (chains.TokenSent, error) {
	eventId := fmt.Sprintf("%s-%d", event.Raw.TxHash.String(), event.Raw.Index)
	eventId = strings.TrimPrefix(eventId, "0x")
	senderAddress := event.Sender.String()
	tokenSent := chains.TokenSent{
		EventID:     eventId,
		SourceChain: c.evmConfig.GetId(),
		TxHash:      strings.TrimPrefix(event.Raw.TxHash.String(), "0x"),
		BlockNumber: event.Raw.BlockNumber,
		LogIndex:    event.Raw.Index,
		//3 follows field are used for query to get back payload, so need to convert to lower case
		SourceAddress:        strings.ToLower(senderAddress),
		DestinationChain:     event.DestinationChain,
		DestinationAddress:   strings.ToLower(event.DestinationAddress),
		Symbol:               event.Symbol,
		TokenContractAddress: c.GetTokenContractAddressFromSymbol(event.Symbol, c.evmConfig.GetId()),
		Amount:               event.Amount.Uint64(),
		Status:               chains.TokenSentStatusPending,
	}
	return tokenSent, nil
}

// Todo: Implement this function
func (c *EvmClient) GetTokenContractAddressFromSymbol(symbol string, sourceChain string) string {
	return ""
}
func (c *EvmClient) ContractCallApprovedEvent2Model(event *contracts.IScalarGatewayContractCallApproved) (models.ContractCallApproved, error) {
	txHash := event.Raw.TxHash.String()
	eventId := strings.ToLower(fmt.Sprintf("%s-%d-%d", txHash, event.SourceEventIndex, event.Raw.Index))
	sourceEventIndex := uint64(0)
	if event.SourceEventIndex != nil && event.SourceEventIndex.IsUint64() {
		sourceEventIndex = event.SourceEventIndex.Uint64()
	}
	record := models.ContractCallApproved{
		EventID:          eventId,
		SourceChain:      event.SourceChain,
		DestinationChain: c.evmConfig.GetId(),
		TxHash:           strings.ToLower(txHash),
		CommandID:        hex.EncodeToString(event.CommandId[:]),
		Sender:           strings.ToLower(event.SourceAddress),
		ContractAddress:  strings.ToLower(event.ContractAddress.String()),
		PayloadHash:      strings.ToLower(hex.EncodeToString(event.PayloadHash[:])),
		SourceTxHash:     strings.ToLower(hex.EncodeToString(event.SourceTxHash[:])),
		SourceEventIndex: sourceEventIndex,
	}
	return record, nil
}

func (c *EvmClient) CommandExecutedEvent2Model(event *contracts.IScalarGatewayExecuted) chains.CommandExecuted {
	cmdExecuted := chains.CommandExecuted{
		SourceChain: c.evmConfig.GetId(),
		Address:     event.Raw.Address.String(),
		TxHash:      strings.ToLower(event.Raw.TxHash.String()),
		BlockNumber: uint64(event.Raw.BlockNumber),
		LogIndex:    uint(event.Raw.Index),
		CommandId:   hex.EncodeToString(event.CommandId[:]),
	}
	return cmdExecuted
}
