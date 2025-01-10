package db

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/db/models"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) CreateRelayDatas(datas []models.RelayData, lastCheckpoint *models.EventCheckPoint) error {
	log.Info().Msgf("Creating %d relay data records", len(datas))
	//Up date checkpoint and relayDatas in a transaction
	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		result := tx.CreateInBatches(&datas, 100)
		if result.Error != nil {
			return result.Error
		}
		if lastCheckpoint != nil {
			UpdateLastEventCheckPoint(tx, lastCheckpoint)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create relay data: %w", err)
	}
	return nil
}

func (db *DatabaseAdapter) updateRelayData(id string, data interface{}) (tx *gorm.DB) {
	result := db.PostgresClient.Model(&models.RelayData{}).Where("id = ?", id).Updates(data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseAdapter] [updateRelayData] %s: %v", id, result.Error)
	}
	log.Debug().Msgf("[DatabaseAdapter] [updateRelayData] %s: %v", id, data)
	return result
}

// TODO: Find any better way to update batch relay data status
func (db *DatabaseAdapter) UpdateBatchRelayDataStatus(data []RelaydataExecuteResult, batchSize int) error {
	// Handle empty data case
	if len(data) == 0 {
		return nil
	}

	// Process updates in batches
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(data); i += batchSize {
			end := i + batchSize
			if end > len(data) {
				end = len(data)
			}

			batch := data[i:end]
			for _, item := range batch {
				updates := models.RelayData{
					Status: int(item.Status),
				}

				result := tx.Model(&models.RelayData{}).
					Where("id = ?", item.RelayDataId).
					Updates(updates)

				if result.Error != nil {
					return fmt.Errorf("failed to update relay data batch: %w", result.Error)
				}
			}
		}
		return nil
	})
}

// --- For Setup and Run Evm and Cosmos Relayer ---
func (db *DatabaseAdapter) UpdateRelayDataStatueWithPacketSequence(id string, status RelayDataStatus, sequence *int) error {
	data := models.RelayData{
		Status:         int(status),
		PacketSequence: sequence,
	}
	updateResult := db.updateRelayData(id, data)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}

func (db *DatabaseAdapter) UpdateRelayDataStatueWithExecuteHash(id string, status RelayDataStatus, executeHash *string) error {
	data := models.RelayData{
		Status:      int(status),
		ExecuteHash: executeHash,
	}
	updateResult := db.updateRelayData(id, data)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}

func (db *DatabaseAdapter) FindRelayDataById(id string, option *QueryOptions) (*models.RelayData, error) {
	var relayData models.RelayData

	query := db.PostgresClient.Where("id = ?", id)

	// Add preload conditions based on options
	if option != nil && option.IncludeCallContract != nil && *option.IncludeCallContract {
		query = query.Preload("CallContract")
	}
	if option != nil && option.IncludeCallContractWithToken != nil && *option.IncludeCallContractWithToken {
		query = query.Preload("CallContractWithToken")
	}

	result := query.First(&relayData)
	if result.Error != nil {
		return nil, result.Error
	}

	return &relayData, nil
}

func (db *DatabaseAdapter) FindPayloadByHash(payloadHash string) ([]byte, error) {
	var contractCall models.CallContract
	log.Debug().Str("payloadHash", payloadHash).Msg("[DatabaseAdapter] [FindPayloadByHash] finding payload by hash")
	result := db.PostgresClient.
		Where("payload_hash = ?", strings.ToLower(payloadHash)).
		First(&contractCall)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find payload by hash: %w", result.Error)
	}

	return contractCall.Payload, nil
}

// Find Realaydata by ContractAddress, SourceAddress, PayloadHash
func (db *DatabaseAdapter) FindRelayDataByContractCall(contractCall *models.CallContract) ([]models.RelayData, error) {
	var relayDatas []models.RelayData
	result := db.PostgresClient.
		Joins("CallContract").
		Where("contract_address = ? AND source_address = ? AND payload_hash = ?",
			strings.ToLower(contractCall.DestContractAddress),
			strings.ToLower(contractCall.SourceAddress),
			strings.ToLower(contractCall.PayloadHash)).
		Where("status IN ?", []int{int(PENDING), int(APPROVED)}).
		Preload("CallContract").
		Find(&relayDatas)

	if result.Error != nil {
		return relayDatas, fmt.Errorf("find relaydatas by contract call with error: %w", result.Error)
	}
	if len(relayDatas) == 0 {
		log.Warn().Str("contractAddress", contractCall.DestContractAddress).Str("sourceAddress", contractCall.SourceAddress).Str("payloadHash", contractCall.PayloadHash).Msg("[DatabaseAdapter] [FindRelayDataByContractCall] no relaydata found")
	}
	return relayDatas, nil

}
