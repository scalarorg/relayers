package db

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) SaveCommands(commands []*scalarnet.Command) error {
	return db.PostgresClient.Save(commands).Error
}
func (db *DatabaseAdapter) UpdateBroadcastedCommands(chainId string, batchedCommandId string, commandIds []string, txHash string) error {
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&scalarnet.Command{}).
			Where("batch_command_id = ? AND command_id IN (?)", batchedCommandId, commandIds).
			Updates(scalarnet.Command{ExecutedTxHash: txHash, Status: scalarnet.CommandStatusBroadcasted}).Error
		if err != nil {
			return fmt.Errorf("failed to update broadcasted commands: %w", err)
		}
		err = tx.Exec(`UPDATE contract_call_with_tokens as ccwt SET status = ? 
						WHERE ccwt.event_id 
						IN (SELECT ccawm.event_id FROM contract_call_approved_with_mints as ccawm WHERE ccawm.command_id IN (?))`,
			chains.ContractCallStatusExecuting, commandIds).Error
		// err = tx.Table("contract_call_with_tokens as ccwt").
		// 	Joins("JOIN contract_call_approved_with_mints as ccawm ON ccwt.event_id = ccawm.event_id").
		// 	Where("ccawm.command_id IN (?)", commandIds).
		// 	Update("ccwt.status", chains.ContractCallStatusExecuting).Error
		if err != nil {
			return fmt.Errorf("failed to update contract call tokens status: %w", err)
		}
		return nil
	})
}

func (db *DatabaseAdapter) UpdateBtcExecutedCommands(chainId string, txHashes []string) error {
	log.Info().Any("txHashes", txHashes).Msg("UpdateBtcExecutedCommands")
	log.Info().Any("chainId", chainId).Msg("UpdateBtcExecutedCommands")

	result := db.PostgresClient.Exec(`UPDATE contract_call_with_tokens as ccwt SET status = ? 
						WHERE ccwt.event_id 
						IN (SELECT ccawm.event_id FROM contract_call_approved_with_mints as ccawm 
							JOIN commands as c ON ccawm.command_id = c.command_id 
							WHERE c.chain_id = ? AND c.executed_tx_hash IN (?))`,
		chains.ContractCallStatusSuccess, chainId, txHashes)

	log.Info().Any("result", result.RowsAffected).Msg("UpdateBtcExecutedCommands")
	return result.Error
}
func (db *DatabaseAdapter) SaveCommandExecuted(cmdExecuted *chains.CommandExecuted, command *chainstypes.CommandResponse, commandId string) error {
	var eventId string
	var err error
	//Find eventId by commandId
	if command != nil {
		switch command.Type {
		case "approveContractCallWithMint":
			err = db.PostgresClient.Select("event_id").Model(&scalarnet.ContractCallApprovedWithMint{}).Where("command_id = ?", commandId).Find(&eventId).Error
		case "mintToken":
			err = db.PostgresClient.Select("event_id").Model(&scalarnet.TokenSentApproved{}).Where("command_id = ?", commandId).First(&eventId).Error
		}
		if err != nil {
			return fmt.Errorf("failed to find eventId: %w", err)
		}
		log.Debug().Str("commandType", command.Type).Str("commandId", commandId).Msgf("[DatabaseAdapter] [SaveCommandExecuted] found eventId: %s", eventId)

	}
	err = db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(cmdExecuted)
		if result.Error != nil {
			return result.Error
		}
		//The eventId is empty only when we restart whole system from beginning
		if eventId != "" && command != nil {
			switch command.Type {
			case "approveContractCallWithMint":
				tx.Model(&chains.ContractCallWithToken{}).Where("event_id = ?", eventId).Update("status", chains.TokenSentStatusSuccess)
			case "mintToken":
				tx.Model(&chains.TokenSent{}).Where("event_id = ?", eventId).Update("status", chains.TokenSentStatusSuccess)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create Single value with checkpoint: %w", err)
	}
	return nil
}
