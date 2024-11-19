package evm

import (
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/scalarorg/bitcoin-vault/go-utils/chain"
)

const COMPONENT_NAME = "EvmClient"

var scalarGwExecuteAbi *abi.ABI

type Byte32 [32]byte
type Bytes []byte
type EvmNetworkConfig struct {
	ChainID    uint64 `mapstructure:"chain_id"`
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

func (c *EvmNetworkConfig) GetChainId() uint64 {
	return c.ChainID
}
func (c *EvmNetworkConfig) GetId() string {
	return c.ID
}
func (c *EvmNetworkConfig) GetName() string {
	return c.Name
}
func (c *EvmNetworkConfig) GetFamily() string {
	return chain.ChainTypeEVM.String()
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

type EvmEvent[T any] struct {
	Hash             string
	BlockNumber      uint64
	LogIndex         uint
	SourceChain      string
	DestinationChain string
	WaitForFinality  func() (*types.Receipt, error)
	Args             T
}
