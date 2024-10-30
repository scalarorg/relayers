package db

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/services/evm"
	"github.com/scalarorg/relayers/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
)

func (dbClient *DatabaseClient) CreateBtcCallContractEvent(event *types.BtcEventTransaction) error {
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

	result := dbClient.postgresClient.Create(&relayData)
	return result.Error
}

func (dbClient *DatabaseClient) FindRelayDataById(id string, includeCallContract, includeCallContractWithToken *bool) (*models.RelayData, error) {
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

	query := dbClient.postgresClient.Where("id = ?", id)

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
func (dbClient *DatabaseClient) UpdateEventStatusWithPacketSequence(id string, status types.Status, sequence *int) error {
	data := models.RelayData{
		Status:         int(status),
		PacketSequence: sequence,
	}

	result := dbClient.postgresClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return result.Error
	}

	log.Info().Msgf("[DBUpdate] %v", data)
	return nil
}

func (dbClient *DatabaseClient) CreateEvmCallContractEvent(event evm.EvmEvent[*contracts.IAxelarGatewayContractCall]) error {
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

	result := dbClient.postgresClient.Create(&data)
	return result.Error
}

func (dbClient *DatabaseClient) FindCosmosToEvmCallContractApproved(event evm.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]) ([]struct {
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

	result := dbClient.postgresClient.
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

func (dbClient *DatabaseClient) CreateEvmContractCallApprovedEvent(event evm.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]) error {
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

	result := dbClient.postgresClient.Create(&data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseClient] Create DB with error: %v", result.Error)
		return result.Error
	}

	log.Debug().Msgf("[DatabaseClient] Create DB result: %v", data)
	return nil
}

func (dbClient *DatabaseClient) UpdateEventStatus(id string, status types.Status) error {
	data := models.RelayData{
		Status: int(status),
	}

	result := dbClient.postgresClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		return result.Error
	}

	log.Info().Msgf("[DBUpdate] %v", data)
	return nil
}

func (dbClient *DatabaseClient) CreateEvmExecutedEvent(event evm.EvmEvent[*contracts.IAxelarGatewayExecuted]) error {
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

	result := dbClient.postgresClient.Create(&data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseClient] Create DB with error: %v", result.Error)
		return result.Error
	}

	log.Debug().Msgf("[DatabaseClient] Create DB result: %v", data)
	return nil
}

func (dbClient *DatabaseClient) GetBurningTx(payloadHash string) (string, error) {
	var relayData models.RelayData

	result := dbClient.postgresClient.
		Preload("CallContract").
		Where("call_contract.payload_hash = ?", strings.ToLower(payloadHash)).
		First(&relayData)

	if result.Error != nil {
		return "", result.Error
	}

	if relayData.CallContract == nil || len(relayData.CallContract.Payload) == 0 {
		return "", fmt.Errorf("PayloadNotFound")
	}

	return hex.EncodeToString(relayData.CallContract.Payload), nil
}

func (dbClient *DatabaseClient) GetChainConfig(sourceChain string) (models.ChainConfig, error) {
	var chainConfig models.ChainConfig

	result := dbClient.mongoDatabase.Collection("dapps").FindOne(context.Background(), bson.M{
		"chain_name": sourceChain,
	})

	if err := result.Decode(&chainConfig); err != nil {
		return models.ChainConfig{}, fmt.Errorf("chain not found: %w", err)
	}

	if chainConfig.RpcUrl == "" {
		return models.ChainConfig{}, fmt.Errorf("RPC URL not found")
	}

	if chainConfig.AccessToken == "" {
		return models.ChainConfig{}, fmt.Errorf("access token not found")
	}

	return chainConfig, nil
}

func (dbClient *DatabaseClient) UpdateCosmosToEvmEvent(event interface{}, txHash *string) error {
	var messageID string
	switch e := event.(type) {
	case *types.IBCEvent[types.ContractCallSubmitted]:
		messageID = e.Args.MessageID
	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
		messageID = e.Args.MessageID
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}
	// If no transaction hash is provided, return early
	if txHash == nil {
		return nil
	}

	lowercaseHash := strings.ToLower(*txHash)
	data := models.RelayData{
		ID:          messageID,
		ExecuteHash: &lowercaseHash,
		Status:      int(types.APPROVED),
	}

	result := dbClient.postgresClient.Model(&models.RelayData{}).Where("id = ?", messageID).Updates(data)
	if result.Error != nil {
		return result.Error
	}

	log.Info().Msgf("[DatabaseClient] [Evm ContractCallApproved]: %v", data)
	return nil
}

func (dbClient *DatabaseClient) HandleMultipleEvmToBtcEventsTx(event interface{}, tx *btcjson.GetTransactionResult, refTxHash, batchedCommandId string) error {
	// Extract common fields based on event type
	var sourceChain, destinationChain, messageID, sender, contractAddress, payloadHash, hash string
	var messageIDIndex int

	switch e := event.(type) {
	case *types.IBCEvent[types.ContractCallSubmitted]:
		sourceChain = e.Args.SourceChain
		destinationChain = e.Args.DestinationChain
		messageID = e.Args.MessageID
		sender = e.Args.Sender
		contractAddress = e.Args.ContractAddress
		payloadHash = e.Args.PayloadHash
		hash = e.Hash
		parts := strings.Split(messageID, "-")
		if len(parts) > 1 {
			if idx, err := strconv.Atoi(parts[1]); err == nil {
				messageIDIndex = idx
			}
		}
	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
		sourceChain = e.Args.SourceChain
		destinationChain = e.Args.DestinationChain
		messageID = e.Args.MessageID
		sender = e.Args.Sender
		contractAddress = e.Args.ContractAddress
		payloadHash = e.Args.PayloadHash
		hash = e.Hash
		parts := strings.Split(messageID, "-")
		if len(parts) > 1 {
			if idx, err := strconv.Atoi(parts[1]); err == nil {
				messageIDIndex = idx
			}
		}
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}

	if tx == nil {
		return nil
	}

	// Use the extracted fields in your data structures
	TxIDLower := strings.ToLower(tx.TxID)
	contractCallData := models.RelayData{
		ExecuteHash: &TxIDLower,
		Status:      int(types.SUCCESS),
	}

	contractCallApprovedData := models.CallContractApproved{
		ID:               messageID,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		TxHash:           strings.ToLower(hash),
		BlockNumber:      int(tx.BlockIndex),
		LogIndex:         0,
		CommandId:        batchedCommandId,
		SourceAddress:    sender,
		ContractAddress:  strings.ToLower(contractAddress),
		PayloadHash:      strings.ToLower(payloadHash),
		SourceTxHash:     strings.ToLower(hash),
		SourceEventIndex: messageIDIndex,
	}

	amount := strconv.FormatFloat(tx.Amount, 'f', -1, 64)
	executedData := models.CommandExecuted{
		ID:               messageID,
		SourceChain:      sourceChain,
		DestinationChain: destinationChain,
		TxHash:           strings.ToLower(tx.TxID),
		BlockNumber:      int(tx.BlockIndex),
		LogIndex:         0,
		CommandId:        batchedCommandId,
		Status:           int(types.SUCCESS),
		ReferenceTxHash:  &refTxHash,
		Amount:           &amount,
	}

	// Begin transaction
	txDbClient := dbClient.postgresClient.Begin()
	if txDbClient.Error != nil {
		return txDbClient.Error
	}

	// Update RelayData
	if err := txDbClient.Model(&models.RelayData{}).Where("id = ?", messageID).Updates(contractCallData).Error; err != nil {
		txDbClient.Rollback()
		return err
	}

	// Create CallContractApproved
	if err := txDbClient.Create(&contractCallApprovedData).Error; err != nil {
		txDbClient.Rollback()
		return err
	}

	// Create CommandExecuted
	if err := txDbClient.Create(&executedData).Error; err != nil {
		txDbClient.Rollback()
		return err
	}

	// Commit transaction
	if err := txDbClient.Commit().Error; err != nil {
		return err
	}

	log.Info().Msgf("[DatabaseClient] [HandleMultipleEvmToBtcEventsTx]: %+v", executedData)
	return nil
}
