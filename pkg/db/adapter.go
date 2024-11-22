package db

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbAdapter *DatabaseAdapter

type DatabaseAdapter struct {
	PostgresClient *gorm.DB
	// MongoClient    *mongo.Client
	// MongoDatabase  *mongo.Database
	// BusEventChan         chan *types.EventEnvelope
	// BusEventReceiverChan chan *types.EventEnvelope
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

// func InitDatabaseAdapter(config *config.Config, busEventChan chan *types.EventEnvelope, receiverChanBufSize int) error {
// 	DbAdapter = &DatabaseAdapter{
// 		BusEventChan:         busEventChan,
// 		BusEventReceiverChan: make(chan *types.EventEnvelope, receiverChanBufSize),
// 	}

// 	var err error
// 	DbAdapter.PostgresClient, err = NewPostgresClient(config)
// 	if err != nil {
// 		return err
// 	}
// 	DbAdapter.MongoClient, DbAdapter.MongoDatabase, err = NewMongoClient()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (da *DatabaseAdapter) ListenEventsFromBusChannel() {
// 	for event := range da.BusEventReceiverChan {
// 		switch event.Component {
// 		case "DbAdapter":
// 			fmt.Printf("Received event in database: %+v\n", event)
// 			da.handleDatabaseEvent(*event)
// 		default:
// 			// Pass the event that not belong to DbAdapter
// 		}
// 	}
// }

// func (da *DatabaseAdapter) handleDatabaseEvent(eventEnvelope types.EventEnvelope) {
// 	switch eventEnvelope.Handler {
// 	case "FindCosmosToEvmCallContractApproved":
// 		// Handle store operations
// 		relayDatas, err := da.FindCosmosToEvmCallContractApproved(eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]))
// 		if err != nil {
// 			log.Error().Err(err).Msg("[DatabaseAdapter] [FindCosmosToEvmCallContractApproved] Error finding Cosmos to Evm Call Contract Approved")
// 		}

// 		if len(relayDatas) > 0 {
// 			err = da.CreateEvmContractCallApprovedEvent(eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]))
// 			if err != nil {
// 				log.Error().Err(err).Msg("[DatabaseAdapter] [FindCosmosToEvmCallContractApproved] Error creating Evm Contract Call Approved")
// 			}

// 			// Create and send new event to EVMAdapter
// 			nextEnvelopeData := types.HandleCosmosToEvmCallContractCompleteEventData{
// 				Event:      eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayContractCallApproved]),
// 				RelayDatas: relayDatas,
// 			}
// 			evmEvent := types.EventEnvelope{
// 				Component:          "EvmAdapter",
// 				ReceiverClientName: eventEnvelope.SenderClientName,
// 				Handler:            "handleCosmosToEvmCallContractCompleteEvent",
// 				Data:               nextEnvelopeData,
// 			}
// 			// Send the envelope to the channel
// 			da.SendEvent(&evmEvent)
// 		}
// 	case "UpdateEventStatus":
// 		id := eventEnvelope.Data.(types.HandleCosmosToEvmCallContractCompleteEventExecuteResult).ID
// 		status := eventEnvelope.Data.(types.HandleCosmosToEvmCallContractCompleteEventExecuteResult).Status
// 		da.UpdateEventStatus(id, status)
// 	case "CreateEvmCallContractEvent":
// 		_, hash, err := da.CreateEvmCallContractEvent(eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayContractCall]))
// 		if err != nil {
// 			log.Error().Err(err).Msg("[DatabaseAdapter] [CreateEvmCallContractEvent] Error creating Evm Contract Call")
// 		}
// 		// Send the hash to the next handler
// 		nextEnvelopeData := types.WaitForTransactionData{
// 			Hash:  hash,
// 			Event: eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayContractCall]),
// 		}
// 		evmEvent := types.EventEnvelope{
// 			Component:          "EvmAdapter",
// 			ReceiverClientName: eventEnvelope.SenderClientName,
// 			Handler:            "waitForTransaction",
// 			Data:               nextEnvelopeData,
// 		}
// 		// Send the envelope to the channel
// 		da.SendEvent(&evmEvent)
// 	case "CreateEvmExecutedEvent":
// 		da.CreateEvmExecutedEvent(eventEnvelope.Data.(*types.EvmEvent[*contracts.IAxelarGatewayExecuted]))

// 	// Add more handlers as needed
// 	default:
// 		log.Warn().
// 			Str("handler", eventEnvelope.Handler).
// 			Msg("Unknown database event handler")
// 	}
// }

// func (da *DatabaseAdapter) SendEvent(event *types.EventEnvelope) {
// 	da.BusEventChan <- event
// 	log.Debug().Msgf("[DatabaseAdapter] Sent event to channel: %v", event.Handler)
// }

func NewPostgresClient(config *config.Config) (*gorm.DB, error) {
	if config == nil || config.ConnnectionString == "" {
		return nil, fmt.Errorf("config is nil or connnection string is empty")
	}
	db, err := gorm.Open(postgres.Open(config.ConnnectionString), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto Migrate the schema
	err = db.AutoMigrate(
		&models.RelayData{},
		&models.CallContract{},
		&models.CallContractWithToken{},
		&models.CallContractApproved{},
		&models.CommandExecuted{},
		&models.ProtocolInfo{},
		&models.EventCheckPoint{},
	)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func NewMongoClient() (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := viper.GetString("MONGODB_URI")
	log.Info().Str("mongodb_uri", mongoURI).Msg("Initial MongoDB URI")

	// Create base client options
	clientOptions := options.Client().ApplyURI(mongoURI)

	// Set direct connection for local environment
	chainEnv := viper.GetString("CHAIN_ENV")
	if chainEnv == "local" {
		log.Info().Msg("Local environment detected, using direct MongoDB connection")
		clientOptions.SetDirect(true)
	}

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
