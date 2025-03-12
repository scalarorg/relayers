package db_test

import (
	"os"
	"testing"
	"time"

	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*db.DatabaseAdapter, func()) {
	// Use test database connection string
	testDBConnString := os.Getenv("TEST_DB_CONNECTION_STRING")
	if testDBConnString == "" {
		testDBConnString = "host=localhost user=postgres password=postgres dbname=relayer port=5432 sslmode=disable TimeZone=UTC"
	}

	cfg := &config.Config{
		ConnnectionString: testDBConnString,
	}

	db, err := db.NewDatabaseAdapter(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Return cleanup function
	cleanup := func() {
		sqlDB, err := db.PostgresClient.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

func TestNewDatabaseAdapter(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty connection string",
			config: &config.Config{
				ConnnectionString: "",
			},
			expectError: true,
		},
		{
			name: "valid config",
			config: &config.Config{
				ConnnectionString: "host=localhost user=postgres password=postgres dbname=relayer port=5432 sslmode=disable",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := db.NewDatabaseAdapter(tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}

func TestSaveCommands(t *testing.T) {
	_db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		commands    []*scalarnet.Command
		batchSize   int
		expectError bool
		setup       func(t *testing.T, _db *db.DatabaseAdapter)
		verify      func(t *testing.T, _db *db.DatabaseAdapter)
	}{
		{
			name: "successful batch insert",
			commands: []*scalarnet.Command{
				{
					CommandID:      "cmd1",
					BatchCommandID: "batch1",
					ChainID:        "chain1",
					CreatedAt:      time.Now(),
				},
				{
					CommandID:      "cmd2",
					BatchCommandID: "batch2",
					ChainID:        "chain2",
					CreatedAt:      time.Now(),
				},
			},
			batchSize:   100,
			expectError: false,
			verify: func(t *testing.T, db *db.DatabaseAdapter) {
				var count int64
				err := db.PostgresClient.Model(&scalarnet.Command{}).Count(&count).Error
				assert.NoError(t, err)
				assert.Equal(t, int64(2), count)
			},
		},
		{
			name: "upsert existing command",
			commands: []*scalarnet.Command{
				{
					CommandID:      "cmd1",
					BatchCommandID: "batch200", // Updated batch ID
					ChainID:        "chain200", // Updated chain ID
					CreatedAt:      time.Now(),
				},
			},
			batchSize:   100,
			expectError: false,
			setup: func(t *testing.T, db *db.DatabaseAdapter) {
				// Insert initial command
				cmd := &scalarnet.Command{
					CommandID:      "cmd1",
					BatchCommandID: "batch1",
					ChainID:        "chain1",
					CreatedAt:      time.Now(),
				}
				err := db.SaveCommands([]*scalarnet.Command{cmd})
				require.NoError(t, err)
			},
			verify: func(t *testing.T, db *db.DatabaseAdapter) {
				var cmd scalarnet.Command
				err := db.PostgresClient.Where("command_id = ?", "cmd1").First(&cmd).Error
				assert.NoError(t, err)
				assert.Equal(t, "batch2", cmd.BatchCommandID)
				assert.Equal(t, "chain2", cmd.ChainID)
			},
		},
		{
			name: "custom batch size",
			commands: []*scalarnet.Command{
				{
					CommandID:      "cmd3",
					BatchCommandID: "batch3",
					ChainID:        "chain3",
					CreatedAt:      time.Now(),
				},
				{
					CommandID:      "cmd4",
					BatchCommandID: "batch3",
					ChainID:        "chain3",
					CreatedAt:      time.Now(),
				},
			},
			batchSize:   1,
			expectError: false,
			verify: func(t *testing.T, db *db.DatabaseAdapter) {
				var count int64
				err := db.PostgresClient.Model(&scalarnet.Command{}).Where("batch_command_id = ?", "batch3").Count(&count).Error
				assert.NoError(t, err)
				assert.Equal(t, int64(2), count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			// err := _db.PostgresClient.Exec("TRUNCATE TABLE commands").Error
			// require.NoError(t, err)

			// Run setup if provided
			if tt.setup != nil {
				tt.setup(t, _db)
			}

			// Execute test
			err := _db.SaveCommands(tt.commands, db.BatchOption{BatchSize: tt.batchSize})
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Run verification if provided
			if tt.verify != nil {
				tt.verify(t, _db)
			}
		})
	}
}
