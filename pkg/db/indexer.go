package db

import (
	"time"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/relayer"
	"gorm.io/gorm/clause"
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
	return db.RelayerClient.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "block_number"}},
		DoNothing: true,
	}).Create(vaultBlock).Error
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
		SELECT * FROM vault_blocks ORDER BY block_number DESC LIMIT 1
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
func (db *DatabaseAdapter) GetLastTokenSentBlock() (*relayer.TokenSentBlock, error) {
	var tokenSentBlock relayer.TokenSentBlock
	query := `
		SELECT * FROM token_sent_blocks ORDER BY block_number DESC LIMIT 1
	`
	err := db.RelayerClient.Raw(query).Scan(&tokenSentBlock).Error
	if err != nil {
		return nil, err
	}
	if tokenSentBlock.ID == 0 {
		return nil, nil
	}
	return &tokenSentBlock, nil
}

// Find if there is any un processed vault transaction in the last processing block
// This is expensive query, should be used only when there is no vault block processed
func (db *DatabaseAdapter) FindLatestUnprocessedVaultTransactions() ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	// Find maximum block number whose transactions appear in command_executeds table
	var maxBlockNumber uint64
	query := `
		SELECT max(vt.block_number) FROM vault_transactions vt join command_executeds ce 
		on vt.tx_hash = ce.command_id and 'evm|'||vt.destination_chain = ce.source_chain
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
			SELECT ce.command_id FROM command_executeds ce
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

// GetUnprocessedVaultTransactionsByBlock gets vault transactions that are not in command_executeds table
func (db *DatabaseAdapter) GetUnprocessedVaultTransactionsByBlock(blockNumber uint64) ([]*chains.VaultTransaction, error) {
	var vaultTxs []*chains.VaultTransaction
	query := `
		SELECT vt.* FROM vault_transactions vt
		WHERE vt.vault_tx_type = 1 AND vt.block_number = $1
		AND vt.tx_hash NOT IN (
			SELECT ce.command_id FROM command_executeds ce
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

func (db *DatabaseAdapter) FindLastExecutedCommands(chainId string) ([]*chains.CommandExecuted, error) {
	var commandExecuteds []*chains.CommandExecuted
	query := `
		SELECT * FROM command_executeds 
		WHERE source_chain = $1
		AND block_number = (SELECT COALESCE(MAX(block_number), 0) FROM command_executeds where source_chain = $1)
	`
	err := db.IndexerClient.Raw(query, chainId).Scan(&commandExecuteds).Error
	if err != nil {
		return nil, err
	}
	return commandExecuteds, nil
}

// Find if there is any un processed token sent transaction in the last processing block
func (db *DatabaseAdapter) FindLatestUnprocessedTokenSents() ([]*chains.TokenSent, error) {
	var tokenSents []*chains.TokenSent
	var maxBlockNumber uint64
	query := `
		SELECT max(ts.block_number) FROM token_sents ts join command_executeds ce on ts.tx_hash = ce.command_id
	`
	err := db.IndexerClient.Raw(query).Scan(&maxBlockNumber).Error
	if err != nil || maxBlockNumber == 0 {
		query = `
			SELECT ts.* FROM token_sents ts
			WHERE ts.block_number = (
				select min(block_number) from token_sents
			)
			ORDER BY ts.tx_position ASC
		`
		err = db.IndexerClient.Raw(query).Scan(&tokenSents).Error
		return tokenSents, err
	}
	query = `
		SELECT ts.* FROM token_sents ts
		WHERE ts.block_number = $1
		AND ts.tx_hash NOT IN (
			SELECT ce.command_id FROM command_executeds ce
			WHERE ce.command_id = ts.tx_hash
		)
		ORDER BY ts.tx_position ASC
	`
	err = db.IndexerClient.Raw(query, maxBlockNumber+1).Scan(&tokenSents).Error
	if err != nil || len(tokenSents) == 0 {
		return db.GetNextTokenSents(maxBlockNumber)
	} else {
		return tokenSents, nil
	}
}

func (db *DatabaseAdapter) GetUnprocessedTokenSentsByBlock(blockNumber uint64) ([]*chains.TokenSent, error) {
	var tokenSents []*chains.TokenSent
	query := `
		SELECT ts.* FROM token_sents ts
		WHERE ts.block_number = $1
		AND ts.tx_hash NOT IN (
			SELECT ce.command_id FROM command_executeds ce
			WHERE ce.command_id = ts.tx_hash
		)
		ORDER BY ts.tx_position ASC
	`
	err := db.IndexerClient.Raw(query, blockNumber).Scan(&tokenSents).Error
	return tokenSents, err
}

func (db *DatabaseAdapter) GetNextTokenSents(lastProcessedBlock uint64) ([]*chains.TokenSent, error) {
	var tokenSents []*chains.TokenSent
	query := `
		SELECT * FROM token_sents
		WHERE block_number = (
			SELECT MIN(block_number) 
			FROM token_sents 
			WHERE block_number > $1
		)
		ORDER BY ts.log_index ASC
	`
	err := db.IndexerClient.Raw(query, lastProcessedBlock).Scan(&tokenSents).Error
	if err != nil {
		return nil, err
	}
	return tokenSents, nil
}

func (db *DatabaseAdapter) CreateTokenSentBlock(tokenSentBlock *relayer.TokenSentBlock) error {
	return db.RelayerClient.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "block_number"}},
		DoNothing: true,
	}).Create(tokenSentBlock).Error
}
