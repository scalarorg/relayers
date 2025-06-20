package electrs

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scalarorg/data-models/scalarnet"
	"github.com/scalarorg/go-electrum/electrum"
	"github.com/scalarorg/go-electrum/electrum/types"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/clients/scalar"
	"github.com/scalarorg/relayers/pkg/db"
	"github.com/scalarorg/relayers/pkg/events"
	"github.com/scalarorg/scalar-core/x/covenant/exported"
)

const (
	EVENT_ELECTRS_VAULT_BLOCK = "vault_block"
)

type Client struct {
	globalConfig    *config.Config
	electrumConfig  *Config
	Electrs         *electrum.Client
	dbAdapter       *db.DatabaseAdapter
	eventBus        *events.EventBus
	scalarClient    *scalar.Client
	custodialGroups []*exported.CustodianGroup
	currentHeight   int
}

func NewElectrumClients(globalConfig *config.Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) ([]*Client, error) {
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
		client, err := NewElectrumClient(globalConfig, &config, dbAdapter, eventBus, scalarClient)
		if err != nil {
			return nil, fmt.Errorf("failed to create electrum client: %w", err)
		}
		clients[i] = client
	}
	return clients, nil
}
func NewElectrumClient(globalConfig *config.Config, config *Config, dbAdapter *db.DatabaseAdapter, eventBus *events.EventBus, scalarClient *scalar.Client) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("electrum rpc host is required")
	}
	if config.Port == 0 {
		return nil, fmt.Errorf("electrum rpc port is required")
	}
	if dbAdapter == nil {
		return nil, fmt.Errorf("dbAdapter is required")
	}
	if config.Confirmations == 0 {
		config.Confirmations = 1
	}

	// Set default timeouts if not provided
	if config.DialTimeout == 0 {
		config.DialTimeout = Duration(10 * time.Second)
	}
	if config.MethodTimeout <= 0 {
		config.MethodTimeout = Duration(60 * time.Second)
	}
	if config.PingInterval <= 0 {
		config.PingInterval = Duration(30 * time.Second)
	}

	log.Debug().Dur("dialTimeout", config.DialTimeout.ToDuration()).
		Dur("methodTimeout", config.MethodTimeout.ToDuration()).
		Dur("pingInterval", config.PingInterval.ToDuration()).
		Msg("[ElectrumClient] [NewElectrumClient] Using timeout configuration")

	rpcEndpoint := fmt.Sprintf("%s:%d", config.Host, config.Port)
	electrs, err := electrum.Connect(&electrum.Options{
		Dial: func() (net.Conn, error) {
			return net.DialTimeout("tcp", rpcEndpoint, config.DialTimeout.ToDuration())
		},
		MethodTimeout:   config.MethodTimeout.ToDuration(),
		PingInterval:    config.PingInterval.ToDuration(),
		SoftwareVersion: "scalar-relayer",
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to connect to electrum server at %s", rpcEndpoint)
		return nil, err
	}
	return &Client{
		globalConfig:   globalConfig,
		electrumConfig: config,
		Electrs:        electrs,
		dbAdapter:      dbAdapter,
		eventBus:       eventBus,
		scalarClient:   scalarClient,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	params := []interface{}{}
	//Set batch size from config or default value
	params = append(params, 1)
	if c.scalarClient != nil {
		var err error
		c.custodialGroups, err = c.scalarClient.GetCovenantGroups(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ElectrumClient] failed to get custodian group")
		}
	}
	lastCheckpoint := c.getLastCheckpoint()
	log.Debug().Msgf("[ElectrumClient] [Start] Last checkpoint: %v", lastCheckpoint)
	if lastCheckpoint.EventKey != "" {
		params = append(params, lastCheckpoint.EventKey)
	} else if c.electrumConfig.LastVaultTx != "" {
		params = append(params, c.electrumConfig.LastVaultTx)
	}
	log.Debug().Msg("[ElectrumClient] [Start] Subscribing to new block event for request to confirm if vault transaction is get enought confirmation")
	c.Electrs.BlockchainHeaderSubscribe(ctx, c.BlockchainHeaderHandler)
	//log.Debug().Msgf("[ElectrumClient] [Start] Subscribing to vault transactions with params: %v", params)
	//c.Electrs.VaultTransactionSubscribe(ctx, c.VaultTxMessageHandler, params...)
	c.Electrs.SubscribeEvent(ctx, types.VaultBlocksSubscribe, c.HandleValueBlockWithVaultTxs, params...)
	return nil
}

func (c *Client) GetSymbol(chainId string, tokenAddress string) (string, error) {
	if c.scalarClient == nil {
		return "", fmt.Errorf("scalar client is not initialized")
	}
	return c.scalarClient.GetSymbol(context.Background(), chainId, tokenAddress)
}

// Get lastcheck point from db, return default value if not found
func (c *Client) getLastCheckpoint() *scalarnet.EventCheckPoint {
	sourceChain := c.electrumConfig.SourceChain
	lastCheckpoint, err := c.dbAdapter.GetLastEventCheckPoint(sourceChain, events.EVENT_ELECTRS_VAULT_TRANSACTION)
	if err != nil {
		log.Warn().Str("chainId", sourceChain).
			Str("eventName", events.EVENT_ELECTRS_VAULT_TRANSACTION).
			Msg("[ElectrumClient] [getLastCheckpoint] using default value")
	}
	return lastCheckpoint
}
