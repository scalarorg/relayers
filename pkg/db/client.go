package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DbClient *DatabaseClient

type DatabaseClient struct {
	postgresClient *gorm.DB
	mongoClient    *mongo.Client
	mongoDatabase  *mongo.Database
}

func InitDatabaseClient() error {
	var err error
	DbClient.postgresClient, err = NewPostgresClient()
	if err != nil {
		return err
	}
	DbClient.mongoClient, DbClient.mongoDatabase, err = NewMongoClient()
	if err != nil {
		return err
	}
	return nil
}

func NewPostgresClient() (*gorm.DB, error) {
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

func NewMongoClient() (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create client options
	clientOptions := options.Client().ApplyURI(viper.GetString("MONGODB_URI"))

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Info().Msg("Connected to MongoDB")

	// Get database instance
	database := client.Database(viper.GetString("MONGODB_DATABASE"))

	return client, database, nil
}
