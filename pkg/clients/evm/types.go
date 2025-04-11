package evm

import (
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/go-utils/types"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	"github.com/scalarorg/relayers/pkg/clients/evm/parser"
	"github.com/scalarorg/relayers/pkg/events"
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
	StartBlock   uint64        `mapstructure:"start_block"`
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
	mutex                   sync.Mutex
	logs                    []ethTypes.Log
	lastSwitchedPhaseEvents map[string]*contracts.IScalarGatewaySwitchPhase //Map each custodian group to the last switched phase event
	Recovered               atomic.Bool                                     //True if logs are recovered
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
	redeemTokenEvent, ok := GetEventByName(events.EVENT_EVM_REDEEM_TOKEN)
	if !ok {
		log.Error().Msgf("RedeemToken event not found")
		return
	}
	switchedPhaseEvent, ok := GetEventByName(events.EVENT_EVM_SWITCHED_PHASE)
	if !ok {
		log.Error().Msgf("SwitchedPhase event not found")
		return
	}
	for _, log := range logs {
		if log.Topics[0] == redeemTokenEvent.ID {
			//Check if the redeem event is in the last phase
			if m.isRedeemLogInLastPhase(redeemTokenEvent, log) {
				m.logs = append(m.logs, log)
			}
		} else if log.Topics[0] != switchedPhaseEvent.ID {
			m.logs = append(m.logs, log)
		}
	}
}
func (m *MissingLogs) isRedeemLogInLastPhase(redeemTokenEvent *abi.Event, eventLog ethTypes.Log) bool {
	redeemToken := &contracts.IScalarGatewayRedeemToken{
		Raw: eventLog,
	}
	err := parser.ParseEventData(&eventLog, redeemTokenEvent.Name, redeemToken)
	if err != nil {
		log.Error().Err(err).Msgf("failed to parse event %s", redeemTokenEvent.Name)
		return false
	}
	groupUid := hex.EncodeToString(redeemToken.CustodianGroupUID[:])
	lastSwitchedPhase, ok := m.GetLastSwitchedPhaseEvent(groupUid)
	if !ok {
		log.Error().Str("groupUid", groupUid).Msgf("Last switched phase event not found")
		return false
	}
	if redeemToken.Sequence != lastSwitchedPhase.Sequence {
		log.Error().Str("groupUid", groupUid).Msgf("Redeem event sequence %d is not equal to last switched phase sequence %d", redeemToken.Sequence, lastSwitchedPhase.Sequence)
		return false
	}
	return true
}
func (m *MissingLogs) SetLastSwitchedPhaseEvents(mapSwitchedPhaseEvents map[string]*contracts.IScalarGatewaySwitchPhase) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.lastSwitchedPhaseEvents = mapSwitchedPhaseEvents
	_, ok := GetEventByName(events.EVENT_EVM_SWITCHED_PHASE)
	if !ok {
		log.Error().Msgf("SwitchedPhase event not found")
		return
	}
	// for groupUid, eventLog := range mapSwitchedPhaseEvents {
	// 	params, err := event.Inputs.Unpack(eventLog.Data)
	// 	if err != nil {
	// 		log.Error().Msgf("Failed to unpack switched phase event: %v", err)
	// 		return
	// 	}
	// 	log.Info().Any("params", params).Msgf("Switched phase event: %v", eventLog)

	// 	// fromPhase := params[0].(uint8)
	// 	// toPhase := params[1].(uint8)
	// 	m.lastSwitchedPhaseEvent[eventLog.Topics[2].String()] = eventLog
	// }

}
func (m *MissingLogs) GetLastSwitchedPhaseEvent(groupUid string) (*contracts.IScalarGatewaySwitchPhase, bool) {
	lastSwitchedPhase, ok := m.lastSwitchedPhaseEvents[groupUid]
	return lastSwitchedPhase, ok
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
