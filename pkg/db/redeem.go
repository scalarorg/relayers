package db

import (
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"gorm.io/gorm/clause"
)

func (db *DatabaseAdapter) CreateRedeemBlock(redeemBlock *relayer.RedeemBlock) error {
	return db.RelayerClient.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "block_number"}},
		DoNothing: true,
	}).Create(redeemBlock).Error
}

func (db *DatabaseAdapter) GetLastRedeemBlock() (*relayer.RedeemBlock, error) {
	var redeemTxBlock relayer.RedeemBlock
	query := `
		SELECT * FROM redeem_blocks ORDER BY block_number DESC LIMIT 1
	`
	err := db.RelayerClient.Raw(query).Scan(&redeemTxBlock).Error
	if err != nil {
		return nil, err
	}
	if redeemTxBlock.ID == 0 {
		return nil, nil
	}
	return &redeemTxBlock, nil
}

func (db *DatabaseAdapter) GetLastRedeemTxs(groupUid string, sessionSequence uint64) ([]*chains.EvmRedeemTx, error) {
	var redeemTxs []*chains.EvmRedeemTx
	query := `
		SELECT * FROM evm_redeem_txes
		WHERE custodian_group_uid = $1
		AND session_sequence = $2
	`
	err := db.IndexerClient.Raw(query, groupUid, sessionSequence).Scan(&redeemTxs).Error
	return redeemTxs, err
}

// GetRedeemTokenEventsByGroupAndSequence retrieves redeem token events from database by group UID and session sequence
func (db *DatabaseAdapter) GetRedeemTokenEventsByGroupAndSequence(groupUid string, sessionSequence uint64) ([]*chains.EvmRedeemTx, error) {
	var redeemTxs []*chains.EvmRedeemTx
	query := `
		SELECT * FROM evm_redeem_txes
		WHERE custodian_group_uid = $1
		AND session_sequence = $2
		ORDER BY block_number ASC, log_index ASC
	`
	err := db.IndexerClient.Raw(query, groupUid, sessionSequence).Scan(&redeemTxs).Error
	return redeemTxs, err
}

// Find redeem txs in the last session in pool model
func (db *DatabaseAdapter) FindPoolRedeemTxsInLastSession(custodianGroupUid string) ([]*chains.EvmRedeemTx, error) {
	var redeemTxs []*chains.EvmRedeemTx
	query := `
		SELECT * FROM evm_redeem_txes
		WHERE session_sequence = (
			SELECT max(session_sequence) FROM evm_redeem_txes
			WHERE custodian_group_uid = $1
		) and session_sequence > 0
	`
	err := db.IndexerClient.Raw(query, custodianGroupUid).Scan(&redeemTxs).Error
	if err != nil {
		return nil, err
	}
	return redeemTxs, nil
}

// Select redeem vault tx
func (db *DatabaseAdapter) FindLastRedeemVaultTxs(groupUid string) ([]*chains.BtcRedeemTx, error) {
	var redeemTxs []*chains.BtcRedeemTx
	query := `
		SELECT * FROM btc_redeem_txes
		WHERE custodian_group_uid = $1
		AND session_sequence = (
			SELECT max(session_sequence) FROM btc_redeem_txes
			WHERE custodian_group_uid = $1
		)
	`
	err := db.IndexerClient.Raw(query, groupUid).Scan(&redeemTxs).Error
	if err != nil {
		return nil, err
	}
	return redeemTxs, nil
}

func (db *DatabaseAdapter) GetNextBtcRedeemExecuteds(lastProcessedBlock uint64) ([]*chains.BtcRedeemTx, error) {
	var btcRedeemTxs []*chains.BtcRedeemTx
	query := `
		SELECT * FROM btc_redeem_txes
		WHERE block_number = (
			SELECT MIN(block_number) 
			FROM btc_redeem_txes
			WHERE block_number > $1
		)
	`
	err := db.IndexerClient.Raw(query, lastProcessedBlock).Scan(&btcRedeemTxs).Error
	return btcRedeemTxs, err
}

func (db *DatabaseAdapter) GetBtcRedeemExecutedsByBlock(blockNumber uint64) ([]*chains.BtcRedeemTx, error) {
	var btcRedeemTxs []*chains.BtcRedeemTx
	query := `
		SELECT * FROM btc_redeem_txes
		WHERE block_number = $1
	`
	err := db.IndexerClient.Raw(query, blockNumber).Scan(&btcRedeemTxs).Error
	return btcRedeemTxs, err
}

// func (db *DatabaseAdapter) SaveBtcRedeemTxs(btcChain string, redeemTxs []*chains.BtcRedeemTx) error {
// 	//2. Update ContractCallWithToken status with execution confirmation from bitcoin network
// 	// Clean redeemTxs to keep only one element per custodianGroupUid and SessionSequence with highest blockNumber
// 	cleanedRedeemTxs := make([]*chains.BtcRedeemTx, 0)
// 	redeemTxMap := make(map[string]*chains.BtcRedeemTx)

// 	for _, tx := range redeemTxs {
// 		key := fmt.Sprintf("%s_%d", tx.CustodianGroupUid, tx.SessionSequence)
// 		existingTx, exists := redeemTxMap[key]
// 		if !exists || tx.BlockNumber > existingTx.BlockNumber {
// 			redeemTxMap[key] = tx
// 		}
// 	}
// 	for _, tx := range cleanedRedeemTxs {
// 		key := fmt.Sprintf("%s_%d", tx.CustodianGroupUid, tx.SessionSequence)
// 		existingTx, exists := redeemTxMap[key]
// 		if !exists || tx.BlockNumber > existingTx.BlockNumber {
// 			redeemTxMap[key] = tx
// 		}
// 	}
// 	oldRedeemTxs := make([]uint, 0)
// 	for _, tx := range redeemTxMap {
// 		var existingTx chains.BtcRedeemTx
// 		err := db.RelayerClient.Where("chain = ? AND custodian_group_uid = ? AND session_sequence = ? AND block_number < ?",
// 			btcChain, tx.CustodianGroupUid, tx.SessionSequence, tx.BlockNumber).First(&existingTx).Error
// 		if err == nil {
// 			log.Error().Err(err).Msg("Redeem Tx with the same custodian_group_uid and session_sequence found")
// 			if tx.BlockNumber > existingTx.BlockNumber {
// 				oldRedeemTxs = append(oldRedeemTxs, existingTx.ID)
// 				cleanedRedeemTxs = append(cleanedRedeemTxs, tx)
// 			}
// 		} else {
// 			//RedeemTx with the same custodian_group_uid and session_sequence is not exist
// 			cleanedRedeemTxs = append(cleanedRedeemTxs, tx)
// 		}
// 	}

// 	return db.RelayerClient.Transaction(func(tx *gorm.DB) error {
// 		// Remove existing records with same custodian_group_uid and session_sequence but smaller block_number
// 		result := tx.Exec(`DELETE FROM btc_redeem_txes where id IN (?)`, oldRedeemTxs)
// 		if result.Error != nil {
// 			return result.Error
// 		}
// 		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Deleted old redeem transactions")
// 		result = tx.Save(cleanedRedeemTxs)
// 		if result.Error != nil {
// 			return result.Error
// 		}
// 		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Saved cleaned redeem transactions")
// 		result = tx.Exec(`UPDATE evm_redeem_txes SET status = ?, execute_hash = brt.tx_hash
// 				FROM evm_redeem_txes ert
// 				JOIN btc_redeem_txes brt
// 				ON ert.custodian_group_uid = brt.custodian_group_uid
// 					AND ert.session_sequence = brt.session_sequence
// 					AND ert.destination_chain = brt.chain
// 				WHERE ert.destination_chain = ?
// 				`,
// 			chains.RedeemStatusSuccess, btcChain)
// 		if result.Error != nil {
// 			return result.Error
// 		}
// 		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Updated evm redeem txes status")
// 		return nil
// 	})
// }

// func (db *DatabaseAdapter) CreateEvmRedeemToken(redeemToken *chains.EvmRedeemTx, lastCheckpoint *scalarnet.EventCheckPoint) error {
// 	err := db.RelayerClient.Transaction(func(tx *gorm.DB) error {
// 		result := tx.Save(redeemToken)
// 		if result.Error != nil {
// 			return result.Error
// 		}
// 		if lastCheckpoint != nil {
// 			UpdateLastEventCheckPoint(tx, lastCheckpoint)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to create evm token send: %w", err)
// 	}
// 	return nil
// }

// // Get last pending redeem transaction by block height
// func (db *DatabaseAdapter) FindExecutingRedeemTxs(chainId string, blockHeight int) ([]*chains.BtcRedeemTx, error) {
// 	var redeemTxs []*chains.BtcRedeemTx
// 	err := db.RelayerClient.Where("chain = ? AND block_number <= ? AND status = ?", chainId, blockHeight, chains.RedeemStatusExecuting).
// 		Find(&redeemTxs).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to find pending redeem transaction: %w", err)
// 	}
// 	return redeemTxs, nil
// }

// func (db *DatabaseAdapter) UpdateRedeemTxsStatus(redeemTxs []*chains.BtcRedeemTx, status chains.RedeemStatus) error {
// 	txHashes := make([]string, len(redeemTxs))
// 	for i, redeemTx := range redeemTxs {
// 		txHashes[i] = redeemTx.TxHash
// 	}
// 	return db.RelayerClient.Model(&chains.BtcRedeemTx{}).Where("tx_hash IN (?)", txHashes).Update("status", status).Error
// }

// func (db *DatabaseAdapter) StoreRedeemEvent(redeemEvent *contracts.IScalarGatewayRedeemToken) error {
// 	return db.RelayerClient.Create(redeemEvent).Error
// }

// func (db *DatabaseAdapter) UpdateRedeemExecutedCommands(chainId string, txHashes []string) error {
// 	log.Info().Str("chainId", chainId).Any("txHashes", txHashes).Msg("[DatabaseAdapter] [UpdateRedeemExecutedCommands]")

// 	result := db.PostgresClient.Exec(`UPDATE evm_redeem_txes as ert SET status = ?
// 						WHERE ert.event_id
// 						IN (SELECT srta.event_id FROM scalar_redeem_token_approveds as srta
// 							JOIN commands as c ON srta.command_id = srta.command_id
// 							WHERE c.chain_id = ? AND c.executed_tx_hash IN (?))`,
// 		chains.RedeemStatusSuccess, chainId, txHashes)
// 	log.Info().Any("RowsAffected", result.RowsAffected).Msg("[DatabaseAdapter] [UpdateRedeemExecutedCommands]")
// 	return result.Error
// }

// Delete existing redeem transactions with same custodian_group_uid and session_sequence but smaller block_number
// func (db *DatabaseAdapter) DeleteBtcRedeemTxsByGroupAndSequence(chainId string, custodianGroupUid string, sessionSequence uint64, blockNumber uint64) error {
// 	return db.RelayerClient.Where("chain = ? AND custodian_group_uid = ? AND session_sequence = ? AND block_number < ?",
// 		chainId, custodianGroupUid, sessionSequence, blockNumber).Delete(&chains.BtcRedeemTx{}).Error
// }
