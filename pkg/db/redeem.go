package db

import (
	"fmt"

	"github.com/scalarorg/data-models/chains"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
)

func (db *DatabaseAdapter) SaveRedeemTxs(redeemTxs []*chains.RedeemTx) error {
	return db.PostgresClient.Save(redeemTxs).Error
	// return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
	// 	err := tx.Save(redeemTxs).Error
	// 	if err != nil {
	// 		return err
	// 	}
	// 	tx.Exec(`
	// 		UPDATE redeem_txs
	// 		SET block_time = block_headers.block_time
	// 		FROM block_headers
	// 		WHERE redeem_txs.chain = block_headers.chain
	// 		AND redeem_txs.block_number = block_headers.block_number
	// 	`)
	// 	return nil
	// })
}

// Get last pending redeem transaction by block height
func (db *DatabaseAdapter) FindExecutingRedeemTxs(chainId string, blockHeight int) ([]*chains.RedeemTx, error) {
	var redeemTxs []*chains.RedeemTx
	err := db.PostgresClient.Where("chain = ? AND block_number <= ? AND status = ?", chainId, blockHeight, chains.RedeemStatusExecuting).
		Find(&redeemTxs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find pending redeem transaction: %w", err)
	}
	return redeemTxs, nil
}

func (db *DatabaseAdapter) UpdateRedeemTxsStatus(redeemTxs []*chains.RedeemTx, status chains.RedeemStatus) error {
	txHashes := make([]string, len(redeemTxs))
	for i, redeemTx := range redeemTxs {
		txHashes[i] = redeemTx.TxHash
	}
	return db.PostgresClient.Model(&chains.RedeemTx{}).Where("tx_hash IN (?)", txHashes).Update("status", status).Error
}

func (db *DatabaseAdapter) StoreRedeemEvent(redeemEvent *contracts.IScalarGatewayRedeemToken) error {
	return db.PostgresClient.Create(redeemEvent).Error
}
