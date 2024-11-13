package evm

import (
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var scalarGwExecuteAbi *abi.ABI

type Byte32 [32]byte
type Bytes []byte
type EvmNetworkConfig struct {
	ChainID    string `mapstructure:"chain_id"`
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	RPCUrl     string `mapstructure:"rpc_url"`
	Gateway    string `mapstructure:"gateway"`
	Finality   int    `mapstructure:"finality"`
	LastBlock  uint64 `mapstructure:"last_block"`
	PrivateKey string `mapstructure:"private_key"`
	GasLimit   uint64 `mapstructure:"gas_limit"`
	MaxRetry   int
	RetryDelay time.Duration
	TxTimeout  time.Duration
}

type DecodedExecuteData struct {
	//Data
	ChainId    uint64
	CommandIds []Byte32
	Commands   []string
	Params     []Bytes
	//Proof
	Operators  []string
	Weights    []uint64
	Threshold  uint64
	Signatures []Bytes
}
