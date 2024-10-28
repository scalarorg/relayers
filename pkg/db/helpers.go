package db

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/types"
)

func CreateBtcCallContractEvent(event *types.BtcEventTransaction) error {
	id := fmt.Sprintf("%s-%d", strings.ToLower(event.TxHash), event.LogIndex)

	// Convert VaultTxHex and Payload to byte slices
	txHexBytes, err := hex.DecodeString(strings.TrimPrefix(event.VaultTxHex, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode VaultTxHex: %w", err)
	}
	payloadBytes, err := hex.DecodeString(strings.TrimPrefix(event.Payload, "0x"))
	if err != nil {
		return fmt.Errorf("failed to decode Payload: %w", err)
	}

	relayData := models.RelayData{
		ID:   id,
		From: event.SourceChain,
		To:   event.DestinationChain,
		CallContract: &models.CallContract{
			TxHash:          event.TxHash,
			TxHex:           txHexBytes,
			BlockNumber:     new(int),
			LogIndex:        new(int),
			ContractAddress: event.DestinationContractAddress,
			SourceAddress:   event.Sender,
			Amount:          new(string),
			Symbol:          new(string),
			Payload:         payloadBytes,
			PayloadHash:     event.PayloadHash,
			StakerPublicKey: &event.StakerPublicKey,
			SenderAddress:   new(string),
		},
	}

	*relayData.CallContract.BlockNumber = int(event.BlockNumber)
	*relayData.CallContract.LogIndex = int(event.LogIndex)
	*relayData.CallContract.Amount = event.MintingAmount
	*relayData.CallContract.Symbol = "" // Set an empty string for consistency
	*relayData.CallContract.SenderAddress = event.Sender

	log.Debug().
		Str("id", id).
		Msg("[DatabaseClient] Creating BtcCallContract")

	result := DatabaseClient.Create(&relayData)
	return result.Error
}

func FindRelayDataById(id string, includeCallContract, includeCallContractWithToken *bool) (*models.RelayData, error) {
	var relayData models.RelayData

	// Set default values if options are nil
	if includeCallContract == nil {
		defaultTrue := true
		includeCallContract = &defaultTrue
	}
	if includeCallContractWithToken == nil {
		defaultTrue := true
		includeCallContractWithToken = &defaultTrue
	}

	query := DatabaseClient.Where("id = ?", id)

	// Add preload conditions based on options
	if *includeCallContract {
		query = query.Preload("CallContract")
	}
	if *includeCallContractWithToken {
		query = query.Preload("CallContractWithToken")
	}

	result := query.First(&relayData)
	if result.Error != nil {
		return nil, result.Error
	}

	return &relayData, nil
}
