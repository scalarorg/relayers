package evm

import "time"

type EvmNetworkConfig struct {
	ChainID    string `mapstructure:"chain_id"`
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	RPCUrl     string `mapstructure:"rpc_url"`
	Gateway    string `mapstructure:"gateway"`
	Finality   int    `mapstructure:"finality"`
	LastBlock  uint64 `mapstructure:"last_block"`
	PrivateKey string `mapstructure:"private_key"`
	MaxRetry   int
	RetryDelay time.Duration
	TxTimeout  time.Duration
}
