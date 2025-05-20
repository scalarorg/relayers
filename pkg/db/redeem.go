package db

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) SaveBtcRedeemTxs(btcChain string, redeemTxs []*chains.BtcRedeemTx) error {
	//2. Update ContractCallWithToken status with execution confirmation from bitcoin network
	// Clean redeemTxs to keep only one element per custodianGroupUid and SessionSequence with highest blockNumber
	cleanedRedeemTxs := make([]*chains.BtcRedeemTx, 0)
	redeemTxMap := make(map[string]*chains.BtcRedeemTx)

	for _, tx := range redeemTxs {
		key := fmt.Sprintf("%s_%d", tx.CustodianGroupUid, tx.SessionSequence)
		existingTx, exists := redeemTxMap[key]
		if !exists || tx.BlockNumber > existingTx.BlockNumber {
			redeemTxMap[key] = tx
		}
	}
	for _, tx := range cleanedRedeemTxs {
		key := fmt.Sprintf("%s_%d", tx.CustodianGroupUid, tx.SessionSequence)
		existingTx, exists := redeemTxMap[key]
		if !exists || tx.BlockNumber > existingTx.BlockNumber {
			redeemTxMap[key] = tx
		}
	}
	oldRedeemTxs := make([]uint, 0)
	for _, tx := range redeemTxMap {
		var existingTx chains.BtcRedeemTx
		err := db.PostgresClient.Where("chain = ? AND custodian_group_uid = ? AND session_sequence = ? AND block_number < ?",
			btcChain, tx.CustodianGroupUid, tx.SessionSequence, tx.BlockNumber).First(&existingTx).Error
		if err == nil {
			log.Error().Err(err).Msg("Redeem Tx with the same custodian_group_uid and session_sequence found")
			if tx.BlockNumber > existingTx.BlockNumber {
				oldRedeemTxs = append(oldRedeemTxs, existingTx.ID)
				cleanedRedeemTxs = append(cleanedRedeemTxs, tx)
			}
		} else {
			//RedeemTx with the same custodian_group_uid and session_sequence is not exist
			cleanedRedeemTxs = append(cleanedRedeemTxs, tx)
		}
	}

	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		// Remove existing records with same custodian_group_uid and session_sequence but smaller block_number
		result := tx.Exec(`DELETE FROM btc_redeem_txes where id IN (?)`, oldRedeemTxs)
		if result.Error != nil {
			return result.Error
		}
		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Deleted old redeem transactions")
		result = tx.Save(cleanedRedeemTxs)
		if result.Error != nil {
			return result.Error
		}
		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Saved cleaned redeem transactions")
		result = tx.Exec(`UPDATE evm_redeem_txes SET status = ?, execute_hash = brt.tx_hash
				FROM evm_redeem_txes ert 
				JOIN btc_redeem_txes brt 
				ON ert.custodian_group_uid = brt.custodian_group_uid
					AND ert.session_sequence = brt.session_sequence
					AND ert.destination_chain = brt.chain
				WHERE ert.destination_chain = ?
				`,
			chains.RedeemStatusSuccess, btcChain)
		if result.Error != nil {
			return result.Error
		}
		log.Info().Any("RowsAffected", result.RowsAffected).Msg("Updated evm redeem txes status")
		return nil
	})
}

func (db *DatabaseAdapter) CreateEvmRedeemToken(redeemToken *chains.EvmRedeemTx, lastCheckpoint *scalarnet.EventCheckPoint) error {
	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(redeemToken)
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

// Get last pending redeem transaction by block height
func (db *DatabaseAdapter) FindExecutingRedeemTxs(chainId string, blockHeight int) ([]*chains.BtcRedeemTx, error) {
	var redeemTxs []*chains.BtcRedeemTx
	err := db.PostgresClient.Where("chain = ? AND block_number <= ? AND status = ?", chainId, blockHeight, chains.RedeemStatusExecuting).
		Find(&redeemTxs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find pending redeem transaction: %w", err)
	}
	return redeemTxs, nil
}

func (db *DatabaseAdapter) UpdateRedeemTxsStatus(redeemTxs []*chains.BtcRedeemTx, status chains.RedeemStatus) error {
	txHashes := make([]string, len(redeemTxs))
	for i, redeemTx := range redeemTxs {
		txHashes[i] = redeemTx.TxHash
	}
	return db.PostgresClient.Model(&chains.BtcRedeemTx{}).Where("tx_hash IN (?)", txHashes).Update("status", status).Error
}

func (db *DatabaseAdapter) StoreRedeemEvent(redeemEvent *contracts.IScalarGatewayRedeemToken) error {
	return db.PostgresClient.Create(redeemEvent).Error
}

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
func (db *DatabaseAdapter) DeleteBtcRedeemTxsByGroupAndSequence(chainId string, custodianGroupUid string, sessionSequence uint64, blockNumber uint64) error {
	return db.PostgresClient.Where("chain = ? AND custodian_group_uid = ? AND session_sequence = ? AND block_number < ?",
		chainId, custodianGroupUid, sessionSequence, blockNumber).Delete(&chains.BtcRedeemTx{}).Error
}
