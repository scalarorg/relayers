package db

import (
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DatabaseClient *gorm.DB

func InitDatabaseClient() error {
	var err error
	DatabaseClient, err = NewDatabaseClient()
	if err != nil {
		return err
	}
	return nil
}

func NewDatabaseClient() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(viper.GetString("DATABASE_URL")), &gorm.Config{})
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

	return db, nil
}

// We don't need Connect and Disconnect methods with GORM
