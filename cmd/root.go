package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	_ "github.com/scalarorg/relayers/internal/codec"
	"github.com/scalarorg/relayers/internal/relayer"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	AccountAddressPrefix   = "scalar"
	AccountPubKeyPrefix    = AccountAddressPrefix + types.PrefixPublic
	ValidatorAddressPrefix = AccountAddressPrefix + types.PrefixValidator + types.PrefixOperator
	ValidatorPubKeyPrefix  = AccountAddressPrefix + types.PrefixValidator + types.PrefixOperator + types.PrefixPublic
	ConsNodeAddressPrefix  = AccountAddressPrefix + types.PrefixValidator + types.PrefixConsensus
	ConsNodePubKeyPrefix   = AccountAddressPrefix + types.PrefixValidator + types.PrefixConsensus + types.PrefixPublic
	BaseAsset              = "ascal"
	environment            string
	rootCmd                = &cobra.Command{
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
	log.Info().Msgf("Running relayer with environment: %s", environment)
	if err := config.LoadEnv(environment); err != nil {
		panic("Failed to load config: " + err.Error())
	}
	evtBusConfig := config.EventBusConfig{}
	eventBus := events.GetEventBus(&evtBusConfig)
	// Initialize relayer service
	service, err := relayer.NewService(&config.GlobalConfig, eventBus)
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
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&environment, "env", "", "Environment name to  to the configuration file")
	viper.BindPFlag("env", rootCmd.PersistentFlags().Lookup("env"))
	//Set account prefix for scalar
	setCosmosAccountPrefix()
}

func setCosmosAccountPrefix() {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount(AccountAddressPrefix, AccountPubKeyPrefix)
	config.SetBech32PrefixForValidator(ValidatorAddressPrefix, ValidatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(ConsNodeAddressPrefix, ConsNodePubKeyPrefix)
}
