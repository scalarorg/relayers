package db

import (
	"context"
	"fmt"
	"time"

	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupTestDB(busEventChan chan *events.EventEnvelope, receiverChanBufSize int) (*DatabaseAdapter, func(), error) {
	ctx := context.Background()

	dbName := "test_db"
	dbUser := "test_user"
	dbPassword := "test_password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, nil, err
	}

	// Get the container's host and port
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		return nil, nil, err
	}
	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		return nil, nil, err
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, dbUser, dbPassword, dbName, port.Int())

	postgresDb, err := gorm.Open(postgresDriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	// Auto migrate the schema
	err = postgresDb.AutoMigrate(
		&models.RelayData{},
		&models.CallContract{},
		&models.CallContractApproved{},
		&models.CommandExecuted{},
		&models.Operatorship{},
	)
	if err != nil {
		return nil, nil, err
	}

	// Mock up data
	// Create test data
	relayData := models.RelayData{
		ID:     "test-id",
		From:   "cosmos",
		To:     "sepolia",
		Status: int(PENDING),
		CallContract: &models.CallContract{
			PayloadHash:     "0000000000000000000000000000000000000000000000000000000000000000",
			SourceAddress:   "0x24a1db57fa3ecafcbad91d6ef068439aceeae090",
			ContractAddress: "0x0000000000000000000000000000000000000000",
			Payload:         []byte("test-payload"),
		},
	}

	// Create another test data with different status
	relayData2 := models.RelayData{
		ID:     "test-id-2",
		From:   "cosmos",
		To:     "sepolia",
		Status: int(APPROVED),
		CallContract: &models.CallContract{
			PayloadHash:     "0000000000000000000000000000000000000000000000000000000000000000",
			SourceAddress:   "0x24a1db57fa3ecafcbad91d6ef068439aceeae090",
			ContractAddress: "0x0000000000000000000000000000000000000000",
			Payload:         []byte("test-payload-2"),
		},
	}

	// Insert test data
	result := postgresDb.Create(&relayData)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	result = postgresDb.Create(&relayData2)
	if result.Error != nil {
		return nil, nil, result.Error
	}

	// Create test DatabaseAdapter
	testAdapter := &DatabaseAdapter{
		PostgresClient: postgresDb,
		// BusEventChan:         busEventChan,
		// BusEventReceiverChan: make(chan *types.EventEnvelope, receiverChanBufSize),
	}

	// Return cleanup function
	cleanup := func() {
		postgresContainer.Terminate(ctx)
	}

	return testAdapter, cleanup, nil
}
