package db

import (
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) SaveBtcRedeemTxs(btcChain string, redeemTxs []*chains.BtcRedeemTx) error {
	//2. Update ContractCallWithToken status with execution confirmation from bitcoin network
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Save(redeemTxs).Error
		if err != nil {
			return err
		}
		tx.Exec(`UPDATE evm_redeem_txes SET status = ?, execute_hash = brt.tx_hash
				FROM evm_redeem_txes ert 
				JOIN btc_redeem_txes brt 
				ON ert.custodian_group_uid = brt.custodian_group_uid
					AND ert.session_sequence = brt.session_sequence
					AND ert.destination_chain = brt.chain
				WHERE ert.destination_chain = ?
				`,
			chains.RedeemStatusSuccess, btcChain)
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
