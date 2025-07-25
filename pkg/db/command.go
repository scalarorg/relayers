package db

import (
	"context"
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
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

func (db *DatabaseAdapter) UpdateBroadcastedCommands(chainId string, batchedCommandId string, commandIds []string, btcTxHash string) error {
	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&scalarnet.Command{}).
			Where("batch_command_id = ? AND command_id IN (?)", batchedCommandId, commandIds).
			Updates(scalarnet.Command{ExecutedTxHash: btcTxHash, Status: scalarnet.CommandStatusBroadcasted}).Error
		if err != nil {
			return fmt.Errorf("failed to update broadcasted commands: %w", err)
		} else {
			tx.Logger.Info(context.Background(),
				fmt.Sprintf("[DatabaseAdapter] UpdateBroadcastedCommands successfully with chainId: %s, batchedCommandId: %s, commandIds: %v, txHash: %s",
					chainId, batchedCommandId, commandIds, btcTxHash))
		}
		err = tx.Model(&relayer.ContractCallWithToken{}).
			Where("tx_hash in (?)", commandIds).Update("status", chains.ContractCallStatusExecuting).Error
		if err != nil {
			return fmt.Errorf("failed to update contract call tokens status: %w", err)
		} else {
			tx.Logger.Info(context.Background(),
				fmt.Sprintf("[DatabaseAdapter] UpdateContractCallTokensStatus successfully with chainId: %s, batchedCommandId: %s, commandIds: %v, txHash: %s",
					chainId, batchedCommandId, commandIds, btcTxHash))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update broadcasted commands: %w", err)
	}
	return nil
}

func (db *DatabaseAdapter) UpdateBatchCommandSigned(chainId string, commandIds []string) error {
	// var blockCommands []struct {
	// 	BlockNumber uint64
	// 	BlockHash   string
	// 	Counter     int
	// }
	// rawSql := `select block_number, block_hash, count(tx_hash) as counter from vault_transactions vt
	// 			where tx_hash in (?)
	// 			group by block_number, block_hash`
	// db.IndexerClient.Raw(rawSql, commandIds).Scan(&blockCommands)
	err := db.RelayerClient.Model(&relayer.VaultTransaction{}).Where("tx_hash in (?)", commandIds).Update("status", "executing").Error
	if err != nil {
		return fmt.Errorf("failed to update signed commands: %w", err)
	}
	return nil
}
