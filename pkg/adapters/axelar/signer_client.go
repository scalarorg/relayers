package axelar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scalarorg/relayers/config"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"
)

// SignerClient handles Axelar network interactions
type SignerClient struct {
	client     client.Context
	keyring    keyring.Keyring
	maxRetries int
	retryDelay time.Duration
	fee        string
}

// NewSignerClient creates a new instance of SignerClient with the given configuration
func NewSignerClient() (*SignerClient, error) {
	// Get chain configuration
	chainID := config.GlobalConfig.Axelar.ChainID
	rpcEndpoint := config.GlobalConfig.Axelar.RPCUrl

	// Initialize client context
	clientCtx := client.Context{}.
		WithChainID(chainID).
		WithNodeURI(rpcEndpoint).
		WithBroadcastMode("sync")

	// Set up keyring
	kr, err := keyring.New(
		"axelar",
		keyring.BackendMemory,
		"",
		nil,
		clientCtx.Codec,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	// Import key using mnemonic
	mnemonic := config.GlobalConfig.Axelar.Mnemonic
	keyName := "default"

	_, err = kr.NewAccount(
		keyName,
		*mnemonic,
		"", // No additional password
		sdk.GetConfig().GetFullBIP44Path(),
		hd.Secp256k1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to import key: %w", err)
	}

	// Get values from config
	maxRetries := viper.GetInt("MAX_RETRY")
	retryDelay := time.Duration(viper.GetInt("RETRY_DELAY")) * time.Millisecond

	// Create client with imported key
	clientCtx = clientCtx.WithKeyring(kr)

	return &SignerClient{
		client:     clientCtx,
		keyring:    kr,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		fee:        "auto",
	}, nil
}

// GetAddress returns the signer's address
func (s *SignerClient) GetAddress() (string, error) {
	record, err := s.keyring.Key("default")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get key from keyring")
		return "", fmt.Errorf("failed to get key: %w", err)
	}

	address, err := record.GetAddress()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get address from keyring")
		return "", fmt.Errorf("failed to get address: %w", err)
	}

	log.Debug().Str("address", address.String()).Msg("Retrieved signer address")
	return address.String(), nil
}

// GetBalance queries the account balance using AxelarJS-SDK
func (s *SignerClient) GetBalance(address string, denom string) (sdk.Coin, error) {
	if denom == "" {
		denom = "uvx"
	}

	// Validate the address
	addr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		log.Error().Err(err).Str("address", address).Msg("Failed to parse address")
		return sdk.Coin{}, err
	}

	// Query balance directly using the client context
	queryClient := s.client.QueryClient()
	resp, err := queryClient.Bank.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: addr.String(),
		Denom:   denom,
	})
	if err != nil {
		log.Error().Err(err).
			Str("address", address).
			Str("denom", denom).
			Msg("Failed to query balance")
		return sdk.Coin{}, err
	}

	log.Debug().
		Str("address", address).
		Str("denom", denom).
		Str("balance", resp.Balance.String()).
		Msg("Balance queried successfully")

	return *resp.Balance, nil
}

// Broadcast sends a transaction and handles retries for sequence mismatch errors
func (s *SignerClient) Broadcast(msgs []sdk.Msg, memo string) (*sdk.TxResponse, error) {
	var retries int

	for {
		if retries >= s.maxRetries {
			return nil, fmt.Errorf("max retries exceeded")
		}

		// Create and sign the transaction
		txf := s.client.TxConfig.NewTxBuilder()
		txf.SetMsgs(msgs...)
		txf.SetMemo(memo)

		// Handle fee setting
		if s.fee == "auto" {
			// Simulate to estimate gas
			simRes, err := s.client.Simulate(txf)
			if err != nil {
				return nil, fmt.Errorf("failed to simulate transaction: %w", err)
			}
			// Add some buffer to estimated gas (e.g., 1.2x)
			gasLimit := uint64(float64(simRes.GasUsed) * 1.2)
			txf.SetGasLimit(gasLimit)
		}

		// Sign and broadcast the transaction
		txBytes, err := s.client.TxConfig.TxEncoder()(txf.GetTx())
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction: %w", err)
		}

		res, err := s.client.BroadcastTx(txBytes)
		if err != nil {
			if strings.Contains(err.Error(), "account sequence mismatch") {
				log.Info().
					Int("retry", retries+1).
					Dur("delay", s.retryDelay).
					Msg("Account sequence mismatch, retrying...")

				time.Sleep(s.retryDelay)
				retries++
				continue
			}
			return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
		}

		return res, nil
	}
}
