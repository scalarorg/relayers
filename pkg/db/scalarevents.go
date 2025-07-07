package db

import (
	"github.com/scalarorg/data-models/scalarnet"
)

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for MintCommand operations
// -------------------------------------------------------------------------------------------------
func (d *DatabaseAdapter) CreateOrUpdateMintCommand(cmd *scalarnet.MintCommand) error {
	return d.RelayerClient.Save(cmd).Error
}

//	func (d *DatabaseAdapter) UpdateMintCommandStatus(id string, status string, txHash string) error {
//		return d.PostgresClient.Model(&models.MintCommand{}).
//			Where("id = ?", id).
//			Updates(map[string]interface{}{
//				"status":  status,
//				"tx_hash": txHash,
//			}).Error
//	}
func (d *DatabaseAdapter) CreateOrUpdateMintCommands(cmdModels []scalarnet.MintCommand) error {
	return d.RelayerClient.Save(cmdModels).Error
}

func (d *DatabaseAdapter) GetMintCommand(id string) (*scalarnet.MintCommand, error) {
	var cmd scalarnet.MintCommand
	err := d.RelayerClient.Where("id = ?", id).First(&cmd).Error
	if err != nil {
		return nil, err
	}
	return &cmd, nil
}

// TODO: add chunking to save token sent approveds
// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for TokenSentApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) SaveTokenSentApproveds(approvals []*scalarnet.TokenSentApproved) error {
	err := db.RelayerClient.Save(approvals).Error
	if err != nil {
		return err
	}

	return err
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApprovedWithMint operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApprovedWithMints(approvals []scalarnet.ContractCallApprovedWithMint) error {
	return db.RelayerClient.Save(approvals).Error
}

// -------------------------------------------------------------------------------------------------
// Add methods to DBAdapter for ContractCallApproved operations
// -------------------------------------------------------------------------------------------------
func (db *DatabaseAdapter) CreateOrUpdateContractCallApproveds(approvals []scalarnet.ContractCallApproved) error {
	return db.RelayerClient.Save(approvals).Error
}
