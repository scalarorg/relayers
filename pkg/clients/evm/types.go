package evm

import "time"

type EvmNetworkConfig struct {
	ChainID    string `mapstructure:"chain_id"`
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	RPCUrl     string `mapstructure:"rpc_url"`
	Gateway    string `mapstructure:"gateway"`
	Finality   int    `mapstructure:"finality"`
	LastBlock  string `mapstructure:"last_block"`
	PrivateKey string `mapstructure:"private_key"`
	MaxRetry   int
	RetryDelay time.Duration
	TxTimeout  time.Duration
}

// type RuntimeEvmNetworkConfig struct {
// 	ChainID       string  `mapstructure:"chainId"`
// 	ID            string  `mapstructure:"id"`
// 	Name          string  `mapstructure:"name"`
// 	MnemonicEvm   *string `mapstructure:"mnemonic_evm,omitempty"`
// 	Mnemonic      *string `mapstructure:"mnemonic,omitempty"`
// 	WalletIndex   *string `mapstructure:"walletIndex,omitempty"`
// 	RPCUrl        string  `mapstructure:"rpcUrl"`
// 	KeyPath       string  `mapstructure:"keyPath"`
// 	PrivateKey    *string `mapstructure:"privateKey,omitempty"`
// 	Gateway       *string `mapstructure:"gateway,omitempty"`
// 	GasService    *string `mapstructure:"gasService,omitempty"`
// 	AuthWeighted  *string `mapstructure:"authWeighted,omitempty"`
// 	TokenDeployer *string `mapstructure:"tokenDeployer,omitempty"`
// 	SBtc          *string `mapstructure:"sBtc,omitempty"`
// }
