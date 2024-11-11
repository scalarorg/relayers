package cosmos

type CosmosNetworkConfig struct {
	ChainID       string `mapstructure:"chain_id"`
	RPCUrl        string `mapstructure:"rpc_url"`
	LCDUrl        string `mapstructure:"lcd_url"`
	WS            string `mapstructure:"ws"`
	Denom         string `mapstructure:"denom"`
	Mnemonic      string `mapstructure:"mnemonic"`
	GasPrice      string `mapstructure:"gas_price"`
	BroadcastMode string `mapstructure:"broadcast_mode"`
}
