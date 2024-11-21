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
	var relayData models.RelayData

	result := db.PostgresClient.
		Preload("CallContract").
		Where("call_contract.payload_hash = ?", strings.ToLower(payloadHash)).
		First(&relayData)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find payload by hash: %w", result.Error)
	}

	if len(relayData.CallContract.Payload) == 0 {
		return nil, fmt.Errorf("payload not found")
	}

	return relayData.CallContract.Payload, nil
}

func (db *DatabaseAdapter) FindRelayDataByContractCall(contractCall *models.CallContract) ([]models.RelayData, error) {
	var relayData models.RelayData

	result := db.PostgresClient.
		Preload("CallContract").
		Select("call_contract", "id").
		Where("call_contract = ? and status IN ?",
			contractCall, []int{int(PENDING), int(APPROVED)}).
		First(&relayData)

	if result.Error != nil {
		return []models.RelayData{relayData}, fmt.Errorf("failed to find relay data by contract call: %w", result.Error)
	}

	return []models.RelayData{relayData}, nil

}
