package cosmos

import "github.com/scalarorg/bitcoin-vault/go-utils/chain"

const (
	DEFAULT_GAS_ADJUSTMENT = 1.2
)

type CosmosNetworkConfig struct {
	ChainID       uint64   `mapstructure:"chain_id"`
	ID            string   `mapstructure:"id"`
	Name          string   `mapstructure:"name"`
	RPCUrl        string   `mapstructure:"rpc_url"`
	LCDUrl        string   `mapstructure:"lcd_url"`
	WSUrl         string   `mapstructure:"ws_url"`
	Denom         string   `mapstructure:"denom"`
	Mnemonic      string   `mapstructure:"mnemonic"`
	Bip44Path     string   `mapstructure:"bip44path"`
	GasPrice      float64  `mapstructure:"gas_price"`
	BroadcastMode string   `mapstructure:"broadcast_mode"`
	MaxRetries    int      `mapstructure:"max_retries"`
	RetryInterval int64    `mapstructure:"retry_interval"` //milliseconds
	PrivateKeys   []string `mapstructure:"private_keys"`
	PublicKeys    []string `mapstructure:"public_keys"`
	SignerNetwork string   `mapstructure:"signer_network"`
}

func (c *CosmosNetworkConfig) GetFamily() string {
	return chain.ChainTypeCosmos.String()
}

func (c *CosmosNetworkConfig) GetChainId() uint64 {
	return c.ChainID
}
func (c *CosmosNetworkConfig) GetId() string {
	return c.ID
}
func (c *CosmosNetworkConfig) GetName() string {
	return c.Name
}
