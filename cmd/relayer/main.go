package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/relayer"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/types"
	"github.com/spf13/viper"
)

func main() {
	// Load environment variables into viper
	if err := config.LoadEnv(); err != nil {
		panic("Failed to load environment variables: " + err.Error())
	}

	// // Initialize OpenObserve
	// appName := viper.GetString("APP_NAME")
	// if viper.GetBool("IS_DEV") {
	// 	appName = appName + "-dev"
	// }
	// openobserve.Init(openobserve.OpenObserveConfig{
	// 	Endpoint:    viper.GetString("OPENOBSERVE_ENDPOINT"),
	// 	Credential:  viper.GetString("OPENOBSERVE_CREDENTIAL"),
	// 	ServiceName: appName,
	// 	Env:         viper.GetString("ENV"),
	// })

	// Initialize logger
	config.InitLogger()

	// Load and initialize global config
	if err := config.Load(); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	// Initialize Event Bus channel
	eventChan := make(chan *types.EventEnvelope, viper.GetInt("CHANNEL_BUFFER_SIZE"))

	// Initialize global DatabaseClient
	err := db.InitDatabaseAdapter(config.GlobalConfig, eventChan, viper.GetInt("CHANNEL_BUFFER_SIZE"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create database adapter")
	}

	// Initialize relayer service
	service, err := relayer.NewService(config.GlobalConfig, eventChan, viper.GetInt("CHANNEL_BUFFER_SIZE"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create relayer service")
	}

	// Start relayer service
	err = service.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start relayer service")
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down relayer...")
	service.Stop()
}
