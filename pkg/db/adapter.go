package db

import (
	"fmt"

	"github.com/scalarorg/data-models/relayer"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DatabaseAdapter struct {
	RelayerClient *gorm.DB
	IndexerClient *gorm.DB
}

func NewDatabaseAdapter(config *config.Config) (*DatabaseAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	dbAdapter := &DatabaseAdapter{}
	var err error
	// Create postgres client for scalar network
	dbAdapter.RelayerClient, err = NewPostgresClient(config.ConnnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}
	// Create postgres client for indexer
	dbAdapter.IndexerClient, err = NewPostgresClient(config.IndexerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}
	return dbAdapter, nil
}

func NewPostgresClient(connectionString string) (*gorm.DB, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("connection string is empty")
	}

	db, err := SetupDatabase(connectionString)
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
	// if err := InitHyperTables(db); err != nil {
	// 	return nil, fmt.Errorf("failed to initialize hyper tables: %w", err)
	//}

	return db, nil
}

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&relayer.VaultTransaction{},
		&relayer.TokenSentBlock{},
		&relayer.ContractCall{},
		&relayer.ContractCallWithToken{},
		&relayer.EvmRedeemTx{},
		// &chains.BlockHeader{},
		// &chains.TokenSent{},
		// &chains.CommandExecuted{},
		// &chains.ContractCall{},
		// &chains.ContractCallWithToken{},
		// &chains.TokenDeployed{},
		//&chains.SwitchedPhase{},
		&scalarnet.Command{},
		&scalarnet.MintCommand{},
		&scalarnet.ScalarRedeemTokenApproved{},
		&scalarnet.CallContractWithToken{},
		&scalarnet.TokenSentApproved{},
		&scalarnet.ContractCallApprovedWithMint{},
		&scalarnet.EventCheckPoint{},
	)
}

// func InitHyperTables(db *gorm.DB) error {
// 	// Convert tables with timestamp columns into hyper tables
// 	tables := []struct {
// 		name       string
// 		timeColumn string
// 	}{
// 		{"commands", "created_at"},
// 		{"token_sents", "created_at"},
// 		// Add other tables that need to be converted to hyper tables
// 	}

// 	for _, table := range tables {
// 		if err := CreateHyperTable(db, table.name, table.timeColumn); err != nil {
// 			return fmt.Errorf("failed to create hyper table for %s: %w", table.name, err)
// 		}
// 	}

// 	return nil
// }

// func CreateHyperTable(db *gorm.DB, tableName string, timeColumn string) error {
// 	sql := fmt.Sprintf(
// 		"SELECT create_hypertable('%s', by_range('%s'), if_not_exists => TRUE, migrate_data => TRUE);",
// 		tableName,
// 		timeColumn,
// 	)

// 	return db.Exec(sql).Error
// }
