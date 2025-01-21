package db

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/relayers/pkg/db/models"
	"gorm.io/gorm"
)

// find relay datas by token sent attributes
func (db *DatabaseAdapter) FindPendingBtcTokenSent(sourceChain string, height int) ([]*chains.TokenSent, error) {
	var tokenSents []*chains.TokenSent
	result := db.PostgresClient.
		Where("source_chain = ? AND block_number <= ?",
			sourceChain,
			height).
		Where("status = ?", string(chains.TokenSentStatusPending)).
		Find(&tokenSents)

	if result.Error != nil {
		return tokenSents, fmt.Errorf("FindPendingBtcTokenSent with error: %w", result.Error)
	}
	if len(tokenSents) == 0 {
		log.Warn().
			Str("sourceChain", sourceChain).
			Msgf("[DatabaseAdapter] [FindPendingBtcTokenSent] no token sent with block height before %d found", height)
	}
	return tokenSents, nil
}

func (db *DatabaseAdapter) SaveTokenSents(tokenSents []chains.TokenSent) error {
	result := db.PostgresClient.Save(tokenSents)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (db *DatabaseAdapter) SaveTokenSent(tokenSent chains.TokenSent, lastCheckpoint *models.EventCheckPoint) error {
	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(&tokenSent)
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

func (db *DatabaseAdapter) UpdateTokenSentsStatus(chain string, status chains.ContractCallStatus) error {
	tx := db.PostgresClient.Model(&chains.TokenSent{}).Where("destination_chain = ?", chain).Update("status", status)
	return tx.Error
}
