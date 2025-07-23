package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/data-models/scalarnet"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (db *DatabaseAdapter) GetLastContractCallWithToken() (*relayer.ContractCallWithToken, error) {
	var contractCallWithToken relayer.ContractCallWithToken
	err := db.RelayerClient.Model(&relayer.ContractCallWithToken{}).
		Order(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{Column: clause.Column{Name: "tx_hash"}, Desc: true},
				{Column: clause.Column{Name: "log_index"}, Desc: true}},
		}).
		First(&contractCallWithToken).Error
	if err != nil {
		return nil, err
	}
	return &contractCallWithToken, nil
}

func (db *DatabaseAdapter) CreateContractCallWithTokens(contractCallWithTokens []*relayer.ContractCallWithToken) error {
	err := db.RelayerClient.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tx_hash"}, {Name: "log_index"}},
		DoNothing: true,
	}).Create(contractCallWithTokens).Error
	if err != nil {
		return err
	}
	return nil
}

// func (db *DatabaseAdapter) GetUnprocessedContractCallTxsByBlock(blockNumber uint64) ([]*chains.ContractCallWithToken, error) {
// 	var contractCallWithTokens []*chains.ContractCallWithToken
// 	query := `
// 		SELECT * FROM contract_call_with_tokens
// 		WHERE block_number = $1
// 		AND tx_hash NOT IN (
// 			SELECT ce.command_id FROM command_executeds ce
// 			WHERE ce.command_id = contract_call_with_tokens.tx_hash
// 		)
// 		ORDER BY log_index ASC
// 	`
// 	result := db.IndexerClient.Raw(query, blockNumber).Scan(&contractCallWithTokens)
// 	if result.Error != nil {
// 		return nil, result.Error
// 	}
// 	return contractCallWithTokens, nil
// }

func (db *DatabaseAdapter) GetNextContractCallWithTokens(lastBlockNumber uint64, lastLogIndex uint64, limit int) ([]*chains.ContractCallWithToken, error) {
	var contractCallWithTokens []*chains.ContractCallWithToken
	query := `
		SELECT * FROM contract_call_with_tokens
		WHERE block_number > $1 OR (block_number = $1 AND log_index > $2)
		LIMIT $3
	`
	result := db.IndexerClient.Raw(query, lastBlockNumber, lastLogIndex, limit).Scan(&contractCallWithTokens)
	if result.Error != nil {
		return nil, result.Error
	}
	sort.Slice(contractCallWithTokens, func(i, j int) bool {
		return contractCallWithTokens[i].BlockNumber < contractCallWithTokens[j].BlockNumber ||
			(contractCallWithTokens[i].BlockNumber == contractCallWithTokens[j].BlockNumber && contractCallWithTokens[i].LogIndex < contractCallWithTokens[j].LogIndex)
	})
	return contractCallWithTokens, nil
}

func (db *DatabaseAdapter) UpdateContractCallApproved(messageID string, executeHash string) error {
	updateData := map[string]interface{}{
		"execute_hash": executeHash,
		"status":       chains.ContractCallStatusApproved,
	}
	record := db.RelayerClient.Model(&chains.ContractCall{}).Where("id = ?", messageID).Updates(updateData)
	if record.Error != nil {
		return record.Error
	}
	log.Info().Msgf("[DatabaseAdapter] [UpdateContractCallApproved]: RelayData[%s]", messageID)
	return nil
}
func (db *DatabaseAdapter) FindContractCallByCommnadId(commandId string) (*chains.ContractCall, error) {
	var contractCall chains.ContractCall
	result := db.RelayerClient.Where("command_id = ?", commandId).First(&contractCall)
	if result.Error != nil {
		return nil, result.Error
	}
	return &contractCall, nil
}
func (db *DatabaseAdapter) FindContractCallWithTokenPayloadByEventId(eventId string) (*chains.ContractCallWithToken, error) {
	var contractCallWithToken chains.ContractCallWithToken
	result := db.RelayerClient.Where("event_id = ?", eventId).First(&contractCallWithToken)
	if result.Error != nil {
		log.Error().Err(result.Error).Str("EventId", eventId).Msg("[DatabaseAdapter] [FindContractCallWithTokenPayloadByEventId] failed to find contract call with token by event id")
		return nil, result.Error
	}
	return &contractCallWithToken, nil
}

func (db *DatabaseAdapter) FindContractCallPayloadByHash(payloadHash string) ([]byte, error) {
	var contractCall chains.ContractCall
	log.Debug().Str("payloadHash", payloadHash).Msg("[DatabaseAdapter] [FindPayloadByHash] finding payload by hash")
	result := db.RelayerClient.
		Where("payload_hash = ?", strings.ToLower(payloadHash)).
		First(&contractCall)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to find payload by hash: %w", result.Error)
	}

	return contractCall.Payload, nil
}

// Find Realaydata by ContractAddress, SourceAddress, PayloadHash
func (db *DatabaseAdapter) FindContractCallByParams(sourceAddress string, destContractAddress string, payloadHash string) ([]chains.ContractCall, error) {
	var contractCalls []chains.ContractCall
	result := db.RelayerClient.
		Joins("CallContract").
		Where("contract_address = ? AND source_address = ? AND payload_hash = ?",
			strings.ToLower(destContractAddress),
			strings.ToLower(sourceAddress),
			strings.ToLower(payloadHash)).
		Where("status IN (?)", []chains.ContractCallStatus{chains.ContractCallStatusPending, chains.ContractCallStatusApproved}).
		Preload("CallContract").
		Find(&contractCalls)

	if result.Error != nil {
		return contractCalls, fmt.Errorf("find relaydatas by contract call with error: %w", result.Error)
	}
	if len(contractCalls) == 0 {
		log.Warn().Str("contractAddress", destContractAddress).Str("sourceAddress", sourceAddress).Str("payloadHash", payloadHash).Msg("[DatabaseAdapter] [FindRelayDataByContractCall] no relaydata found")
	}
	return contractCalls, nil

}

// func (db *DatabaseAdapter) UpdateCallContractWithExecuteHash(eventId string, status chains.ContractCallStatus, executeHash *string) error {
// 	data := chains.ContractCall{
// 		Status:      status,
// 		ExecuteHash: executeHash,
// 	}
// 	updateResult := db.updateCallContract(eventId, data)
// 	if updateResult.Error != nil {
// 		return updateResult.Error
// 	}
// 	return nil
// }

func (db *DatabaseAdapter) UpdateCallContractWithTokenExecuteHash(eventId string, status chains.ContractCallStatus, executeHash string) error {
	data := chains.ContractCallWithToken{
		ContractCall: chains.ContractCall{
			Status:      status,
			ExecuteHash: executeHash,
		},
	}
	updateResult := db.updateCallContractWithToken(eventId, data)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	return nil
}

func (db *DatabaseAdapter) CreateContractCall(contractCall chains.ContractCall, lastCheckpoint *scalarnet.EventCheckPoint) error {
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(&contractCall)
		if result.Error != nil {
			return result.Error
		}
		if lastCheckpoint != nil {
			UpdateLastEventCheckPoint(tx, lastCheckpoint)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create evm token send: %w", err)
	}
	return nil
}

// Cantract call with tokens which are processed by relayer
func (db *DatabaseAdapter) CreateRelayerContractCallWithTokens(contractCallWithTokens []*relayer.ContractCallWithToken) error {
	err := db.RelayerClient.Create(contractCallWithTokens).Error
	if err != nil {
		return fmt.Errorf("failed to create relayer contract call with tokens: %w", err)
	}
	return nil
}

func (db *DatabaseAdapter) updateCallContract(eventId string, data interface{}) (tx *gorm.DB) {
	result := db.RelayerClient.Model(&chains.ContractCall{}).Where("event_id = ?", eventId).Updates(data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseAdapter] [updateCallContract] %s: %v", eventId, result.Error)
	}
	log.Debug().Msgf("[DatabaseAdapter] [updateCallContract] %s: %v", eventId, data)
	return result
}

func (db *DatabaseAdapter) updateCallContractWithToken(eventId string, data interface{}) (tx *gorm.DB) {
	result := db.RelayerClient.Model(&chains.ContractCallWithToken{}).Where("event_id = ?", eventId).Updates(data)
	if result.Error != nil {
		log.Error().Msgf("[DatabaseAdapter] [updateCallContractWithToken] %s: %v", eventId, result.Error)
	}
	return result
}

// TODO: Find any better way to update batch relay data status
func (db *DatabaseAdapter) UpdateBatchContractCallStatus(data []ContractCallExecuteResult, batchSize int) error {
	// Handle empty data case
	if len(data) == 0 {
		return nil
	}

	// Process updates in batches
	return db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(data); i += batchSize {
			end := i + batchSize
			if end > len(data) {
				end = len(data)
			}

			batch := data[i:end]
			for _, item := range batch {
				updates := chains.ContractCall{
					Status: item.Status,
				}

				result := tx.Model(&chains.ContractCall{}).
					Where("event_id = ?", item.EventId).
					Updates(updates)

				if result.Error != nil {
					return fmt.Errorf("failed to update contract call batch: %w", result.Error)
				}
			}
		}
		return nil
	})
}

func (db *DatabaseAdapter) UpdateContractCallWithMintsStatus(ctx context.Context, cmdIds []string, status chains.ContractCallStatus) error {
	log.Debug().Any("cmdIds", cmdIds).Msg("[DatabaseAdapter] UpdateContractCallWithMintsStatus")
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		eventIds := tx.Model(&scalarnet.ContractCallApprovedWithMint{}).Select("event_id").Where("command_id IN (?)", cmdIds)
		//only update the token sent that is not success
		result := tx.Model(&chains.ContractCallWithToken{}).Where("event_id IN (?) and status != ?", eventIds, chains.ContractCallStatusSuccess).Update("status", status)
		return result.Error
	})
	return err
}
