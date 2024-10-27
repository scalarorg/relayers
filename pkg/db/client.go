package db

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	db *gorm.DB
}

var DatabaseClient *Client

func InitDatabaseClient() error {
	var err error
	DatabaseClient, err = NewDatabaseClient()
	if err != nil {
		return err
	}
	return nil
}

func NewDatabaseClient() (*Client, error) {
	db, err := gorm.Open(postgres.Open(config.GlobalConfig.Database.URL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto Migrate the schema
	err = db.AutoMigrate(
		&models.RelayData{},
		&models.CallContract{},
		&models.CallContractApproved{},
		&models.Approved{},
		&models.CommandExecuted{},
		&models.Operatorship{},
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		db: db,
	}, nil
}

func (c *Client) CreateBtcCallContractEvent(event *types.BtcEventTransaction) error {
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

	result := c.db.Create(&relayData)
	return result.Error
}

// We don't need Connect and Disconnect methods with GORM
