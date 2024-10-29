package db

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/services/evm"
	"github.com/scalarorg/relayers/pkg/types"
)

func CreateBtcCallContractEvent(event *types.BtcEventTransaction) error {
	id := fmt.Sprintf("%s-%d", strings.ToLower(event.TxHash), event.LogIndex)

	// Convert VaultTxHex and Payload to byte slices
	txHexBytes, err := hex.DecodeString(strings.TrimPrefix(event.VaultTxHex, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode VaultTxHex: %w", err)
	}
	payloadBytes, err := hex.DecodeString(strings.TrimPrefix(event.Payload, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode Payload: %w", err)
	}

	relayData := models.RelayData{
		ID:   id,
		From: event.SourceChain,
		To:   event.DestinationChain,
		CallContract: &models.CallContract{
			TxHash:          event.TxHash,
			TxHex:           txHexBytes,
			BlockNumber:     new(int),
			LogIndex:        new(int),
			ContractAddress: event.DestinationContractAddress,
			SourceAddress:   event.Sender,
			Amount:          new(string),
			Symbol:          new(string),
			Payload:         payloadBytes,
			PayloadHash:     event.PayloadHash,
			StakerPublicKey: &event.StakerPublicKey,
			SenderAddress:   new(string),
		},
	}

	*relayData.CallContract.BlockNumber = int(event.BlockNumber)
	*relayData.CallContract.LogIndex = int(event.LogIndex)
	*relayData.CallContract.Amount = event.MintingAmount
	*relayData.CallContract.Symbol = "" // Set an empty string for consistency
	*relayData.CallContract.SenderAddress = event.Sender

	log.Debug().
		Str("id", id).
		Msg("[DatabaseClient] Creating BtcCallContract")

	result := DatabaseClient.Create(&relayData)
	return result.Error
}

func FindRelayDataById(id string, includeCallContract, includeCallContractWithToken *bool) (*models.RelayData, error) {
	var relayData models.RelayData

	// Set default values if options are nil
	if includeCallContract == nil {
		defaultTrue := true
		includeCallContract = &defaultTrue
	}
	if includeCallContractWithToken == nil {
		defaultTrue := true
		includeCallContractWithToken = &defaultTrue
	}

	query := DatabaseClient.Where("id = ?", id)

	// Add preload conditions based on options
	if *includeCallContract {
		query = query.Preload("CallContract")
	}
	if *includeCallContractWithToken {
		query = query.Preload("CallContractWithToken")
	}

	result := query.First(&relayData)
	if result.Error != nil {
		return nil, result.Error
	}

	return &relayData, nil
}

// --- For Setup and Run Evm and Cosmos Relayer ---
func UpdateEventStatusWithPacketSequence(id string, status types.Status, sequence *int) error {
	data := models.RelayData{
		Status:         int(status),
		PacketSequence: sequence,
		UpdatedAt:      time.Now(),
	}

	result := DatabaseClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return result.Error
	}

	log.Info().Msgf("[DBUpdate] %v", data)
	return nil
}

func CreateEvmCallContractEvent(event evm.EvmEvent[*contracts.IAxelarGatewayContractCall]) error {
	id := fmt.Sprintf("%s-%d", strings.ToLower(event.Hash), event.LogIndex)
	payloadBytes, err := hex.DecodeString(strings.TrimPrefix(string(event.Args.Payload), "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	data := models.RelayData{
		ID:   id,
		From: event.SourceChain,
		To:   event.DestinationChain,
		CallContract: &models.CallContract{
			TxHash:          strings.ToLower(event.Hash),
			BlockNumber:     new(int),
			Payload:         payloadBytes,
			PayloadHash:     strings.ToLower(hex.EncodeToString(event.Args.PayloadHash[:])),
			ContractAddress: strings.ToLower(event.Args.DestinationContractAddress),
			SourceAddress:   strings.ToLower(event.Args.Sender.String()),
			SenderAddress:   new(string),
			LogIndex:        new(int),
		},
	}

	*data.CallContract.BlockNumber = int(event.BlockNumber)
	*data.CallContract.LogIndex = int(event.LogIndex)
	*data.CallContract.SenderAddress = event.Args.Sender.String()

	log.Debug().Msgf("[DatabaseClient] Create Evm Call Contract Event: %v", data)

	result := DatabaseClient.Create(&data)
	return result.Error
}

func FindCosmosToEvmCallContractApproved(event evm.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]) ([]struct {
	ID      string
	Payload []byte
}, error) {
	var datas []models.RelayData

	// Using the existing RelayData and CallContract models for type safety
	pendingQuery := models.RelayData{
		Status: int(types.PENDING),
		CallContract: &models.CallContract{
			PayloadHash:     strings.ToLower(hex.EncodeToString(event.Args.PayloadHash[:])),
			SourceAddress:   strings.ToLower(event.Args.SourceAddress),
			ContractAddress: strings.ToLower(event.Args.ContractAddress.String()),
		},
	}

	approvedQuery := models.RelayData{
		Status: int(types.APPROVED),
		CallContract: &models.CallContract{
			PayloadHash:     strings.ToLower(hex.EncodeToString(event.Args.PayloadHash[:])),
			SourceAddress:   strings.ToLower(event.Args.SourceAddress),
			ContractAddress: strings.ToLower(event.Args.ContractAddress.String()),
		},
	}

	result := DatabaseClient.
		Preload("CallContract").
		Where(map[string]interface{}{
			"call_contract.payload_hash":     pendingQuery.CallContract.PayloadHash,
			"call_contract.source_address":   pendingQuery.CallContract.SourceAddress,
			"call_contract.contract_address": pendingQuery.CallContract.ContractAddress,
			"status":                         pendingQuery.Status,
		}).Or(map[string]interface{}{
		"call_contract.payload_hash":     approvedQuery.CallContract.PayloadHash,
		"call_contract.source_address":   approvedQuery.CallContract.SourceAddress,
		"call_contract.contract_address": approvedQuery.CallContract.ContractAddress,
		"status":                         approvedQuery.Status,
	}).
		Order("updated_at desc").
		Find(&datas)

	if result.Error != nil {
		return nil, result.Error
	}

	mappedResult := make([]struct {
		ID      string
		Payload []byte
	}, len(datas))

	for i, data := range datas {
		mappedResult[i] = struct {
			ID      string
			Payload []byte
		}{
			ID:      data.ID,
			Payload: data.CallContract.Payload,
		}
	}

	return mappedResult, nil
}

func CreateEvmContractCallApprovedEvent(event evm.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]) error {
	id := fmt.Sprintf("%s-%d-%d", event.Hash, event.Args.SourceEventIndex, event.LogIndex)
	data := models.CallContractApproved{
		ID:               id,
		SourceChain:      event.SourceChain,
		DestinationChain: event.DestinationChain,
		TxHash:           strings.ToLower(event.Hash),
		BlockNumber:      int(event.BlockNumber),
		LogIndex:         int(event.LogIndex),
		CommandId:        hex.EncodeToString(event.Args.CommandId[:]),
		SourceAddress:    strings.ToLower(event.Args.SourceAddress),
		ContractAddress:  strings.ToLower(event.Args.ContractAddress.String()),
		PayloadHash:      strings.ToLower(hex.EncodeToString(event.Args.PayloadHash[:])),
		SourceTxHash:     strings.ToLower(hex.EncodeToString(event.Args.SourceTxHash[:])),
		SourceEventIndex: int(event.Args.SourceEventIndex.Int64()),
	}

	log.Debug().Msgf("[DatabaseClient] Create EvmContractCallApproved: %v", data)

	result := DatabaseClient.Create(&data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseClient] Create DB with error: %v", result.Error)
		return result.Error
	}

	log.Debug().Msgf("[DatabaseClient] Create DB result: %v", data)
	return nil
}

func UpdateEventStatus(id string, status types.Status) error {
	data := models.RelayData{
		Status:    int(status),
		UpdatedAt: time.Now(),
	}

	result := DatabaseClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return result.Error
	}

	log.Info().Msgf("[DBUpdate] %v", data)
	return nil
}

func CreateEvmExecutedEvent(event evm.EvmEvent[*contracts.IAxelarGatewayExecuted]) error {
	id := fmt.Sprintf("%s-%d", event.Hash, event.LogIndex)
	log.Debug().Msgf("[DatabaseClient] Create EvmExecuted: %s", id)

	data := models.CommandExecuted{
		ID:               id,
		SourceChain:      event.SourceChain,
		DestinationChain: event.DestinationChain,
		TxHash:           event.Hash,
		BlockNumber:      int(event.BlockNumber),
		LogIndex:         int(event.LogIndex),
		CommandId:        hex.EncodeToString(event.Args.CommandId[:]),
		Status:           int(types.SUCCESS),
	}

	result := DatabaseClient.Create(&data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseClient] Create DB with error: %v", result.Error)
		return result.Error
	}

	log.Debug().Msgf("[DatabaseClient] Create DB result: %v", data)
	return nil
}
