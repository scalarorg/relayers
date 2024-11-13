package db

import "github.com/scalarorg/relayers/pkg/db/models"

func (db *DatabaseAdapter) FindProtocolInfo(chainName string, contractAddress string) (*models.ProtocolInfo, error) {
	var protocolInfo models.ProtocolInfo

	//query := db.PostgresClient.Where("chain_name = ? AND smart_contract_address = ? AND token_contract_address = ?", chainName, contractAddress, tokenAddress)
	query := db.PostgresClient.Where("chain_name = ? AND smart_contract_address = ?", chainName, contractAddress)
	result := query.First(&protocolInfo)
	if result.Error != nil {
		return nil, result.Error
	}

	return &protocolInfo, nil
}
