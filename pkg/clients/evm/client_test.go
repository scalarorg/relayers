package evm_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/evm"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	CHAIN_ID_SEPOLIA = "evm|11155111"
	CHAIN_ID_BNB     = "evm|97"
	TOKEN_SYMBOL     = "pBtc"
)

var (
	globalConfig config.Config = config.Config{
		ConnnectionString: "postgres://postgres:postgres@localhost:5432/relayer?sslmode=disable",
	}
	sepoliaClient  *ethclient.Client
	bnbClient      *ethclient.Client
	evmPrivateKey  string
	evmUserPrivKey string
	evmUserAddress string
	sepoliaConfig  *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:    11155111,
		ID:         CHAIN_ID_SEPOLIA,
		Name:       "Ethereum sepolia",
		RPCUrl:     "wss://eth-sepolia.g.alchemy.com/v2/nNbspp-yjKP9GtAcdKi8xcLnBTptR2Zx",
		Gateway:    "0x842C080EE1399addb76830CFe21D41e47aaaf57e",
		PrivateKey: "",
		Finality:   1,
		BlockTime:  time.Second * 12,
		LastBlock:  7121800,
		GasLimit:   300000,
	}
	bnbConfig *evm.EvmNetworkConfig = &evm.EvmNetworkConfig{
		ChainID:    97,
		ID:         CHAIN_ID_BNB,
		Name:       "Ethereum bnb",
		RPCUrl:     "wss://bnb-testnet.g.alchemy.com/v2/DpCscOiv_evEPscGYARI3cOVeJ59CRo8",
		Gateway:    "0x8cFc0173f7D1701bf5010B15B9762264d88c4235",
		PrivateKey: "",
		Finality:   1,
		BlockTime:  time.Second * 12,
		LastBlock:  47254017,
		GasLimit:   300000,
	}
	evmClient *evm.EvmClient
)

func TestMain(m *testing.M) {
	// Load .env file
	err := godotenv.Load("../../../.env.test")
	if err != nil {
		log.Error().Err(err).Msg("Error loading .env.test file: %v")
	}
	evmPrivateKey = os.Getenv("EVM_PRIVATE_KEY")
	evmUserPrivKey = os.Getenv("EVM_USER_PRIVATE_KEY")
	evmUserAddress = os.Getenv("EVM_USER_ADDRESS")
	bnbConfig.RPCUrl = os.Getenv("URL_BNB_WSS")
	sepoliaConfig.PrivateKey = evmPrivateKey
	bnbConfig.PrivateKey = evmPrivateKey
	sepoliaClient, _ = createEVMClient("RPC_SEPOLIA")
	bnbClient, _ = createEVMClient("RPC_BNB")

	log.Info().Msgf("Creating evm client with config: %v", sepoliaConfig)
	dbAdapter, err := db.NewDatabaseAdapter(&globalConfig)
	if err != nil {
		log.Error().Msgf("failed to create db adapter: %v", err)
	}
	evmClient, err = evm.NewEvmClient(&globalConfig, sepoliaConfig, dbAdapter, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	os.Exit(m.Run())
}
func createEVMClient(key string) (*ethclient.Client, error) {
	rpcEndpoint := os.Getenv(key)
	rpcSepolia, err := rpc.DialContext(context.Background(), rpcEndpoint)
	if err != nil {
		fmt.Printf("failed to connect to sepolia with rpc %s: %v", rpcEndpoint, err)
		return nil, err
	}
	return ethclient.NewClient(rpcSepolia), nil
}
func TestEvmClientListenContractCallEvent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &sepoliaConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IScalarGatewayContractCall)

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
	watchOpts := bind.WatchOpts{Start: &sepoliaConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IScalarGatewayContractCallApproved)

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
	watchOpts := bind.WatchOpts{Start: &sepoliaConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IScalarGatewayExecuted)

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
func TestRecoverEvent(t *testing.T) {
	fnCreateEventData := func(log types.Log) *contracts.IScalarGatewayContractCall {
		return &contracts.IScalarGatewayContractCall{
			Raw: log,
		}
	}
	err := evm.RecoverEvent[*contracts.IScalarGatewayContractCall](evmClient, context.Background(), events.EVENT_EVM_CONTRACT_CALL, fnCreateEventData)
	require.NoError(t, err)
}
func TestEvmSubscribe(t *testing.T) {
	fmt.Println("Test evm client")

	// Connect to Ethereum client
	client, err := ethclient.Dial(sepoliaConfig.RPCUrl)
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
		Addresses: []common.Address{common.HexToAddress(sepoliaConfig.Gateway)},
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
func TestRecoverEventTokenSent(t *testing.T) {
	bnbClient, err := evm.NewEvmClient(&globalConfig, bnbConfig, nil, nil, nil)
	require.NoError(t, err)
	//Get current block number
	blockNumber, err := bnbClient.Client.BlockNumber(context.Background())
	require.NoError(t, err)
	lastCheckpoint := scalarnet.EventCheckPoint{
		ChainName:   bnbConfig.ID,
		EventName:   events.EVENT_EVM_TOKEN_SENT,
		BlockNumber: blockNumber - 10000,
		TxHash:      "",
		LogIndex:    0,
		EventKey:    "",
	}
	missingEvents, err := evm.GetMissingEvents[*contracts.IScalarGatewayTokenSent](bnbClient, events.EVENT_EVM_TOKEN_SENT,
		&lastCheckpoint, func(log types.Log) *contracts.IScalarGatewayTokenSent {
			return &contracts.IScalarGatewayTokenSent{
				Raw: log,
			}
		})
	require.NoError(t, err)
	fmt.Printf("missingEvents %v\n", missingEvents)
}
func TestRecoverEventContractCallWithToken(t *testing.T) {
	bnbClient, err := evm.NewEvmClient(&globalConfig, bnbConfig, nil, nil, nil)
	require.NoError(t, err)
	//Get current block number
	blockNumber, err := bnbClient.Client.BlockNumber(context.Background())
	fmt.Printf("blockNumber %v\n", blockNumber)
	require.NoError(t, err)
	lastCheckpoint := scalarnet.EventCheckPoint{
		ChainName:   bnbConfig.ID,
		EventName:   events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		BlockNumber: bnbConfig.LastBlock,
		TxHash:      "",
		LogIndex:    0,
		EventKey:    "",
	}
	missingEvents, err := evm.GetMissingEvents[*contracts.IScalarGatewayContractCallWithToken](bnbClient, events.EVENT_EVM_CONTRACT_CALL_WITH_TOKEN,
		&lastCheckpoint, func(log types.Log) *contracts.IScalarGatewayContractCallWithToken {
			return &contracts.IScalarGatewayContractCallWithToken{
				Raw: log,
			}
		})
	require.NoError(t, err)
	fmt.Printf("%d missing events found\n", len(missingEvents))

	txHash := "0xc7d4fac102169c129a4f04ccd4e3fa17fcd962f137e4928fb2462c52da039899"
	tx, isPending, err := bnbClient.Client.TransactionByHash(context.Background(), common.HexToHash(txHash))
	fmt.Printf("tx %v\n", tx)
	fmt.Printf("isPending %v\n", isPending)
	fmt.Printf("err %v\n", err)
	txHash = "1c9623e21b55e9c4767a12b27f9f68578c167284651efb2b87a51ce438e9fa53"
	tx, isPending, err = bnbClient.Client.TransactionByHash(context.Background(), common.HexToHash(txHash))
	require.NoError(t, err)
	fmt.Printf("tx %v\n", tx)
	fmt.Printf("isPending %v\n", isPending)
	fmt.Printf("err %v\n", err)

	log.Info().Str("txHash", txHash).Any("tx", tx).Msgf("ContractCallWithToken")
	for _, event := range missingEvents {
		receipt, err := bnbClient.Client.TransactionReceipt(context.Background(), common.HexToHash(event.Hash))
		require.NoError(t, err)
		log.Info().Str("txHash", event.Hash).Any("receipt", receipt).Msgf("ContractCallWithToken")
	}
}

func TestEvmClientWatchTokenSent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &sepoliaConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IScalarGatewayTokenSent)
	bnbClient, err := evm.NewEvmClient(&globalConfig, bnbConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	subscription, err := bnbClient.Gateway.WatchTokenSent(&watchOpts, sink, nil)
	require.NoError(t, err)
	defer subscription.Unsubscribe()
	log.Info().Msgf("[EvmClient] [watchEVMTokenSent] success. Listening to TokenSent")

	for {
		select {
		case err := <-subscription.Err():
			log.Error().Msgf("[EvmClient] [watchEVMTokenSent] error: %v", err)
		case event := <-sink:
			log.Info().Any("event", event).Msgf("EvmClient] [watchEVMTokenSent]")
		}
	}
}
func TestSendTokenFromSepoliaToBnb(t *testing.T) {
	fmt.Println("Test SendToken From Sepolia to BnB")
	fmt.Printf("DestChain %s, TokenSymbol %s, UserAddress %s", CHAIN_ID_BNB, TOKEN_SYMBOL, evmUserAddress)
	sepoliaGwAddr := "0x842C080EE1399addb76830CFe21D41e47aaaf57e"
	amount := big.NewInt(10000)

	sepoliaGateway, _, err := evm.CreateGateway("Sepolia", sepoliaGwAddr, sepoliaClient)
	assert.NoError(t, err)
	sepoliaConfig.PrivateKey = evmUserPrivKey
	fmt.Printf("SepoliaConfig %v\n", sepoliaConfig)
	transOpts, err := evm.CreateEvmAuth(sepoliaConfig)
	assert.NoError(t, err)
	//Need to incrate allowance if need
	// tokenAddress := "0x6e3B806C5F6413e0a0670666301ccB6b10628A52"
	// proxy, err := createErc20ProxyContract(tokenAddress, sepoliaClient)
	// assert.NoError(t, err)
	// allowance, err := proxy.Allowance(callOpts, common.HexToAddress(evmUserAddress), common.HexToAddress(sepoliaGwAddr))
	// assert.NoError(t, err)
	// if allowance.Int64() < amount.Int64() {
	// 	proxy.IncreaseAllowance(transOpts, common.HexToAddress(sepoliaGwAddr), amount)
	// }
	time.Sleep(15 * time.Second)
	tx, err := sepoliaGateway.SendToken(transOpts, CHAIN_ID_BNB, evmUserAddress, TOKEN_SYMBOL, amount)
	assert.NoError(t, err)
	fmt.Printf("SendToken tx %v\n", tx)
}
func TestReconnectWithWatchTokenSent(t *testing.T) {
	watchOpts := bind.WatchOpts{Start: &sepoliaConfig.LastBlock, Context: context.Background()}
	sink := make(chan *contracts.IScalarGatewayTokenSent)
	bnbClient, err := evm.NewEvmClient(&globalConfig, bnbConfig, nil, nil, nil)
	if err != nil {
		log.Error().Msgf("failed to create evm client: %v", err)
	}
	subscription, err := bnbClient.Gateway.WatchTokenSent(&watchOpts, sink, nil)
	require.NoError(t, err)
	defer subscription.Unsubscribe()
	log.Info().Msgf("[EvmClient] [watchEVMTokenSent] success. Listening to TokenSent")
	done := false
	for !done {
		select {
		case err := <-subscription.Err():
			log.Error().Err(err).Msg("[EvmClient] [watchEVMTokenSent] error with subscription, perform reconnect")
			subscription, err = bnbClient.Gateway.WatchTokenSent(&watchOpts, sink, nil)
			require.NoError(t, err)
		case <-watchOpts.Context.Done():
			log.Info().Msgf("[EvmClient] [watchEVMTokenSent] context done")
			done = true
		case event := <-sink:
			log.Info().Any("event", event).Msgf("EvmClient] [watchEVMTokenSent]")
		}
	}
}
func createErc20ProxyContract(proxyAddress string, client *ethclient.Client) (*contracts.IScalarERC20CrossChain, error) {
	proxy, err := contracts.NewIScalarERC20CrossChain(common.HexToAddress(proxyAddress), client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize proxy contract from address: %s", proxyAddress)
	}
	return proxy, nil
}
