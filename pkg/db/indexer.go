package db

import (
	"time"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"gorm.io/gorm"
)

func (db *DatabaseAdapter) GetLastSwitchedPhases(groupUid string) ([]*chains.SwitchedPhase, error) {
	var switchedPhases []*chains.SwitchedPhase
	query := `
	    SELECT * FROM switched_phases INNER JOIN (
			SELECT source_chain, max(session_sequence) as session_sequence
			FROM switched_phases
			WHERE custodian_group_uid = $1
			GROUP BY source_chain
		) AS last_session
		ON switched_phases.source_chain = last_session.source_chain
		AND switched_phases.session_sequence = last_session.session_sequence
	`
	err := db.IndexerClient.Raw(query, groupUid).Scan(&switchedPhases).Error
	return switchedPhases, err
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

// VaultBlock operations
func (db *DatabaseAdapter) GetLastProcessedVaultBlock() (uint64, error) {
	var blockNumber uint64
	query := `
		SELECT COALESCE(MAX(block_number), 0) FROM vault_blocks WHERE processed_tx_count > 0
	`
	err := db.RelayerClient.Raw(query).Scan(&blockNumber).Error
	return blockNumber, err
}

func (db *DatabaseAdapter) UpdateVaultBlockStatus(blockNumber uint64, status string, broadcastTxHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == "processing" {
		now := time.Now()
		updates["processed_at"] = &now
	} else if status == "completed" {
		now := time.Now()
		updates["completed_at"] = &now
		updates["broadcast_tx_hash"] = broadcastTxHash
	}

	return db.RelayerClient.Model(&relayer.VaultBlock{}).
		Where("block_number = ?", blockNumber).
		Updates(updates).Error
}

func (db *DatabaseAdapter) CreateVaultBlock(vaultBlock *relayer.VaultBlock) error {
	return db.RelayerClient.Create(vaultBlock).Error
}

func (db *DatabaseAdapter) GetVaultBlockByNumber(blockNumber uint64) (*relayer.VaultBlock, error) {
	var vaultBlock relayer.VaultBlock
	err := db.RelayerClient.Where("block_number = ?", blockNumber).First(&vaultBlock).Error
	if err != nil {
		return nil, err
	}
	return &vaultBlock, nil
}

// GetLastUncompletedVaultBlock gets the next uncompleted vault block
func (db *DatabaseAdapter) GetLastVaultBlock() (*relayer.VaultBlock, error) {
	var vaultBlock relayer.VaultBlock
	query := `
		SELECT max(block_number) FROM vault_blocks
	`
	err := db.RelayerClient.Raw(query).Scan(&vaultBlock).Error
	if err != nil {
		return nil, err
	}
	if vaultBlock.ID == 0 {
		return nil, nil
	}
	return &vaultBlock, nil
}

// This is expensive query, should be used only when there is no vault block processed
func (db *DatabaseAdapter) FindLatestUnprocessedVaultTransactions() ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	// Find maximum block number whose transactions appear in command_executed table
	var maxBlockNumber uint64
	query := `
		SELECT max(vt.block_number) FROM vault_transactions vt join command_executed ce on vt.tx_hash = ce.command_id
	`
	err := db.IndexerClient.Raw(query).Scan(&maxBlockNumber).Error
	if err != nil || maxBlockNumber == 0 {
		// No vault block transaction executed yet, get all vault transactions in the first block
		query = `
			SELECT vt.* FROM vault_transactions vt
			WHERE vt.vault_tx_type = 1 AND vt.block_number = (
				select min(block_number) from vault_transactions where vault_tx_type = 1
			)
			ORDER BY vt.tx_position ASC
		`
		err = db.IndexerClient.Raw(query).Scan(&vaultTxs).Error
		return vaultTxs, err
	}
	// Get unprocessed transactions from the next block after the last processed block
	query = `
		SELECT vt.* FROM vault_transactions vt
		WHERE vt.vault_tx_type = 1 AND vt.block_number = $1
		AND vt.tx_hash NOT IN (
			SELECT ce.command_id FROM command_executed ce
			WHERE ce.command_id = vt.tx_hash
		)
		ORDER BY vt.tx_position ASC
	`
	err = db.IndexerClient.Raw(query, maxBlockNumber).Scan(&vaultTxs).Error
	if err != nil || len(vaultTxs) == 0 {
		// No unprocessed transactions found, return empty slice
		return db.GetNextVaultTransactions(maxBlockNumber)
	} else {
		return vaultTxs, nil
	}
}

// GetUnprocessedVaultTransactionsByBlock gets vault transactions that are not in command_executed table
func (db *DatabaseAdapter) GetUnprocessedVaultTransactionsByBlock(blockNumber uint64) ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	query := `
		SELECT vt.* FROM vault_transactions vt
		WHERE vt.vault_tx_type = 1 AND vt.block_number = $1
		AND vt.tx_hash NOT IN (
			SELECT ce.command_id FROM command_executed ce
			WHERE ce.command_id = vt.tx_hash
		)
		ORDER BY vt.tx_position ASC
	`
	err := db.IndexerClient.Raw(query, blockNumber).Scan(&vaultTxs).Error
	return vaultTxs, err
}

func (db *DatabaseAdapter) GetNextVaultTransactions(lastProcessedBlock uint64) ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	query := `
		SELECT * FROM vault_transactions
		WHERE vault_tx_type = 1 AND block_number = (
			SELECT MIN(block_number) 
			FROM vault_transactions 
			WHERE vault_tx_type = 1 AND block_number > $1
		)
		ORDER BY tx_position ASC
	`
	err := db.IndexerClient.Raw(query, lastProcessedBlock).Scan(&vaultTxs).Error
	if err != nil {
		return nil, err
	}
	return vaultTxs, nil
}

func (db *DatabaseAdapter) GetVaultTransactionsByBlock(blockNumber uint64) ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	query := `
		SELECT * FROM vault_transactions
		WHERE vault_tx_type = 1 AND block_number = $1
		ORDER BY tx_position ASC
	`
	err := db.IndexerClient.Raw(query, blockNumber).Scan(&vaultTxs).Error
	return vaultTxs, err
}

// IncrementProcessedTxCount increments the processed transaction count for a vault block
func (db *DatabaseAdapter) IncrementProcessedTxCount(blockNumber uint64) error {
	return db.RelayerClient.Model(&relayer.VaultBlock{}).
		Where("block_number = ?", blockNumber).
		UpdateColumn("processed_tx_count", gorm.Expr("processed_tx_count + ?", 1)).Error
}

// IsBlockFullyProcessed checks if all transactions in a block have been processed
func (db *DatabaseAdapter) IsBlockFullyProcessed(blockNumber uint64) (bool, error) {
	var vaultBlock relayer.VaultBlock
	err := db.RelayerClient.Select("transaction_count, processed_tx_count").
		Where("block_number = ?", blockNumber).
		First(&vaultBlock).Error
	if err != nil {
		return false, err
	}
	return vaultBlock.ProcessedTxCount >= vaultBlock.TransactionCount, nil
}

// MarkBlockAsCompleted marks a block as completed when all transactions are processed
func (db *DatabaseAdapter) MarkVaultBlockAsCompleted(blockNumber uint64, broadcastTxHash string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":            "completed",
		"completed_at":      &now,
		"broadcast_tx_hash": broadcastTxHash,
	}

	return db.RelayerClient.Model(&relayer.VaultBlock{}).
		Where("block_number = ?", blockNumber).
		Updates(updates).Error
}
