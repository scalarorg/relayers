package db

import (
	"fmt"

	"github.com/scalarorg/data-models/chains"
)

func (db *DatabaseAdapter) SaveRedeemTxs(redeemTxs []*chains.RedeemTx) error {
	return db.PostgresClient.Save(redeemTxs).Error
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
