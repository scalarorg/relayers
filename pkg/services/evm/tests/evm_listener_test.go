package evm_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/services/evm"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*db.DatabaseAdapter, func()) {
	ctx := context.Background()

	dbName := "users"
	dbUser := "user"
	dbPassword := "password"

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
	require.NoError(t, err)

	// Get the container's host and port
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, dbUser, dbPassword, dbName, port.Int())

	postgresDb, err := gorm.Open(postgresDriver.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate the schema
	err = postgresDb.AutoMigrate(
		&models.RelayData{},
		&models.CallContract{},
		&models.CallContractApproved{},
		&models.CommandExecuted{},
		&models.Operatorship{},
	)
	require.NoError(t, err)

	// Create test DatabaseAdapter
	testAdapter := &db.DatabaseAdapter{
		PostgresClient: postgresDb,
		EventChan:      make(chan *types.EventEnvelope, 100),
	}
	db.DbAdapter = testAdapter

	// Return cleanup function
	cleanup := func() {
		postgresContainer.Terminate(ctx)
	}

	return testAdapter, cleanup
}

func TestEvmListener(t *testing.T) {
	// Setup test database
	dbAdapter, cleanup := setupTestDB(t)
	defer cleanup() // This ensures the container is cleaned up after the test

	eventChan := make(chan *types.EventEnvelope, 100)

	evmListener, err := evm.NewEvmAdapter(config.EvmNetworkConfig{
		Name:       "sepolia",
		RPCUrl:     "https://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
		Gateway:    "0x2bb588d7bb6faAA93f656C3C78fFc1bEAfd1813D",
		PrivateKey: "81271046a6de40cea0798935b391ddcf1e57646db7273228fd0d7e147154aaaa",
		ChainID:    "11155111",
	}, eventChan)
	require.NoError(t, err)

	// Start listening for events in a separate goroutine
	go func() {
		for event := range eventChan {
			t.Logf("Received event: %+v", event)
			if event.Component == "DbAdapter" {
				dbAdapter.EventChan <- event
			}
		}
	}()

	// Start listening for database events
	go dbAdapter.ListenEvents()

	evmListener.PollForContractCallApproved()
}
