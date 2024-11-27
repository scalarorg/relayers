package evm

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/scalarorg/bitcoin-vault/go-utils/chain"
)

const (
	COMPONENT_NAME = "EvmClient"
	RETRY_INTERVAL = time.Second * 12 // Initial retry interval
)

type Byte32 [32]uint8
type Bytes []byte
type EvmNetworkConfig struct {
	ChainID    uint64        `mapstructure:"chain_id"`
	ID         string        `mapstructure:"id"`
	Name       string        `mapstructure:"name"`
	RPCUrl     string        `mapstructure:"rpc_url"`
	Gateway    string        `mapstructure:"gateway"`
	Finality   int           `mapstructure:"finality"`
	LastBlock  uint64        `mapstructure:"last_block"`
	PrivateKey string        `mapstructure:"private_key"`
	GasLimit   uint64        `mapstructure:"gas_limit"`
	BlockTime  time.Duration `mapstructure:"blockTime"` //Timeout im ms for pending txs
	MaxRetry   int
	RetryDelay time.Duration
	TxTimeout  time.Duration `mapstructure:"tx_timeout"` //Timeout for send txs (~3s)
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
	CommandIds [][32]byte
	Commands   []string
	Params     [][]byte
	//Proof
	Operators  []common.Address
	Weights    []uint64
	Threshold  uint64
	Signatures []string
	//Input
	Input []byte
}
