package electrs_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/go-electrum/electrum"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/pkg/clients/electrs"
)

var (
	electrsConfig *electrs.Config
	electrsClient *electrum.Client
)

func TestMain(m *testing.M) {
	// Load .env file
	err := godotenv.Load("../../../.env.test")
	if err != nil {
		log.Error().Err(err).Msg("Error loading .env.test file: %v")
	}
	electrumHost := os.Getenv("ELECTRUM_HOST")
	electrumPort, err := strconv.Atoi(os.Getenv("ELECTRUM_PORT"))
	if err != nil {
		log.Error().Err(err).Msgf("failed to convert electrum port to int: %v", err)
	}
	electrumUser := os.Getenv("ELECTRUM_USER")
	electrumPassword := os.Getenv("ELECTRUM_PASSWORD")
	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		log.Error().Err(err).Msgf("failed to convert batch size to int: %v", err)
	}
	confirmations, err := strconv.Atoi(os.Getenv("CONFIRMATIONS"))
	if err != nil {
		log.Error().Err(err).Msgf("failed to convert confirmations to int: %v", err)
	}
	lastVaultTx := os.Getenv("LAST_VAULT_TX")
	sourceChain := os.Getenv("SOURCE_CHAIN")

	electrsConfig = &electrs.Config{
		Host:          electrumHost,
		Port:          electrumPort,
		User:          electrumUser,
		Password:      electrumPassword,
		BatchSize:     batchSize,
		Confirmations: confirmations,
		LastVaultTx:   lastVaultTx,
		SourceChain:   sourceChain,
	}
	rpcEndpoint := fmt.Sprintf("%s:%d", electrsConfig.Host, electrsConfig.Port)
	electrsClient, _ = electrum.Connect(&electrum.Options{
		Dial: func() (net.Conn, error) {
			return net.DialTimeout("tcp", rpcEndpoint, time.Second)
		},
		MethodTimeout:   time.Second,
		PingInterval:    -1,
		SoftwareVersion: "scalar-relayer",
	})
	if err != nil {
		log.Error().Err(err).Msgf("failed to create electrum client: %v", err)
	}
	os.Exit(m.Run())
}

func TestElectrsSubscription(t *testing.T) {
	// Create test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	log.Debug().Msg("[ElectrumClient] [Start] Subscribing to new block event for request to confirm if vault transaction is get enought confirmation")
	//electrsClient.BlockchainHeaderSubscribe(ctx, BlockchainHeaderHandler)
	params := []interface{}{}
	//Set batch size from config or default value
	params = append(params, electrsConfig.BatchSize)
	log.Debug().Msgf("[ElectrumClient] [Start] Subscribing to vault transactions with params: %v", params)
	electrsClient.VaultTransactionSubscribe(ctx, VaultTxMessageHandler, params...)
	select {}
}

func BlockchainHeaderHandler(header *types.BlockchainHeader, err error) error {
	log.Debug().Msgf("[ElectrumClient] [BlockchainHeaderHandler] Received header: %v", header)
	return nil
}

func VaultTxMessageHandler(vaultTxs []types.VaultTransaction, err error) error {
	log.Debug().Msgf("[ElectrumClient] [VaultTxMessageHandler] Received %d vault transactions", len(vaultTxs))
	for _, vaultTx := range vaultTxs {
		log.Debug().Msgf("[ElectrumClient] [VaultTxMessageHandler] Received vault transaction: %v", vaultTx)
	}
	return nil
}
