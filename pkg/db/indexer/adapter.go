package indexer

// type IndexerDatabaseAdapter struct {
// 	DbClient *gorm.DB
// }

// func NewIndexerDatabaseAdapter(connectionString string) (*IndexerDatabaseAdapter, error) {
// 	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &IndexerDatabaseAdapter{DbClient: db}, nil
// }

// func (db *IndexerDatabaseAdapter) GetLastSwitchedPhases(groupUid string) ([]*chains.SwitchedPhase, error) {
// 	var switchedPhases []*chains.SwitchedPhase
// 	query := `
// 	    SELECT * FROM switched_phases INNER JOIN (
// 			SELECT source_chain, max(session_sequence) as session_sequence
// 			FROM switched_phases
// 			WHERE custodian_group_uid = $1
// 			GROUP BY source_chain
// 		) AS last_session
// 		ON switched_phases.source_chain = last_session.source_chain
// 		AND switched_phases.session_sequence = last_session.session_sequence
// 	`
// 	err := db.DbClient.Raw(query, groupUid).Scan(&switchedPhases).Error
// 	return switchedPhases, err
// }

// func (db *IndexerDatabaseAdapter) GetLastRedeemTxs(groupUid string, sessionSequence uint64) ([]*chains.EvmRedeemTx, error) {
// 	var redeemTxs []*chains.EvmRedeemTx
// 	query := `
// 		SELECT * FROM evm_redeem_txes
// 		WHERE custodian_group_uid = $1
// 		AND session_sequence = $2
// 	`
// 	err := db.DbClient.Raw(query, groupUid, sessionSequence).Scan(&redeemTxs).Error
// 	return redeemTxs, err
// }

// // GetRedeemTokenEventsByGroupAndSequence retrieves redeem token events from database by group UID and session sequence
// func (db *IndexerDatabaseAdapter) GetRedeemTokenEventsByGroupAndSequence(groupUid string, sessionSequence uint64) ([]*chains.EvmRedeemTx, error) {
// 	var redeemTxs []*chains.EvmRedeemTx
// 	query := `
// 		SELECT * FROM evm_redeem_txes
// 		WHERE custodian_group_uid = $1
// 		AND session_sequence = $2
// 		ORDER BY block_number ASC, log_index ASC
// 	`
// 	err := db.DbClient.Raw(query, groupUid, sessionSequence).Scan(&redeemTxs).Error
// 	return redeemTxs, err
// }

// // GetUnexecutedVaultTxsByBlock retrieves unexecuted vault transactions for a specific block
// func (db *IndexerDatabaseAdapter) GetUnexecutedVaultTxsByBlock(blockNumber uint64) ([]*chains.VaultTransaction, error) {
// 	var vaultTxs []*chains.VaultTransaction
// 	query := `
// 		SELECT * FROM vault_transactions
// 		WHERE block_number = $1
// 		AND status = 'pending'
// 		ORDER BY tx_position ASC
// 	`
// 	err := db.DbClient.Raw(query, blockNumber).Scan(&vaultTxs).Error
// 	return vaultTxs, err
// }

// // GetUnexecutedTokenSentsByBlock retrieves unexecuted token sent events for a specific block
// func (db *IndexerDatabaseAdapter) GetUnexecutedTokenSentsByBlock(blockNumber uint64) ([]*chains.TokenSent, error) {
// 	var tokenSents []*chains.TokenSent
// 	query := `
// 		SELECT * FROM token_sents
// 		WHERE block_number = $1
// 		AND status = 'pending'
// 		ORDER BY log_index ASC
// 	`
// 	err := db.DbClient.Raw(query, blockNumber).Scan(&tokenSents).Error
// 	return tokenSents, err
// }

// // GetUnexecutedEvmRedeemTxsByBlock retrieves unexecuted EVM redeem transactions for a specific block
// func (db *IndexerDatabaseAdapter) GetUnexecutedEvmRedeemTxsByBlock(blockNumber uint64) ([]*chains.EvmRedeemTx, error) {
// 	var redeemTxs []*chains.EvmRedeemTx
// 	query := `
// 		SELECT * FROM evm_redeem_txes
// 		WHERE block_number = $1
// 		AND status = 'pending'
// 		ORDER BY log_index ASC
// 	`
// 	err := db.DbClient.Raw(query, blockNumber).Scan(&redeemTxs).Error
// 	return redeemTxs, err
// }

// // GetLatestBlockNumber retrieves the latest block number from vault transactions
// func (db *IndexerDatabaseAdapter) GetLatestBlockNumber() (uint64, error) {
// 	var blockNumber uint64
// 	query := `
// 		SELECT COALESCE(MAX(block_number), 0) FROM vault_transactions
// 	`
// 	err := db.DbClient.Raw(query).Scan(&blockNumber).Error
// 	return blockNumber, err
// }
