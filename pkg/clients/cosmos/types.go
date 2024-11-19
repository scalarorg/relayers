package cosmos

const (
	DEFAULT_GAS_ADJUSTMENT = 1.2
)

type CosmosNetworkConfig struct {
	ChainID       string   `mapstructure:"chain_id"`
	RPCUrl        string   `mapstructure:"rpc_url"`
	LCDUrl        string   `mapstructure:"lcd_url"`
	WSUrl         string   `mapstructure:"ws_url"`
	Denom         string   `mapstructure:"denom"`
	Mnemonic      string   `mapstructure:"mnemonic"`
	GasPrice      float64  `mapstructure:"gas_price"`
	BroadcastMode string   `mapstructure:"broadcast_mode"`
	MaxRetries    int      `mapstructure:"max_retries"`
	RetryInterval int64    `mapstructure:"retry_interval"` //milliseconds
	PrivateKeys   []string `mapstructure:"private_keys"`
	PublicKeys    []string `mapstructure:"public_keys"`
	SignerNetwork string   `mapstructure:"signer_network"`
}
