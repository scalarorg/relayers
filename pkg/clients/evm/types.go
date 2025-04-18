package evm

import (
	"encoding/hex"
	"fmt"
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
	AuthWeighted string        `mapstructure:"auth_weighted"`
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
	chainId                     string
	mutex                       sync.Mutex
	logs                        []ethTypes.Log
	lastPreparingSwitchedEvents map[string]*contracts.IScalarGatewaySwitchPhase //Map each custodian group to the last switched phase event
	lastExecutingSwitchedEvents map[string]*contracts.IScalarGatewaySwitchPhase //Map each custodian group to the last switched phase event
	Recovered                   atomic.Bool                                     //True if logs are recovered
	RedeemTxs                   map[string][]string                             //Map each destination chain to the array of redeem txs
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
	log.Info().Str("Chain", m.chainId).Int("Number of event logs", len(logs)).Msgf("[EvmClient] [AppendLogs] appending logs")
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
	for _, eventLog := range logs {
		if eventLog.Topics[0] == redeemTokenEvent.ID {
			m.appendRedeemLog(redeemTokenEvent, eventLog)
		} else if eventLog.Topics[0] != switchedPhaseEvent.ID {
			m.logs = append(m.logs, eventLog)
		}
	}
	log.Info().Str("Chain", m.chainId).Int("Number of logs", len(m.logs)).Msgf("[EvmClient] [AppendLogs] appended logs")
}
func (m *MissingLogs) appendRedeemLog(redeemEvent *abi.Event, eventLog ethTypes.Log) error {
	redeemToken := &contracts.IScalarGatewayRedeemToken{
		Raw: eventLog,
	}
	err := parser.ParseEventData(&eventLog, redeemEvent.Name, redeemToken)
	if err != nil {
		log.Error().Err(err).Msgf("failed to parse event %s", redeemEvent.Name)
		return err
	}
	groupUid := hex.EncodeToString(redeemToken.CustodianGroupUID[:])
	if m.isRedeemLogInLastPhase(groupUid, redeemToken) {
		log.Info().Str("Chain", m.chainId).Any("eventLog", eventLog).Msgf("[EvmClient] [AppendLogs] appending redeem log")
		m.logs = append(m.logs, eventLog)
		if m.RedeemTxs == nil {
			m.RedeemTxs = make(map[string][]string)
		}
		txs := m.RedeemTxs[redeemToken.DestinationChain]
		m.RedeemTxs[redeemToken.DestinationChain] = append(txs, redeemToken.Raw.TxHash.Hex())
	} else {
		log.Info().Str("Chain", m.chainId).Any("eventLog", eventLog).Msgf("[EvmClient] [AppendLogs] skipping redeem log due to not in last phase")
		return fmt.Errorf("redeem log not in last phase")
	}
	return nil
}
func (m *MissingLogs) isRedeemLogInLastPhase(groupUid string, redeemToken *contracts.IScalarGatewayRedeemToken) bool {
	lastSwitchedPhase, ok := m.lastPreparingSwitchedEvents[groupUid]
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
func (m *MissingLogs) SetLastSwitchedEvents(mapPreparingEvents map[string]*contracts.IScalarGatewaySwitchPhase,
	mapExecutingEvents map[string]*contracts.IScalarGatewaySwitchPhase) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.lastPreparingSwitchedEvents = mapPreparingEvents
	m.lastExecutingSwitchedEvents = mapExecutingEvents
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
func (m *MissingLogs) GetExecutingEvents() map[string]*contracts.IScalarGatewaySwitchPhase {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.lastExecutingSwitchedEvents
}
func (m *MissingLogs) GetLogs(count int) []ethTypes.Log {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var logs []ethTypes.Log
	if len(m.logs) <= count {
		logs = m.logs
		m.logs = []ethTypes.Log{}
	} else {
		logs = m.logs[:count]
		m.logs = m.logs[count:]
	}
	log.Info().Str("Chain", m.chainId).Int("Number of logs", len(logs)).
		Int("Remaining logs", len(m.logs)).
		Msgf("[MissingLogs] [GetLogs] returned logs")
	return logs
}
