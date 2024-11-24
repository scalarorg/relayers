package evm_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/stretchr/testify/require"
)

var (
	globalConfig config.Config = config.Config{
		ConnnectionString: "postgres://postgres:postgres@localhost:5432/relayer?sslmode=disable",
	}
	evmConfig *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:    11155111,
		ID:         "ethereum-sepolia",
		Name:       "Ethereum sepolia",
		RPCUrl:     "wss://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
		Gateway:    "0xc9c5EC5975070a5CF225656e36C53e77eEa318b5",
		PrivateKey: "",
		Finality:   1,
		BlockTime:  time.Second * 12,
		LastBlock:  7121800,
	}
	evmClient *evm.EvmClient
)

func TestMain(m *testing.M) {
	var err error
	log.Info().Msgf("Creating evm client with config: %v", evmConfig)
	dbAdapter, err := db.NewDatabaseAdapter(&globalConfig)
	if err != nil {
		log.Error().Msgf("failed to create db adapter: %v", err)
	}
	evmClient, err = evm.NewEvmClient(&globalConfig, evmConfig, dbAdapter, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	os.Exit(m.Run())
}
func TestEvmClientListenContractCallEvent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &evmConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IAxelarGatewayContractCall)

	subContractCall, err := evmClient.Gateway.WatchContractCall(&watchOpts, sink, nil, nil)
	if err != nil {
		log.Error().Err(err).Msg("ContractCallEvent")
	}
	if subContractCall != nil {
		log.Info().Msg("Subscribed to ContractCallEvent successfully.")
		go func() {
			log.Info().Msg("Waiting for events...")
			for event := range sink {
				log.Info().Any("event", event).Msgf("ContractCall")
			}
		}()
		go func() {
			errChan := subContractCall.Err()
			if err := <-errChan; err != nil {
				log.Error().Err(err).Msg("Received error")
			}
		}()
	}
	select {}
}

func TestEvmClientListenContractCallApprovedEvent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &evmConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IAxelarGatewayContractCallApproved)

	subContractCallApproved, err := evmClient.Gateway.WatchContractCallApproved(&watchOpts, sink, nil, nil, nil)
	if err != nil {
		log.Error().Err(err).Msg("ContractCallApprovedEvent")
	}
	if subContractCallApproved != nil {
		log.Info().Msg("Subscribed to ContractCallApprovedEvent successfully.")
		go func() {
			log.Info().Msg("Waiting for events...")
			for event := range sink {
				log.Info().Any("event", event).Msgf("ContractCallApproved")
			}
		}()
		go func() {
			errChan := subContractCallApproved.Err()
			if err := <-errChan; err != nil {
				log.Error().Err(err).Msg("Received error")
			}
			subContractCallApproved.Unsubscribe()
		}()
	}
	select {}
}
func TestEvmClientListenEVMExecutedEvent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &evmConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IAxelarGatewayExecuted)

	subExecuted, err := evmClient.Gateway.WatchExecuted(&watchOpts, sink, nil)
	if err != nil {
		log.Error().Err(err).Msg("ExecutedEvent")
	}
	if subExecuted != nil {
		log.Info().Msg("Subscribed to ExecutedEvent successfully. Waiting for events...")
		go func() {
			for event := range sink {
				log.Info().Any("event", event).Msgf("Executed")
			}
		}()
		go func() {
			errChan := subExecuted.Err()
			if err := <-errChan; err != nil {
				log.Error().Err(err).Msg("Received error")
			}
		}()
	}
	select {}
}

func TestEvmSubscribe(t *testing.T) {
	fmt.Println("Test evm client")

	// Connect to Ethereum client
	client, err := ethclient.Dial(evmConfig.RPCUrl)
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to connect to the Ethereum client: %v", err)
	}

	// Get current block
	currentBlock, err := client.BlockNumber(context.Background())
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to get current block: %v", err)
	}
	fmt.Printf("Current block %d\n", currentBlock)

	// Create the event signature
	contractCallSig := []byte("ContractCall(address,string,string,bytes32,bytes)")

	// Create the filter query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(evmConfig.Gateway)},
		Topics: [][]common.Hash{{
			crypto.Keccak256Hash(contractCallSig),
		}},
	}

	// Subscribe to events
	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	require.NoError(t, err)
	if err != nil {
		fmt.Printf("failed to subscribe to logs: %v", err)
	}

	// Handle events in a separate goroutine
	go func() {
		for {
			select {
			case err := <-sub.Err():
				fmt.Printf("Received error: %v", err)
			case vLog := <-logs:
				fmt.Println("Log:", vLog)
			}
		}
	}()

	// Keep the program running
	select {}
}
