package db

import (
	"fmt"

	"github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbAdapter *DatabaseAdapter

type DatabaseAdapter struct {
	PostgresClient *gorm.DB
}

func NewDatabaseAdapter(config *config.Config) (*DatabaseAdapter, error) {
	if dbAdapter == nil {
		pgClient, err := NewPostgresClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres client: %w", err)
		}
		dbAdapter = &DatabaseAdapter{
			PostgresClient: pgClient,
		}
	}
	return dbAdapter, nil
}

func NewPostgresClient(config *config.Config) (*gorm.DB, error) {
	if config == nil || config.ConnnectionString == "" {
		return nil, fmt.Errorf("config is nil or connnection string is empty")
	}

	db, err := SetupDatabase(config.ConnnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	return db, nil
}

func SetupDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	err = RunMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	// Convert tables to hyper tables
	if err := InitHyperTables(db); err != nil {
		return nil, fmt.Errorf("failed to initialize hyper tables: %w", err)
	}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&chains.TokenSent{},
		&chains.MintCommand{},
		&chains.CommandExecuted{},
		&chains.ContractCall{},
		&chains.ContractCallWithToken{},
		&scalarnet.Command{},
		&scalarnet.CallContractWithToken{},
		&scalarnet.TokenSentApproved{},
		&scalarnet.ContractCallApprovedWithMint{},
		&scalarnet.EventCheckPoint{},
	)
}

func InitHyperTables(db *gorm.DB) error {
	// Convert tables with timestamp columns into hyper tables
	tables := []struct {
		name       string
		timeColumn string
	}{
		{"commands", "created_at"},
		{"token_sents", "created_at"},
		// Add other tables that need to be converted to hyper tables
	}

	for _, table := range tables {
		if err := CreateHyperTable(db, table.name, table.timeColumn); err != nil {
			return fmt.Errorf("failed to create hyper table for %s: %w", table.name, err)
		}
	}

	return nil
}

func CreateHyperTable(db *gorm.DB, tableName string, timeColumn string) error {
	sql := fmt.Sprintf(
		"SELECT create_hypertable('%s', by_range('%s'), if_not_exists => TRUE, migrate_data => TRUE);",
		tableName,
		timeColumn,
	)

	return db.Exec(sql).Error
}
