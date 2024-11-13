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
	"github.com/scalarorg/relayers/pkg/clients/common"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/db/models"
	"github.com/scalarorg/relayers/pkg/events"
)

type Client struct {
	config       *Config
	commonConfig *common.CommonConfig
	electrs      *electrum.Client
	dbAdapter    *db.DatabaseAdapter
	eventBus     *events.EventBus
}

func NewElectrumClients(configPath string, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) ([]*Client, error) {
	electrumCfgPath := fmt.Sprintf("%s/electrum.json", configPath)
	configs, err := config.ReadJsonArrayConfig[Config](electrumCfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read electrum configs: %w", err)
	}
	clients := make([]*Client, len(configs))
	for i, config := range configs {
		client, err := NewElectrumClient(&config, dbAdapter, eventBus)
		if err != nil {
			return nil, fmt.Errorf("failed to create electrum client: %w", err)
		}
		clients[i] = client
	}
	return clients, nil
}
func NewElectrumClient(config *Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("Electrum rpc host is required")
	}
	if config.Port == 0 {
		return nil, fmt.Errorf("Electrum rpc port is required")
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
		SoftwareVersion: "testclient",
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to electrum server at %s", rpcEndpoint)
		return nil, err
	}
	return &Client{
		config:    config,
		electrs:   electrs,
		dbAdapter: dbAdapter,
		eventBus:  eventBus,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	params := []interface{}{}
	if c.config.BatchSize > 0 {
		params = append(params, c.config.BatchSize)
	}
	if c.config.LastVaultTx != "" {
		if c.config.BatchSize == 0 {
			//"BatchSize is required when LastVaultTx is provided"
			//Set default batch size to 1
			params = append(params, 1)
		}
		params = append(params, c.config.LastVaultTx)
	}

	log.Debug().Msgf("Subscribing to vault transactions with params: %v", params)
	c.electrs.VaultTransactionSubscribe(ctx, c.vaultTxMessageHandler, params...)
	return nil
}

// Handle vault messages
// Todo: Add some logging, metric and error handling if needed
func (c *Client) vaultTxMessageHandler(vaultTxs []types.VaultTransaction, err error) {
	if err != nil {
		log.Error().Err(err).Msg("Failed to receive vault transaction")
	}
	c.PreProcessMessages(vaultTxs)
	//1. Store to the db
	relayDatas, err := c.CreateRelayDatas(vaultTxs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert vault transaction to relay data")
		return
	}
	err = c.dbAdapter.CreateRelayDatas(relayDatas)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store relay data to the db")
	}
	//2. Send to the event bus with destination chain is scalar for confirmation
	grouped := c.GroupVaultTxsByDestinationChain(relayDatas)
	c.eventBus.BroadcastEvent(&events.EventEnvelope{
		EventType:        events.EVENT_ELECTRS_VAULT_TRANSACTION,
		DestinationChain: scalar.SCALAR_NETWORK_NAME,
		Data:             grouped,
	})
}

// Todo: Log and validate incomming message
func (c *Client) PreProcessMessages(vaultTxs []types.VaultTransaction) error {
	log.Info().Msgf("Received %d vault transactions", len(vaultTxs))
	for _, vaultTx := range vaultTxs {
		log.Debug().Msgf("Received vaultTx with key=>%v; stakerAddress=>%v; stakerPubkey=>%v", vaultTx.Key, vaultTx.StakerAddress, vaultTx.StakerPubkey)
	}
	return nil
}

// GroupVaultTxsByDestinationChain groups vault transactions by their destination chain
func (c *Client) GroupVaultTxsByDestinationChain(relayDatas []models.RelayData) map[string][]string {
	grouped := make(map[string][]string)
	for _, item := range relayDatas {
		grouped[item.To] = append(grouped[item.To], item.CallContract.TxHash)
	}
	return grouped
}
