package btc

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/rpcclient"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/scalarorg/relayers/config"
)

var BtcClients []*BtcClientType

type BtcClientType struct {
	client *rpcclient.Client
	config *config.BtcNetworkConfig
}

func NewBtcClients(configs []config.BtcNetworkConfig) error {
	for _, cfg := range configs {
		// Create a new instance of config for each client
		clientConfig := cfg

		// Configure connection
		connCfg := &rpcclient.ConnConfig{
			Host:         fmt.Sprintf("%s:%d", clientConfig.Host, clientConfig.Port),
			User:         clientConfig.User,
			Pass:         clientConfig.Password,
			HTTPPostMode: true,
			DisableTLS:   clientConfig.SSL == nil || !*clientConfig.SSL,
		}

		// Create new client
		client, err := rpcclient.New(connCfg, nil)
		if err != nil {
			return fmt.Errorf("failed to create BTC client for network %s: %w", clientConfig.Network, err)
		}

		BtcClients = append(BtcClients, &BtcClientType{
			client: client,
			config: &clientConfig,
		})
	}

	if len(BtcClients) == 0 {
		return fmt.Errorf("no BTC client was initialized")
	}
	return nil
}

// GetTransaction retrieves detailed information about a transaction given its ID
func (c *BtcClientType) GetTransaction(txID string) (*btcjson.GetTransactionResult, error) {
	result, err := c.client.RawRequest("gettransaction", []json.RawMessage{[]byte(`"` + txID + `"`)})
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction info: %w", err)
	}

	var tx btcjson.GetTransactionResult
	if err := json.Unmarshal(result, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &tx, nil
}

// IsSigner returns true if the client is configured as a signer node
func (c *BtcClientType) IsSigner() bool {
	return strings.ToLower(c.config.Type) == "signer"
}

// IsBroadcast returns true if the client is configured as a broadcast node
func (c *BtcClientType) IsBroadcast() bool {
	return strings.ToLower(c.config.Type) == "broadcast"
}
