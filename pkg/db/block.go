package db

import (
	"github.com/scalarorg/data-models/chains"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (db *DatabaseAdapter) FindBlockHeader(chainId string, blockNumber uint64) (*chains.BlockHeader, error) {
	var blockHeader chains.BlockHeader
	result := db.PostgresClient.Where("chain = ? AND block_number = ?", chainId, blockNumber).First(&blockHeader)
	if result.Error != nil {
		return nil, result.Error
	}
	return &blockHeader, nil
}
func (db *DatabaseAdapter) CreateBlockHeaders(blockHeaders []chains.BlockHeader) error {
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "chain"}, {Name: "block_number"}},
				DoUpdates: clause.AssignmentColumns([]string{"block_hash", "block_time"}),
			},
		).Create(blockHeaders).Error
		if err != nil {
			return err
		}
		return nil
	})
}
func (db *DatabaseAdapter) CreateBlockHeader(blockHeader *chains.BlockHeader) error {
	return db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "chain"}, {Name: "block_number"}},
				DoUpdates: clause.AssignmentColumns([]string{"block_hash", "block_time"}),
			},
		).Create(blockHeader).Error

		if err != nil {
			return err
		}
		//Update token sent block_time
		err = tx.Model(&chains.TokenSent{}).
			Where("block_number = ? AND source_chain = ?", blockHeader.BlockNumber, blockHeader.Chain).
			Update("block_time", blockHeader.BlockTime).Error
		if err != nil {
			return err
		}
		//Update contract call with token
		err = tx.Model(&chains.ContractCallWithToken{}).
			Where("block_number = ? AND source_chain = ?", blockHeader.BlockNumber, blockHeader.Chain).
			Update("block_time", blockHeader.BlockTime).Error
		if err != nil {
			return err
		}
		//Update Redeem token block_time
		err = tx.Model(&chains.EvmRedeemTx{}).
			Where("block_number = ? AND source_chain = ?", blockHeader.BlockNumber, blockHeader.Chain).
			Update("block_time", blockHeader.BlockTime).Error
		if err != nil {
			return err
		}
		//Update command executed block_time
		err = tx.Model(&chains.CommandExecuted{}).
			Where("block_number = ? AND source_chain = ?", blockHeader.BlockNumber, blockHeader.Chain).
			Update("block_time", blockHeader.BlockTime).Error
		if err != nil {
			return err
		}
		return nil
	})
}

func (db *DatabaseAdapter) GetBlockTime(chainId string, blockNumbers []uint64) (map[uint64]uint64, error) {
	var blockHeaders []*chains.BlockHeader
	result := db.PostgresClient.Where("chain = ? AND block_number IN ?", chainId, blockNumbers).Find(&blockHeaders)
	if result.Error != nil {
		return nil, result.Error
	}
	blockTimeMap := make(map[uint64]uint64)
	for _, blockHeader := range blockHeaders {
		blockTimeMap[blockHeader.BlockNumber] = blockHeader.BlockTime
	}
	return blockTimeMap, nil
}
