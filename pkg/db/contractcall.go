package db

import (
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/db/models"
)

func (db *DatabaseAdapter) UpdateContractCallApproved(messageID string, executeHash string) error {
	updateData := map[string]interface{}{
		"execute_hash": executeHash,
		"status":       APPROVED,
	}
	record := db.PostgresClient.Model(&models.RelayData{}).Where("id = ?", messageID).Updates(updateData)
	if record.Error != nil {
		return record.Error
	}
	log.Info().Msgf("[DatabaseAdapter] [UpdateContractCallApproved]: RelayData[%s]", messageID)
	return nil
}
func (db *DatabaseAdapter) FindContractCallByCommnadId(commandId string) (*models.CallContract, error) {
	var contractCall models.CallContract
	result := db.PostgresClient.Where("command_id = ?", commandId).First(&contractCall)
	if result.Error != nil {
		return nil, result.Error
	}
	return &contractCall, nil
}

// func (db *DatabaseAdapter) FindCosmosToEvmCallContractApproved(event *types.EvmEvent[*contracts.IScalarGatewayContractCallApproved]) ([]types.FindCosmosToEvmCallContractApproved, error) {
// 	var datas []models.RelayData

// 	result := db.PostgresClient.
// 		Preload("CallContract").
// 		Joins("JOIN call_contracts ON relay_data.id = call_contracts.relay_data_id").
// 		Where("call_contracts.payload_hash = ? AND call_contracts.source_address = ? AND call_contracts.contract_address = ? AND relay_data.status IN ?",
// 			strings.ToLower(hex.EncodeToString(event.Args.PayloadHash[:])),
// 			strings.ToLower(event.Args.SourceAddress),
// 			strings.ToLower(event.Args.ContractAddress.String()),
// 			[]int{int(types.PENDING), int(types.APPROVED)}).
// 		Order("relay_data.updated_at desc").
// 		Find(&datas)

// 	if result.Error != nil {
// 		return nil, result.Error
// 	}

// 	mappedResult := make([]types.FindCosmosToEvmCallContractApproved, len(datas))

// 	for i, data := range datas {
// 		mappedResult[i] = types.FindCosmosToEvmCallContractApproved{
// 			ID:      data.ID,
// 			Payload: data.CallContract.Payload,
// 		}
// 	}

// 	return mappedResult, nil
// }
