package evm

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog/log"
	chains "github.com/scalarorg/data-models/chains"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/relayers/config"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"golang.org/x/crypto/sha3"
)

type EvmClient struct {
	globalConfig      *config.Config
	EvmConfig         *EvmNetworkConfig
	Client            *ethclient.Client
	ScalarClient      *scalar.Client
	ChainName         string
	GatewayAddress    common.Address
	Gateway           *contracts.IScalarGateway
	auth              *bind.TransactOpts
	dbAdapter         *db.DatabaseAdapter
	eventBus          *events.EventBus
	MissingLogs       MissingLogs
	ChnlReceivedBlock chan uint64
	// pendingTxs     pending.PendingTxs //Transactions sent to Gateway for approval, waiting for event from EVM chain.
	retryInterval time.Duration
}

// This function is used to adjust the rpc url to the ws prefix
// format: ws:// -> http://
// format: wss:// -> https://
// Todo: Improve this implementation

func NewEvmClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) ([]*EvmClient, error) {
	if globalConfig == nil || globalConfig.ConfigPath == "" {
		return nil, fmt.Errorf("config path is not set")
	}
	evmCfgPath := fmt.Sprintf("%s/evm.json", globalConfig.ConfigPath)
	configs, err := config.ReadJsonArrayConfig[EvmNetworkConfig](evmCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read electrum configs: %w", err)
	}

	evmClients := make([]*EvmClient, 0, len(configs))
	for _, evmConfig := range configs {
		//Inject evm private keys
		preparePrivateKey(&evmConfig)
		//Set default value for block time if is not set
		if evmConfig.BlockTime == 0 {
			evmConfig.BlockTime = 12 * time.Second
		} else {
			evmConfig.BlockTime = evmConfig.BlockTime * time.Millisecond
		}
		//Set default gaslimit to 300000
		if evmConfig.GasLimit == 0 {
			evmConfig.GasLimit = 3000000
		}
		if evmConfig.RecoverRange == 0 {
			evmConfig.RecoverRange = 1000000
		}
		client, err := NewEvmClient(globalConfig, &evmConfig, dbAdapter, eventBus, scalarClient)
		if err != nil {
			log.Warn().Msgf("Failed to create evm client for %s: %v", evmConfig.GetName(), err)
			continue
		}
		globalConfig.AddChainConfig(config.IChainConfig(&evmConfig))
		evmClients = append(evmClients, client)
	}

	return evmClients, nil
}

func NewEvmClient(globalConfig *config.Config, evmConfig *EvmNetworkConfig, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) (*EvmClient, error) {
	// Setup
	ctx := context.Background()
	log.Info().Any("evmConfig", evmConfig).Msgf("[EvmClient] [NewEvmClient] connecting to EVM network")
	// Connect to a test network
	rpc, err := rpc.DialContext(ctx, evmConfig.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM network %s: %w", evmConfig.Name, err)
	}
	client := ethclient.NewClient(rpc)
	gateway, gatewayAddress, err := CreateGateway(evmConfig.Name, evmConfig.Gateway, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway for network %s: %w", evmConfig.Name, err)
	}
	auth, err := CreateTransactOpts(evmConfig)
	if err != nil {
		//Not fatal, we can still use the gateway without auth
		//auth is only used for sending transaction
		panic(fmt.Errorf("[EvmClient] [NewEvmClient] failed to create auth for network %s: %w", evmConfig.Name, err))
	}
	evmClient := &EvmClient{
		globalConfig:   globalConfig,
		EvmConfig:      evmConfig,
		Client:         client,
		ScalarClient:   scalarClient,
		GatewayAddress: *gatewayAddress,
		Gateway:        gateway,
		auth:           auth,
		dbAdapter:      dbAdapter,
		eventBus:       eventBus,
		MissingLogs: MissingLogs{
			chainId:   evmConfig.GetId(),
			RedeemTxs: make(map[string][]string),
		},
		ChnlReceivedBlock: make(chan uint64),
		retryInterval:     RETRY_INTERVAL,
	}

	return evmClient, nil
}
func CreateGateway(networName string, gwAddr string, client *ethclient.Client) (*contracts.IScalarGateway, *common.Address, error) {
	if gwAddr == "" {
		return nil, nil, fmt.Errorf("gateway address is not set for network %s", networName)
	}
	gatewayAddress := common.HexToAddress(gwAddr)
	gateway, err := contracts.NewIScalarGateway(gatewayAddress, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize gateway contract for network %s: %w", networName, err)
	}
	return gateway, &gatewayAddress, nil
}
func CreateTransactOpts(evmConfig *EvmNetworkConfig) (*bind.TransactOpts, error) {
	if evmConfig.PrivateKey == "" {
		return nil, fmt.Errorf("private key is not set for network %s", evmConfig.Name)
	}
	privateKey, err := crypto.HexToECDSA(evmConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key for network %s: %w", evmConfig.Name, err)
	}
	chainID := big.NewInt(int64(evmConfig.ChainID))
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth for network %s: %w", evmConfig.Name, err)
	}
	auth.GasLimit = evmConfig.GasLimit
	return auth, nil
}

func (ec *EvmClient) CreateCallOpts() (*bind.CallOpts, error) {
	callOpt := &bind.CallOpts{
		From:    ec.auth.From,
		Context: context.Background(),
	}
	return callOpt, nil
}

func (ec *EvmClient) gatewayExecute(input []byte) (*types.Transaction, error) {
	//ec.auth.NoSend = false
	log.Info().Bool("NoSend", ec.auth.NoSend).Msgf("[EvmClient] [gatewayExecute] sending transaction")
	signedTx, err := ec.Gateway.Execute(ec.auth, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	// err = ec.Client.SendTransaction(context.Background(), signedTx)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to send transaction: %w", err)
	// }
	return signedTx, nil
}
func preparePrivateKey(evmCfg *EvmNetworkConfig) error {
	evm_key, err := base64.StdEncoding.DecodeString(config.GlobalConfig.EvmKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to decode evm_key: %v", err))
	}
	nonce, err := base64.StdEncoding.DecodeString(config.GlobalConfig.Nonce)
	if err != nil {
		panic(fmt.Sprintf("Failed to decode nonce: %v", err))
	}
	hash := sha3.Sum256([]byte(config.APP_NAME))
	decrypted, err := AESGCMDecrypt(hash[:], nonce, evm_key)
	if err != nil {
		panic(fmt.Sprintf("Failed to decrypt evm_key: %v", err))
	}
	evmCfg.PrivateKey = string(decrypted)
	return nil
}
func (c *EvmClient) SetAuth(auth *bind.TransactOpts) {
	c.auth = auth
}

func (c *EvmClient) Start(ctx context.Context) error {
	//c.startFetchBlock()
	//Subscribe to the event bus
	c.subscribeEventBus()
	//c.ConnectWithRetry(ctx)
	return fmt.Errorf("context cancelled")
}

// Get missing events from the last checkpoint block number to the current block number
// In query we filter out the event with index equal to the last checkpoint log index
func GetMissingEvents[T ValidEvmEvent](c *EvmClient, eventName string, lastCheckpoint *scalarnet.EventCheckPoint, fnCreateEventData func(types.Log) T) ([]*parser.EvmEvent[T], error) {
	event, ok := scalarGatewayAbi.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event %s not found", eventName)
	}

	// Set up a query for logs
	// Todo: config default last checkpoint block number to make sure we don't recover too old events
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(lastCheckpoint.BlockNumber)),
		Addresses: []common.Address{c.GatewayAddress},
		Topics:    [][]common.Hash{{event.ID}}, // Filter by event signature
	}
	// // Fetch the logs
	logs, err := c.Client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	log.Debug().Int("logsCount", len(logs)).Any("logs", logs).Msg("[EvmClient] [GetMissingEvents] fetched logs")
	result := []*parser.EvmEvent[T]{}
	// Parse the logs
	for _, receiptLog := range logs {
		if receiptLog.BlockNumber < lastCheckpoint.BlockNumber ||
			(receiptLog.BlockNumber == lastCheckpoint.BlockNumber && receiptLog.Index <= lastCheckpoint.LogIndex) {
			log.Info().Uint64("receiptLogBlockNumber", receiptLog.BlockNumber).
				Uint("receiptLogIndex", receiptLog.Index).
				Msg("[EvmClient] [GetMissingEvents] skip log")
			continue
		}
		var eventData = fnCreateEventData(receiptLog)
		err := parser.ParseEventData(&receiptLog, eventName, eventData)
		if err == nil {
			evmEvent := parser.CreateEvmEventFromArgs[T](eventData, &receiptLog)
			result = append(result, evmEvent)
			log.Info().
				Str("eventName", eventName).
				Uint64("blockNumber", evmEvent.BlockNumber).
				Str("txHash", evmEvent.Hash).
				Uint("txIndex", evmEvent.TxIndex).
				Uint("logIndex", evmEvent.LogIndex).
				Msg("[EvmClient] [GetMissingEvents] parsing successfully.")
		} else {
			log.Error().Err(err).Msg("[EvmClient] [GetMissingEvents] failed to unpack log data")
		}
	}
	log.Info().Int("eventCount", len(logs)).
		Str("eventName", eventName).
		Any("query", query).
		Any("lastCheckpoint", lastCheckpoint).
		Any("missingEventsCount", len(result)).
		Msg("[EvmClient] [GetMissingEvents]")
	return result, nil
}

//	func (c *EvmClient) startFetchBlock() {
//		go func() {
//			for blockNumber := range c.ChnlReceivedBlock {
//				if c.dbAdapter == nil {
//					log.Error().Msgf("[EvmClient] [startFetchBlock] db adapter is not set")
//					continue
//				}
//				log.Info().Str("ChainId", c.EvmConfig.ID).Uint64("BlockNumber", blockNumber).Msg("[EvmClient] Fetch block by number")
//				//block, err := c.Client.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
//				blockHeader, err := c.Client.HeaderByNumber(context.Background(), big.NewInt(int64(blockNumber)))
//				if err != nil {
//					log.Error().Err(err).Msgf("[EvmClient] [startFetchBlock] failed to fetch block %d", blockNumber)
//				} else if blockHeader == nil {
//					log.Error().Msgf("[EvmClient] [startFetchBlock] block %d not found", blockNumber)
//				} else {
//					log.Info().Uint64("blockNumber", blockHeader.Number.Uint64()).
//						Uint64("blockTime", blockHeader.Time).
//						Msgf("[EvmClient] [startFetchBlock] block found")
//					blockNumber := blockHeader.Number.Uint64()
//					blockHeader := &chains.BlockHeader{
//						Chain:       c.EvmConfig.GetId(),
//						BlockNumber: blockNumber,
//						BlockHash:   hex.EncodeToString(blockHeader.Hash().Bytes()),
//						BlockTime:   blockHeader.Time,
//					}
//					err = c.dbAdapter.CreateBlockHeader(blockHeader)
//					if err != nil {
//						log.Error().Err(err).Msgf("[EvmClient] [startFetchBlock] failed to save block header %d", blockNumber)
//					}
//				}
//			}
//		}()
//	}
func (c *EvmClient) fetchBlockHeader(blockNumber uint64) (*chains.BlockHeader, error) {
	blockHeader, err := c.dbAdapter.FindBlockHeader(c.EvmConfig.GetId(), blockNumber)
	if err == nil && blockHeader != nil {
		log.Info().Any("blockHeader", blockHeader).Msgf("[EvmClient] [startFetchBlock] block header already exists")
		return blockHeader, nil
	}
	c.ChnlReceivedBlock <- blockNumber
	return nil, nil
}
func (c *EvmClient) subscribeEventBus() {
	if c.eventBus != nil {
		log.Debug().Msgf("[EvmClient] [Start] subscribe to the event bus %s", c.EvmConfig.GetId())
		receiver := c.eventBus.Subscribe(c.EvmConfig.GetId())
		go func() {
			for event := range receiver {
				err := c.handleEventBusMessage(event)
				if err != nil {
					log.Error().Err(err).Msgf("[EvmClient] [EventBusHandler]")
				}
			}
		}()
	} else {
		log.Warn().Msgf("[EvmClient] [subscribeEventBus] event bus is not set")
	}
}
