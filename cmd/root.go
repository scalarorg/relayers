package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/internal/relayer"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	environment string
	rootCmd     = &cobra.Command{
		Use:   "relayer",
		Short: "Scalar Relayer",
		Run:   run,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	// Initialize logger
	config.InitLogger()
	// Initialize OpenObserve
	initObserve()
	// Load and initialize global config
	if err := config.Load(environment); err != nil {
		panic("Failed to load config: " + err.Error())
	}

	// Initialize global DatabaseClient
	eventBus := events.GetEventBus(&config.GlobalConfig.EventBus)
	dbAdapter, err := db.NewDatabaseAdapter(config.GlobalConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create database adapter")
	}

	// Initialize relayer service
	service, err := relayer.NewService(config.GlobalConfig, dbAdapter, eventBus)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create relayer service")
	}

	// Start relayer service
	err = service.Start(context.TODO())
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

func initObserve() {
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
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&environment,
		"env",
		"local",
		"Environment name to  to the configuration file",
	)
	viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
}
