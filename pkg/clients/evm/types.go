package evm

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/scalarorg/bitcoin-vault/go-utils/types"
)

const (
	COMPONENT_NAME = "EvmClient"
	RETRY_INTERVAL = time.Second * 12 // Initial retry interval
)

type Byte32 [32]uint8
type Bytes []byte
type EvmNetworkConfig struct {
	ChainID      uint64        `mapstructure:"chain_id"`
	ID           string        `mapstructure:"id"`
	Name         string        `mapstructure:"name"`
	RPCUrl       string        `mapstructure:"rpc_url"`
	Gateway      string        `mapstructure:"gateway"`
	Finality     int           `mapstructure:"finality"`
	LastBlock    uint64        `mapstructure:"last_block"`
	PrivateKey   string        `mapstructure:"private_key"`
	GasLimit     uint64        `mapstructure:"gas_limit"`
	BlockTime    time.Duration `mapstructure:"blockTime"` //Timeout im ms for pending txs
	MaxRetry     int
	RecoverRange uint64 `mapstructure:"recover_range"` //Max block range to recover events in single query
	RetryDelay   time.Duration
	TxTimeout    time.Duration `mapstructure:"tx_timeout"` //Timeout for send txs (~3s)
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
	return types.ChainTypeEVM.String()
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

type ExecuteData[T any] struct {
	//Data
	Data T
	//Proof
	Operators  []common.Address
	Weights    []uint64
	Threshold  uint64
	Signatures []string
	//Input
	Input []byte
}
type ApproveContractCall struct {
	ChainId    uint64
	CommandIds [][32]byte
	Commands   []string
	Params     [][]byte
}
type DeployToken struct {
	//Data
	Name         string
	Symbol       string
	Decimals     uint8
	Cap          uint64
	TokenAddress common.Address
	MintLimit    uint64
}

type RedeemPhase struct {
	Sequence uint64
	Phase    uint8
}

type MissingLogs struct {
	mutex     sync.Mutex
	logs      []ethTypes.Log
	Recovered atomic.Bool //True if logs are recovered
}

func (m *MissingLogs) IsRecovered() bool {
	return m.Recovered.Load()
}
func (m *MissingLogs) SetRecovered(recovered bool) {
	m.Recovered.Store(recovered)
}

func (m *MissingLogs) AppendLogs(logs []ethTypes.Log) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logs = append(m.logs, logs...)
}

func (m *MissingLogs) GetLogs(count int) []ethTypes.Log {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.logs) <= count {
		logs := m.logs
		m.logs = []ethTypes.Log{}
		return logs
	} else {
		logs := m.logs[:count]
		m.logs = m.logs[count:]
		return logs
	}
}
