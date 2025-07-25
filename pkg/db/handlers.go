package db

import (
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/pkg/utils"
	"gorm.io/gorm"
)

// func (db *DatabaseAdapter) SaveValues(values any) error {
// 	return db.SaveSingleValueWithCheckpoint(values, nil)
// }

func (db *DatabaseAdapter) SaveValuesWithCheckpoint(values any, lastCheckpoint *scalarnet.EventCheckPoint) error {
	//Up date checkpoint and relayDatas in a transaction
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(values)
		if result.Error != nil {
			return result.Error
		}
		if lastCheckpoint != nil {
			UpdateLastEventCheckPoint(tx, lastCheckpoint)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create Single value with checkpoint: %w", err)
	}
	return nil
}

func (db *DatabaseAdapter) SaveSingleValue(value any) error {
	result := db.RelayerClient.Save(value)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// func (db *DatabaseAdapter) SaveSingleValueWithCheckpoint(value any, lastCheckpoint *models.EventCheckPoint) error {
// 	//Up date checkpoint and relayDatas in a transaction
// 	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
// 		result := tx.Save(value)
// 		if result.Error != nil {
// 			return result.Error
// 		}
// 		if lastCheckpoint != nil {
// 			UpdateLastEventCheckPoint(tx, lastCheckpoint)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to create Single value with checkpoint: %w", err)
// 	}
// 	return nil
// }

func (db *DatabaseAdapter) CreateBatchValue(values any, batchSize int) error {
	result := db.RelayerClient.CreateInBatches(values, batchSize)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Get earliest event check point for all eventNames
func (db *DatabaseAdapter) GetLastCheckPoint(chainName string) (*scalarnet.EventCheckPoint, error) {
	//Default value
	lastBlock := scalarnet.EventCheckPoint{
		ChainName:   chainName,
		EventName:   "",
		BlockNumber: 0,
		TxHash:      "",
		LogIndex:    0,
		EventKey:    "",
	}
	result := db.RelayerClient.Where("chain_name = ?", chainName).Order("block_number ASC").First(&lastBlock)
	return &lastBlock, result.Error
}

func (db *DatabaseAdapter) GetLastEventCheckPoint(chainName, eventName string) (*scalarnet.EventCheckPoint, error) {
	//Default value
	lastBlock := scalarnet.EventCheckPoint{
		ChainName:   chainName,
		EventName:   eventName,
		BlockNumber: 0,
		TxHash:      "",
		LogIndex:    0,
		EventKey:    "",
	}
	result := db.RelayerClient.Where("chain_name = ? AND event_name = ?", chainName, eventName).First(&lastBlock)
	return &lastBlock, result.Error
}
func (db *DatabaseAdapter) UpdateLastEventCheckPoints(checkpoints map[string]scalarnet.EventCheckPoint) error {
	return db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		for _, checkpoint := range checkpoints {
			err := UpdateLastEventCheckPoint(tx, &checkpoint)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DatabaseAdapter) UpdateLastEventCheckPoint(value *scalarnet.EventCheckPoint) error {
	return UpdateLastEventCheckPoint(db.RelayerClient, value)
}

// For transactional update
func UpdateLastEventCheckPoint(db *gorm.DB, value *scalarnet.EventCheckPoint) error {
	//1. Get event check point from db
	storedEventCheckPoint := scalarnet.EventCheckPoint{}
	err := db.Where("chain_name = ? AND event_name = ?", value.ChainName, value.EventName).First(&storedEventCheckPoint).Error
	if err != nil {
		err = db.Create(value).Error
	} else {
		err = db.Model(&scalarnet.EventCheckPoint{}).Where("chain_name = ? AND event_name = ?", value.ChainName, value.EventName).Updates(map[string]interface{}{
			"block_number": value.BlockNumber,
			"tx_hash":      utils.NormalizeHash(value.TxHash),
			"log_index":    value.LogIndex,
			"event_key":    value.EventKey,
		}).Error
	}

	// err := db.Clauses(
	// 	clause.OnConflict{
	// 		Columns: []clause.Column{{Name: "chain_name"}, {Name: "event_name"}},
	// 		DoUpdates: clause.Assignments(map[string]interface{}{
	// 			"block_number": value.BlockNumber,
	// 			"tx_hash":      utils.NormalizeHash(value.TxHash),
	// 			"log_index":    value.LogIndex,
	// 			"event_key":    value.EventKey,
	// 		}),
	// 	},
	// ).Create(value).Error
	if err != nil {
		return fmt.Errorf("failed to update last event check point: %w", err)
	}

	return nil
}

func (db *DatabaseAdapter) UpdateEventStatusWithPacketSequence(id string, status chains.ContractCallStatus, sequence *int) error {
	// data := models.RelayData{
	// 	Status: int(status),
	// }
	return nil
}

// 	result := db.PostgresClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
// 	if result.Error != nil {
// 		return result.Error
// 	}

// 	log.Info().Msgf("[DBUpdate] %v", data)
// 	return nil
// }

// func (db *DatabaseAdapter) CreateEvmExecutedEvent(event *types.EvmEvent[*contracts.IScalarGatewayExecuted]) error {
// 	id := fmt.Sprintf("%s-%d", event.Hash, event.LogIndex)
// 	log.Debug().Msgf("[DatabaseClient] Create EvmExecuted: %s", id)

// 	data := models.CommandExecuted{
// 		ID:               id,
// 		SourceChain:      event.SourceChain,
// 		DestinationChain: event.DestinationChain,
// 		TxHash:           event.Hash,
// 		BlockNumber:      event.BlockNumber,
// 		LogIndex:         event.LogIndex,
// 		CommandId:        hex.EncodeToString(event.Args.CommandId[:]),
// 		Status:           int(types.SUCCESS),
// 	}

// 	result := db.PostgresClient.Create(&data)
// 	if result.Error != nil {
// 		log.Error().Msgf("[DatabaseClient] Create DB with error: %v", result.Error)
// 		return result.Error
// 	}

// 	log.Debug().Msgf("[DatabaseClient] Create DB result: %v", data)
// 	return nil
// }

// func (db *DatabaseAdapter) GetChainConfig(sourceChain string) (models.ChainConfig, error) {
// 	var chainConfig models.ChainConfig

// 	result := db.MongoDatabase.Collection("dapps").FindOne(context.Background(), bson.M{
// 		"chain_name": sourceChain,
// 	})

// 	if err := result.Decode(&chainConfig); err != nil {
// 		return models.ChainConfig{}, fmt.Errorf("chain not found: %w", err)
// 	}

// 	if chainConfig.RpcUrl == "" {
// 		return models.ChainConfig{}, fmt.Errorf("RPC URL not found")
// 	}

// 	if chainConfig.AccessToken == "" {
// 		return models.ChainConfig{}, fmt.Errorf("access token not found")
// 	}

// 	return chainConfig, nil
// }

// func (db *DatabaseAdapter) UpdateCosmosToEvmEvent(event interface{}, txHash *string) error {
// 	var messageID string
// 	switch e := event.(type) {
// 	case *types.IBCEvent[types.ContractCallSubmitted]:
// 		messageID = e.Args.MessageID
// 	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
// 		messageID = e.Args.MessageID
// 	default:
// 		return fmt.Errorf("unsupported event type: %T", event)
// 	}
// 	// If no transaction hash is provided, return early
// 	if txHash == nil {
// 		return nil
// 	}

// 	lowercaseHash := strings.ToLower(*txHash)
// 	data := models.RelayData{
// 		ID:          messageID,
// 		ExecuteHash: &lowercaseHash,
// 		Status:      int(types.APPROVED),
// 	}

// 	result := db.PostgresClient.Model(&models.RelayData{}).Where("id = ?", messageID).Updates(data)
// 	if result.Error != nil {
// 		return result.Error
// 	}

// 	log.Info().Msgf("[DatabaseClient] [Evm ContractCallApproved]: %v", data)
// 	return nil
// }

// func (db *DatabaseAdapter) HandleMultipleEvmToBtcEventsTx(event interface{}, tx *btcjson.GetTransactionResult, refTxHash, batchedCommandId string) error {
// 	// Extract common fields based on event type
// 	var sourceChain, destinationChain, messageID, sender, contractAddress, payloadHash, hash string
// 	var messageIDIndex int

// 	switch e := event.(type) {
// 	case *types.IBCEvent[types.ContractCallSubmitted]:
// 		sourceChain = e.Args.SourceChain
// 		destinationChain = e.Args.DestinationChain
// 		messageID = e.Args.MessageID
// 		sender = e.Args.Sender
// 		contractAddress = e.Args.ContractAddress
// 		payloadHash = e.Args.PayloadHash
// 		hash = e.Hash
// 		parts := strings.Split(messageID, "-")
// 		if len(parts) > 1 {
// 			if idx, err := strconv.Atoi(parts[1]); err == nil {
// 				messageIDIndex = idx
// 			}
// 		}
// 	case *types.IBCEvent[types.ContractCallWithTokenSubmitted]:
// 		sourceChain = e.Args.SourceChain
// 		destinationChain = e.Args.DestinationChain
// 		messageID = e.Args.MessageID
// 		sender = e.Args.Sender
// 		contractAddress = e.Args.ContractAddress
// 		payloadHash = e.Args.PayloadHash
// 		hash = e.Hash
// 		parts := strings.Split(messageID, "-")
// 		if len(parts) > 1 {
// 			if idx, err := strconv.Atoi(parts[1]); err == nil {
// 				messageIDIndex = idx
// 			}
// 		}
// 	default:
// 		return fmt.Errorf("unsupported event type: %T", event)
// 	}

// 	if tx == nil {
// 		return nil
// 	}

// 	// Use the extracted fields in your data structures
// 	TxIDLower := strings.ToLower(tx.TxID)
// 	contractCallData := models.RelayData{
// 		ExecuteHash: &TxIDLower,
// 		Status:      int(types.SUCCESS),
// 	}

// 	contractCallApprovedData := models.CallContractApproved{
// 		ID:               messageID,
// 		SourceChain:      sourceChain,
// 		DestinationChain: destinationChain,
// 		TxHash:           strings.ToLower(hash),
// 		BlockNumber:      uint64(tx.BlockIndex),
// 		LogIndex:         0,
// 		CommandId:        batchedCommandId,
// 		SourceAddress:    sender,
// 		ContractAddress:  strings.ToLower(contractAddress),
// 		PayloadHash:      strings.ToLower(payloadHash),
// 		SourceTxHash:     strings.ToLower(hash),
// 		SourceEventIndex: uint64(messageIDIndex),
// 	}

// 	amount := strconv.FormatFloat(tx.Amount, 'f', -1, 64)
// 	executedData := models.CommandExecuted{
// 		ID:               messageID,
// 		SourceChain:      sourceChain,
// 		DestinationChain: destinationChain,
// 		TxHash:           strings.ToLower(tx.TxID),
// 		BlockNumber:      tx.BlockIndex,
// 		LogIndex:         0,
// 		CommandId:        batchedCommandId,
// 		Status:           int(types.SUCCESS),
// 		ReferenceTxHash:  &refTxHash,
// 		Amount:           &amount,
// 	}

// 	// Begin transaction
// 	txDbClient := db.PostgresClient.Begin()
// 	if txDbClient.Error != nil {
// 		return txDbClient.Error
// 	}

// 	// Update RelayData
// 	if err := txDbClient.Model(&models.RelayData{}).Where("id = ?", messageID).Updates(contractCallData).Error; err != nil {
// 		txDbClient.Rollback()
// 		return err
// 	}

// 	// Create CallContractApproved
// 	if err := txDbClient.Create(&contractCallApprovedData).Error; err != nil {
// 		txDbClient.Rollback()
// 		return err
// 	}

// 	// Create CommandExecuted
// 	if err := txDbClient.Create(&executedData).Error; err != nil {
// 		txDbClient.Rollback()
// 		return err
// 	}

// 	// Commit transaction
// 	if err := txDbClient.Commit().Error; err != nil {
// 		return err
// 	}

// 	log.Info().Msgf("[DatabaseClient] [HandleMultipleEvmToBtcEventsTx]: %+v", executedData)
// 	return nil
// }
