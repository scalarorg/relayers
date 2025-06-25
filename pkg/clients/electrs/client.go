package electrs

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
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
	currentHeight   int64
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
	// Set default values first
	if config.Confirmations == 0 {
		config.Confirmations = 1
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1
	}
	if config.DialTimeout.ToDuration().Seconds() <= 10 {
		config.DialTimeout = Duration(10 * time.Second)
	}
	if config.MethodTimeout.ToDuration().Seconds() <= 60 {
		config.MethodTimeout = Duration(60 * time.Second)
	}
	if config.PingInterval <= 0 {
		config.PingInterval = Duration(30 * time.Second)
	}
	if config.MaxReconnectAttempts <= 0 {
		config.MaxReconnectAttempts = 120
	}
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = Duration(5 * time.Second)
	}
	if !config.EnableAutoReconnect {
		config.EnableAutoReconnect = true
	}

	// Validate required fields
	if config.Host == "" {
		return nil, fmt.Errorf("electrum rpc host is required")
	}
	if config.Port == 0 {
		return nil, fmt.Errorf("electrum rpc port is required")
	}
	if dbAdapter == nil {
		return nil, fmt.Errorf("dbAdapter is required")
	}

	log.Debug().Dur("dialTimeout", config.DialTimeout.ToDuration()).
		Dur("methodTimeout", config.MethodTimeout.ToDuration()).
		Dur("pingInterval", config.PingInterval.ToDuration()).
		Msg("[ElectrumClient] [NewElectrumClient] Using timeout configuration")

	log.Debug().Bool("enableAutoReconnect", config.EnableAutoReconnect).
		Int("maxReconnectAttempts", config.MaxReconnectAttempts).
		Dur("reconnectDelay", config.ReconnectDelay.ToDuration()).
		Msg("[ElectrumClient] [NewElectrumClient] Using reconnection configuration")

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
	params = append(params, c.electrumConfig.BatchSize)
	if c.scalarClient != nil {
		var err error
		c.custodialGroups, err = c.scalarClient.GetCovenantGroups(ctx)
		if err != nil {
			log.Error().Err(err).Msg("[ElectrumClient] failed to get custodian group")
		}
	}
	lastCheckpoint := c.getLastCheckpoint()
	log.Debug().Msgf("[ElectrumClient] [Start] Last checkpoint: %v", lastCheckpoint)
	// if lastCheckpoint.EventKey != "" {
	// 	params = append(params, lastCheckpoint.EventKey)
	// } else if c.electrumConfig.LastVaultTx != "" {
	// 	params = append(params, lastCheckpoint.EventKey)
	// }
	//params = append(params, 87220)
	params = append(params, 87970)

	// Start blockchain header subscription with reconnection logic
	go c.startBlockchainHeaderSubscriptionWithReconnect(ctx)

	//wait for lastblock is updated
	for {
		if c.GetCurrentHeight() > 0 {
			break
		}
		log.Debug().Msgf("[ElectrumClient] [Start] Waiting for current height to be updated")
		time.Sleep(1 * time.Second)
	}
	log.Debug().Int64("CurrentHeight", c.GetCurrentHeight()).Msgf("[ElectrumClient] [Start] Subscribing to vault transactions with params: %v", params)

	// Start vault transaction subscription with reconnection logic
	go c.startVaultTransactionSubscriptionWithReconnect(ctx, params)

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

// GetCurrentHeight returns the current height in a thread-safe manner
func (c *Client) GetCurrentHeight() int64 {
	return atomic.LoadInt64(&c.currentHeight)
}

// SetCurrentHeight sets the current height in a thread-safe manner
func (c *Client) SetCurrentHeight(height int64) {
	atomic.StoreInt64(&c.currentHeight, height)
}

// recreateConnection recreates the electrum client connection
func (c *Client) recreateConnection() error {
	rpcEndpoint := fmt.Sprintf("%s:%d", c.electrumConfig.Host, c.electrumConfig.Port)

	// Close existing connection if it exists
	if c.Electrs != nil {
		c.Electrs.Close()
	}

	electrs, err := electrum.Connect(&electrum.Options{
		Dial: func() (net.Conn, error) {
			return net.DialTimeout("tcp", rpcEndpoint, c.electrumConfig.DialTimeout.ToDuration())
		},
		MethodTimeout:   c.electrumConfig.MethodTimeout.ToDuration(),
		PingInterval:    c.electrumConfig.PingInterval.ToDuration(),
		SoftwareVersion: "scalar-relayer",
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to reconnect to electrum server at %s", rpcEndpoint)
		return err
	}

	c.Electrs = electrs
	log.Info().Msgf("Successfully reconnected to electrum server at %s", rpcEndpoint)
	return nil
}

// startBlockchainHeaderSubscriptionWithReconnect starts the blockchain header subscription with automatic reconnection
func (c *Client) startBlockchainHeaderSubscriptionWithReconnect(ctx context.Context) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Context cancelled, stopping reconnection attempts")
			return
		default:
			if attempt > 0 {
				log.Info().Int("attempt", attempt).Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Attempting to reconnect")
			} else {
				log.Debug().Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Starting blockchain header subscription")
			}

			// Create a context with timeout for this subscription attempt
			subscriptionCtx, cancel := context.WithTimeout(ctx, c.electrumConfig.MethodTimeout.ToDuration())

			// Subscribe to blockchain headers in a goroutine
			subscriptionDone := make(chan error, 1)
			go func() {
				defer cancel()
				c.Electrs.BlockchainHeaderSubscribe(subscriptionCtx, c.BlockchainHeaderHandler)
				subscriptionDone <- nil
			}()

			// Wait for either completion or timeout
			select {
			case <-subscriptionCtx.Done():
				if subscriptionCtx.Err() == context.DeadlineExceeded {
					log.Warn().Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Subscription timeout detected")
				} else {
					log.Info().Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Context cancelled")
					return
				}
			case err := <-subscriptionDone:
				if err != nil {
					log.Error().Err(err).Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Subscription error")
				} else {
					log.Info().Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Blockchain header subscription completed successfully")
					return
				}
			}
		}

		// If we reach here, there was a timeout or error
		attempt++
		if !c.electrumConfig.EnableAutoReconnect || attempt > c.electrumConfig.MaxReconnectAttempts {
			log.Error().Int("attempt", attempt).Int("maxAttempts", c.electrumConfig.MaxReconnectAttempts).
				Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Max reconnection attempts reached, stopping")
			return
		}

		log.Warn().Int("attempt", attempt).Int("maxAttempts", c.electrumConfig.MaxReconnectAttempts).
			Dur("delay", c.electrumConfig.ReconnectDelay.ToDuration()).
			Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Reconnecting after timeout")

		// Wait before reconnecting
		time.Sleep(c.electrumConfig.ReconnectDelay.ToDuration())

		// Try to recreate the connection
		if err := c.recreateConnection(); err != nil {
			log.Error().Err(err).Int("attempt", attempt).
				Msg("[ElectrumClient] [startBlockchainHeaderSubscriptionWithReconnect] Failed to recreate connection")
			continue
		}
	}
}

// startVaultTransactionSubscriptionWithReconnect starts the vault transaction subscription with automatic reconnection
func (c *Client) startVaultTransactionSubscriptionWithReconnect(ctx context.Context, params []interface{}) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Context cancelled, stopping reconnection attempts")
			return
		default:
			if attempt > 0 {
				log.Info().Int("attempt", attempt).Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Attempting to reconnect")
			} else {
				log.Debug().Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Starting vault transaction subscription")
			}

			// Create a context with timeout for this subscription attempt
			subscriptionCtx, cancel := context.WithTimeout(ctx, c.electrumConfig.MethodTimeout.ToDuration())

			// Subscribe to vault transactions in a goroutine
			subscriptionDone := make(chan error, 1)
			go func() {
				defer cancel()
				c.Electrs.SubscribeEvent(subscriptionCtx, types.VaultBlocksSubscribe, c.HandleValueBlockWithVaultTxs, params...)
				subscriptionDone <- nil
			}()

			// Wait for either completion or timeout
			select {
			case <-subscriptionCtx.Done():
				if subscriptionCtx.Err() == context.DeadlineExceeded {
					log.Warn().Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Subscription timeout detected")
				} else {
					log.Info().Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Context cancelled")
					return
				}
			case err := <-subscriptionDone:
				if err != nil {
					log.Error().Err(err).Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Subscription error")
				} else {
					log.Info().Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Vault transaction subscription completed successfully")
					return
				}
			}
		}

		// If we reach here, there was a timeout or error
		attempt++
		if !c.electrumConfig.EnableAutoReconnect || attempt > c.electrumConfig.MaxReconnectAttempts {
			log.Error().Int("attempt", attempt).Int("maxAttempts", c.electrumConfig.MaxReconnectAttempts).
				Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Max reconnection attempts reached, stopping")
			return
		}

		log.Warn().Int("attempt", attempt).Int("maxAttempts", c.electrumConfig.MaxReconnectAttempts).
			Dur("delay", c.electrumConfig.ReconnectDelay.ToDuration()).
			Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Reconnecting after timeout")

		// Wait before reconnecting
		time.Sleep(c.electrumConfig.ReconnectDelay.ToDuration())

		// Try to recreate the connection
		if err := c.recreateConnection(); err != nil {
			log.Error().Err(err).Int("attempt", attempt).
				Msg("[ElectrumClient] [startVaultTransactionSubscriptionWithReconnect] Failed to recreate connection")
			continue
		}
	}
}
