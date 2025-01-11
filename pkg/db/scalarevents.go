package db

import "github.com/scalarorg/relayers/pkg/db/models"

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for MintCommand operations
// -------------------------------------------------------------------------------------------------
func (d *DatabaseAdapter) CreateOrUpdateMintCommand(cmd *models.MintCommand) error {
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
func (d *DatabaseAdapter) CreateOrUpdateMintCommands(cmdModels []models.MintCommand) error {
	return d.PostgresClient.Save(cmdModels).Error
}

func (d *DatabaseAdapter) GetMintCommand(id string) (*models.MintCommand, error) {
	var cmd models.MintCommand
	err := d.PostgresClient.Where("id = ?", id).First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for TokenSentApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateTokenSentApproveds(approvals []models.TokenSentApproved) error {
	return db.PostgresClient.Save(approvals).Error
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApprovedWithMint operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApprovedWithMints(approvals []models.ContractCallApprovedWithMint) error {
	return db.PostgresClient.Save(approvals).Error
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApproveds(approvals []models.ContractCallApproved) error {
	return db.PostgresClient.Save(approvals).Error
}