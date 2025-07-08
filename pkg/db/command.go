package db

import (
	"context"
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (db *DatabaseAdapter) SaveCommands(commands []*scalarnet.Command) error {
	return db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "command_id"}},
			DoNothing: true,
		}).Create(commands).Error
		if err != nil {
			return fmt.Errorf("failed to save commands: %w", err)
		}
		return nil
	})
}

func (db *DatabaseAdapter) SaveCommandExecuted(cmdExecuted *chains.CommandExecuted, command *chainstypes.CommandResponse) error {
	commandId := cmdExecuted.CommandID
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		//TODO: use original postgres for upsert command instead of timescaledb
		// storedCmdExecuted := chains.CommandExecuted{}
		// err = tx.Where("command_id = ?", cmdExecuted.CommandID).First(&storedCmdExecuted).Error
		// if err != nil {
		// 	result := tx.Save(cmdExecuted)
		// 	if result.Error != nil {
		// 		return result.Error
		// 	}
		// }
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "command_id"}},
			DoNothing: true,
		}).Create(cmdExecuted).Error
		if err != nil {
			return fmt.Errorf("failed to create command executed: %w", err)
		}
		//Update or create Command record
		commandModel := scalarnet.Command{
			CommandID:      cmdExecuted.CommandID,
			ChainID:        cmdExecuted.SourceChain,
			ExecutedTxHash: cmdExecuted.TxHash,
			Status:         scalarnet.CommandStatusExecuted,
		}
		//1. Get command from db
		// storedCommand := scalarnet.Command{}
		// err = tx.Where("command_id = ?", cmdExecuted.CommandID).First(&storedCommand).Error
		// if err != nil {
		// 	tx.Create(&commandModel)
		// } else {
		// 	tx.Model(&scalarnet.Command{}).Where("command_id = ?", cmdExecuted.CommandID).Updates(map[string]interface{}{
		// 		"executed_tx_hash": cmdExecuted.TxHash,
		// 		"status":           scalarnet.CommandStatusExecuted,
		// 	})
		// }
		err = tx.Clauses(
			clause.OnConflict{
				Columns: []clause.Column{{Name: "command_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"executed_tx_hash": cmdExecuted.TxHash,
					"status":           scalarnet.CommandStatusExecuted,
				}),
			},
		).Create(&commandModel).Error
		if err != nil {
			return fmt.Errorf("failed to create command: %w", err)
		}
		tx.Exec(`UPDATE contract_call_with_tokens SET status = ? WHERE tx_hash = ?;`, chains.TokenSentStatusSuccess, commandId)
		tx.Exec(`UPDATE token_sents SET status = ? WHERE tx_hash = ?;`, chains.TokenSentStatusSuccess, commandId)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to save command executed: %w", err)
	}
	// if command != nil {
	// 	db.PostgresClient.Transaction(func(tx *gorm.DB) error {
	// 		var eventId string
	// 		var err error
	// 		//Find eventId by commandId
	// 		switch command.Type {
	// 		case "approveContractCallWithMint":
	// 			err = db.PostgresClient.Select("event_id").Model(&scalarnet.ContractCallApprovedWithMint{}).Where("command_id = ?", commandId).Find(&eventId).Error
	// 			if err == nil {
	// 				tx.Model(&chains.ContractCallWithToken{}).Where("event_id = ?", eventId).Update("status", chains.TokenSentStatusSuccess)
	// 			} else {
	// 				log.Error().Err(err).
	// 					Str("commandType", command.Type).
	// 					Str("commandId", commandId).
	// 					Msgf("[DatabaseAdapter] [SaveCommandExecuted] failed to find eventId")
	// 			}
	// 		case "mintToken":
	// 			err = db.PostgresClient.Select("event_id").Model(&scalarnet.TokenSentApproved{}).Where("command_id = ?", commandId).First(&eventId).Error
	// 			if err == nil {
	// 				tx.Model(&chains.TokenSent{}).Where("event_id = ?", eventId).Update("status", chains.TokenSentStatusSuccess)
	// 			} else {
	// 				log.Error().Err(err).
	// 					Str("commandType", command.Type).
	// 					Str("commandId", commandId).
	// 					Msgf("[DatabaseAdapter] [SaveCommandExecuted] failed to find eventId")
	// 			}
	// 		}
	// 		return err
	// 	})
	// }
	return nil
}

func (db *DatabaseAdapter) UpdateBroadcastedCommands(chainId string, batchedCommandId string, commandIds []string, txHash string) error {
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&scalarnet.Command{}).
			Where("batch_command_id = ? AND command_id IN (?)", batchedCommandId, commandIds).
			Updates(scalarnet.Command{ExecutedTxHash: txHash, Status: scalarnet.CommandStatusBroadcasted}).Error
		if err != nil {
			return fmt.Errorf("failed to update broadcasted commands: %w", err)
		} else {
			tx.Logger.Info(context.Background(),
				fmt.Sprintf("[DatabaseAdapter] UpdateBroadcastedCommands successfully with chainId: %s, batchedCommandId: %s, commandIds: %v, txHash: %s",
					chainId, batchedCommandId, commandIds, txHash))
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
		} else {
			tx.Logger.Info(context.Background(),
				fmt.Sprintf("[DatabaseAdapter] UpdateContractCallTokensStatus successfully with chainId: %s, batchedCommandId: %s, commandIds: %v, txHash: %s",
					chainId, batchedCommandId, commandIds, txHash))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update broadcasted commands: %w", err)
	}
	return nil
}
