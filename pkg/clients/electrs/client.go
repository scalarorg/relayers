package electrs

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/go-electrum/electrum"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
)

type Client struct {
	globalConfig   *config.Config
	electrumConfig *Config
	electrs        *electrum.Client
	dbAdapter      *db.DatabaseAdapter
	eventBus       *events.EventBus
}

func NewElectrumClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) ([]*Client, error) {
	if globalConfig == nil || globalConfig.ConfigPath == "" {
		return nil, fmt.Errorf("config path is required")
	}
	electrumCfgPath := fmt.Sprintf("%s/electrum.json", globalConfig.ConfigPath)
	configs, err := config.ReadJsonArrayConfig[Config](electrumCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read electrum configs: %w", err)
	}
	clients := make([]*Client, len(configs))
	for i, config := range configs {
		//Update batch size to default if not provided
		if config.BatchSize == 0 {
			config.BatchSize = 1
		}
		client, err := NewElectrumClient(globalConfig, &config, dbAdapter, eventBus)
		if err != nil {
			return nil, fmt.Errorf("failed to create electrum client: %w", err)
		}
		clients[i] = client
	}
	return clients, nil
}
func NewElectrumClient(globalConfig *config.Config, config *Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("electrum rpc host is required")
	}
	if config.Port == 0 {
		return nil, fmt.Errorf("electrum rpc port is required")
	}
	if dbAdapter == nil {
		return nil, fmt.Errorf("dbAdapter is required")
	}
	rpcEndpoint := fmt.Sprintf("%s:%d", config.Host, config.Port)
	electrs, err := electrum.Connect(&electrum.Options{
		Dial: func() (net.Conn, error) {
			return net.DialTimeout("tcp", rpcEndpoint, time.Second)
		},
		MethodTimeout:   time.Second,
		PingInterval:    -1,
		SoftwareVersion: "scalar-relayer",
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to electrum server at %s", rpcEndpoint)
		return nil, err
	}
	return &Client{
		globalConfig:   globalConfig,
		electrumConfig: config,
		electrs:        electrs,
		dbAdapter:      dbAdapter,
		eventBus:       eventBus,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	params := []interface{}{}
	//Set batch size from config or default value
	params = append(params, c.electrumConfig.BatchSize)

	lastCheckpoint := c.getLastCheckpoint()
	log.Debug().Msgf("[ElectrumClient] [Start] Last checkpoint: %v", lastCheckpoint)
	if lastCheckpoint.EventKey != "" {
		params = append(params, lastCheckpoint.EventKey)
	} else if c.electrumConfig.LastVaultTx != "" {
		params = append(params, c.electrumConfig.LastVaultTx)
	}

	log.Debug().Msgf("[ElectrumClient] [Start] Subscribing to vault transactions with params: %v", params)
	c.electrs.VaultTransactionSubscribe(ctx, c.vaultTxMessageHandler, params...)
	return nil
}

// Handle vault messages
// Todo: Add some logging, metric and error handling if needed
func (c *Client) vaultTxMessageHandler(vaultTxs []types.VaultTransaction, err error) error {
	if err != nil {
		log.Warn().Msgf("[ElectrumClient] [vaultTxMessageHandler] Failed to receive vault transaction: %v", err)
		return fmt.Errorf("failed to receive vault transaction: %w", err)
	}
	if len(vaultTxs) == 0 {
		log.Debug().Msg("[ElectrumClient] [vaultTxMessageHandler] No vault transactions received")
		return nil
	}
	c.PreProcessMessages(vaultTxs)
	//1. parse vault transactions to relay data
	relayDatas, err := c.CreateRelayDatas(vaultTxs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert vault transaction to relay data")
		return fmt.Errorf("failed to convert vault transaction to relay data: %w", err)
	} else {
		payloadHashes := make([]string, len(relayDatas))
		for i, relayData := range relayDatas {
			payloadHashes[i] = relayData.CallContract.PayloadHash
		}
		log.Debug().Msgf("Successfully stored %d vault transactions to relay data, payload hashes: %v", len(relayDatas), payloadHashes)
	}
	//2. update last checkpoint
	lastCheckpoint := c.getLastCheckpoint()
	for _, tx := range vaultTxs {
		if uint64(tx.Height) > lastCheckpoint.BlockNumber ||
			(uint64(tx.Height) == lastCheckpoint.BlockNumber && uint(tx.TxPosition) > lastCheckpoint.LogIndex) {
			lastCheckpoint.BlockNumber = uint64(tx.Height)
			lastCheckpoint.TxHash = tx.TxHash
			lastCheckpoint.LogIndex = uint(tx.TxPosition)
			lastCheckpoint.EventKey = tx.Key
		}
	}
	//3. store relay data to the db, update last checkpoint
	err = c.dbAdapter.CreateRelayDatas(relayDatas, lastCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store relay data to the db")
		return fmt.Errorf("failed to store relay data to the db: %w", err)
	}
	//4. Send to the event bus with destination chain is scalar for confirmation
	confirmTxs := events.ConfirmTxsRequest{
		ChainName: c.electrumConfig.SourceChain,
		TxHashs:   make(map[string]string),
	}
	for _, item := range relayDatas {
		confirmTxs.TxHashs[item.CallContract.TxHash] = item.To
	}
	if c.eventBus != nil {
		c.eventBus.BroadcastEvent(&events.EventEnvelope{
			EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
			DestinationChain: events.SCALAR_NETWORK_NAME,
			Data:             confirmTxs,
		})
	} else {
		log.Warn().Msg("[ElectrumClient] [vaultTxMessageHandler] event bus is undefined")
	}
	return nil
}

// Get lastcheck point from db, return default value if not found
func (c *Client) getLastCheckpoint() *models.EventCheckPoint {
	sourceChain := c.electrumConfig.SourceChain
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(sourceChain, events.EVENT_ELECTRS_VAULT_TRANSACTION)
	if err != nil {
		log.Warn().Str("chainId", sourceChain).
			Str("eventName", events.EVENT_ELECTRS_VAULT_TRANSACTION).
			Msg("[ElectrumClient] [getLastCheckpoint] using default value")
	}
	return lastCheckpoint
}

// Todo: Log and validate incomming message
func (c *Client) PreProcessMessages(vaultTxs []types.VaultTransaction) error {
	log.Info().Msgf("Received %d vault transactions", len(vaultTxs))
	for _, vaultTx := range vaultTxs {
		log.Debug().Msgf("Received vaultTx with key=>%v; stakerAddress=>%v; stakerPubkey=>%v", vaultTx.Key, vaultTx.StakerAddress, vaultTx.StakerPubkey)
	}
	return nil
}
