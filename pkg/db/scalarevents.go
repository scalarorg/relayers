package db

import (
	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for MintCommand operations
// -------------------------------------------------------------------------------------------------
func (d *DatabaseAdapter) CreateOrUpdateMintCommand(cmd *chains.MintCommand) error {
	return d.PostgresClient.Save(cmd).Error
}

//	func (d *DatabaseAdapter) UpdateMintCommandStatus(id string, status string, txHash string) error {
//		return d.PostgresClient.Model(&models.MintCommand{}).
//			Where("id = ?", id).
//			Updates(map[string]interface{}{
//				"status":  status,
//				"tx_hash": txHash,
//			}).Error
//	}
func (d *DatabaseAdapter) CreateOrUpdateMintCommands(cmdModels []chains.MintCommand) error {
	return d.PostgresClient.Save(cmdModels).Error
}

func (d *DatabaseAdapter) GetMintCommand(id string) (*chains.MintCommand, error) {
	var cmd chains.MintCommand
	err := d.PostgresClient.Where("id = ?", id).First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for TokenSentApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) SaveTokenSentApproveds(approvals []scalarnet.TokenSentApproved) error {
	eventIds := make([]string, len(approvals))
	for i, approval := range approvals {
		eventIds[i] = approval.EventID
	}
	err := db.PostgresClient.Transaction(func(tx *gorm.DB) error {
		result := tx.Save(approvals)
		if result.Error != nil {
			return result.Error
		}
		result = tx.Model(&chains.TokenSent{}).Where("event_id IN (?)", eventIds).Updates(map[string]interface{}{"status": chains.TokenSentStatusApproved})
		if result.Error != nil {
			return result.Error
		}
		return nil
	})
	return err
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApprovedWithMint operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApprovedWithMints(approvals []scalarnet.ContractCallApprovedWithMint) error {
	return db.PostgresClient.Save(approvals).Error
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApproveds(approvals []scalarnet.ContractCallApproved) error {
	return db.PostgresClient.Save(approvals).Error
}
