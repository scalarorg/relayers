package db

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	chainstypes "github.com/scalarorg/scalar-core/x/chains/types"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) SaveCommands(commands []*scalarnet.Command) error {
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		// for _, command := range commands {
		// 	err := tx.Clauses(clause.OnConflict{
		// 		Columns: []clause.Column{{Name: "command_id"}},
		// 		DoUpdates: clause.Assignments(map[string]interface{}{
		// 			"batch_command_id": command.BatchCommandID,
		// 			"command_type":     command.CommandType,
		// 			"key_id":           command.KeyID,
		// 			"params":           command.Params,
		// 			"chain_id":         command.ChainID,
		// 		}),
		// 	}).Create(command).Error
		// 	if err != nil {
		// 		return fmt.Errorf("[DatabaseAdapter] failed to save command: %w", err)
		// 	}
		// }
		//1. Try get all stored command by commandIds
		commandIds := make([]string, len(commands))
		for _, command := range commands {
			commandIds = append(commandIds, command.CommandID)
		}
		storedCommands := make([]*scalarnet.Command, len(commandIds))
		err := tx.Where("command_id IN (?)", commandIds).Find(&storedCommands).Error
		if err != nil {
			return fmt.Errorf("[DatabaseAdapter] failed to get stored commands: %w", err)
		}
		storedCommandMap := make(map[string]*scalarnet.Command)
		for _, command := range storedCommands {
			storedCommandMap[command.CommandID] = command
		}
		for _, command := range commands {
			_, ok := storedCommandMap[command.CommandID]
			if !ok {
				err := tx.Create(command).Error
				if err != nil {
					tx.Logger.Error(context.Background(), fmt.Sprintf("[DatabaseAdapter] failed to save command: %s", command.CommandID))
				}
			} else {
				err := tx.Model(&scalarnet.Command{}).Where("command_id = ?", command.CommandID).Updates(map[string]interface{}{
					"batch_command_id": command.BatchCommandID,
					"command_type":     command.CommandType,
					"key_id":           command.KeyID,
					"params":           command.Params,
					"chain_id":         command.ChainID,
				}).Error
				if err != nil {
					tx.Logger.Error(context.Background(), fmt.Sprintf("[DatabaseAdapter] failed to update command: %s", command.CommandID))
				}
			}
		}
		return nil
	})
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
		storedCmdExecuted := chains.CommandExecuted{}
		err = tx.Where("command_id = ?", cmdExecuted.CommandID).First(&storedCmdExecuted).Error
		if err != nil {
			result := tx.Save(cmdExecuted)
			if result.Error != nil {
				return result.Error
			}
		}

		//Update or create Command record
		commandModel := scalarnet.Command{
			CommandID:      cmdExecuted.CommandID,
			ChainID:        cmdExecuted.SourceChain,
			ExecutedTxHash: cmdExecuted.TxHash,
			Status:         scalarnet.CommandStatusExecuted,
		}
		// result = tx.Clauses(
		// 	clause.OnConflict{
		// 		Columns: []clause.Column{{Name: "command_id"}},
		// 		DoUpdates: clause.Assignments(map[string]interface{}{
		// 			"executed_tx_hash": cmdExecuted.TxHash,
		// 			"status":           scalarnet.CommandStatusExecuted,
		// 		}),
		// 	},
		// ).Create(&commandModel)

		//1. Get command from db
		storedCommand := scalarnet.Command{}
		err = tx.Where("command_id = ?", cmdExecuted.CommandID).First(&storedCommand).Error
		if err != nil {
			tx.Create(&commandModel)
		} else {
			tx.Model(&scalarnet.Command{}).Where("command_id = ?", cmdExecuted.CommandID).Updates(map[string]interface{}{
				"executed_tx_hash": cmdExecuted.TxHash,
				"status":           scalarnet.CommandStatusExecuted,
			})
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
		return fmt.Errorf("failed to save command executed: %w", err)
	}
	return nil
}

func (db *DatabaseAdapter) UpdateBroadcastedCommands(chainId string, batchedCommandId string, commandIds []string, txHash string) error {
	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
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

func (db *DatabaseAdapter) UpdateBtcExecutedCommands(chainId string, txHashes []string) error {
	log.Info().Str("chainId", chainId).Any("txHashes", txHashes).Msg("[DatabaseAdapter] [UpdateBtcExecutedCommands]")

	result := db.PostgresClient.Exec(`UPDATE contract_call_with_tokens as ccwt SET status = ? 
						WHERE ccwt.event_id 
						IN (SELECT ccawm.event_id FROM contract_call_approved_with_mints as ccawm 
							JOIN commands as c ON ccawm.command_id = c.command_id 
							WHERE c.chain_id = ? AND c.executed_tx_hash IN (?))`,
		chains.ContractCallStatusSuccess, chainId, txHashes)
	log.Info().Any("RowsAffected", result.RowsAffected).Msg("[DatabaseAdapter] [UpdateBtcExecutedCommands]")
	return result.Error
}

// Get last pending redeem transaction by block height
func (db *DatabaseAdapter) FindPendingRedeemTransaction(chainId string, blockHeight int) (*chains.RedeemTx, error) {
	var redeemTx chains.RedeemTx
	err := db.PostgresClient.Where("chain = ? AND block_number <= ?", chainId, blockHeight).Where("status = ?", chains.RedeemStatusExecuting).First(&redeemTx).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find pending redeem transaction: %w", err)
	}
	return &redeemTx, nil
}
