package types

import (
	"math"

	"github.com/btcsuite/btcd/btcutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/bitcoin-vault/go-utils/btc"
	"github.com/scalarorg/bitcoin-vault/go-utils/types"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
)

type ExecuteParams struct {
	SourceChain      string
	SourceAddress    string
	ContractAddress  common.Address
	PayloadHash      [32]byte
	SourceTxHash     [32]byte
	SourceEventIndex uint64
}

// AxelarListenerEvent represents an event with generic type T for parsed events
type ScalarListenerEvent[T any] struct {
	TopicID    string
	Type       string
	ParseEvent func(events map[string][]string) (T, error)
}

// IBCEvent represents a generic IBC event with generic type T for Args
type IBCEvent[T any] struct {
	Hash        string `json:"hash"`
	SrcChannel  string `json:"srcChannel,omitempty"`
	DestChannel string `json:"destChannel,omitempty"`
	Args        T      `json:"args"`
}

// IBCPacketEvent represents an IBC packet event
type IBCPacketEvent struct {
	Hash        string      `json:"hash"`
	SrcChannel  string      `json:"srcChannel"`
	DestChannel string      `json:"destChannel"`
	Denom       string      `json:"denom"`
	Amount      string      `json:"amount"`
	Sequence    int         `json:"sequence"`
	Memo        interface{} `json:"memo"`
}

// ------ Payloads ------
// TODO: USING COSMOS SDK TO DEFINE THESE TYPES LATER
const (
	ChainsProtobufPackage = "scalar.chains.v1beta1"
	AxelarProtobufPackage = "scalar.scalarnet.v1beta1"
)

// Fee represents a fee structure with amount and recipient
type Fee struct {
	Amount    *sdk.Coin `json:"amount,omitempty"` // Optional amount field using Cosmos SDK Coin type
	Recipient []byte    `json:"recipient"`        // Recipient as byte array
}

// ConfirmGatewayTxRequest represents a request to confirm a gateway transaction
type ConfirmGatewayTxRequest struct {
	Sender []byte `json:"sender"`
	Chain  string `json:"chain"`
	TxID   []byte `json:"txId"`
}

// CallContractRequest represents a request to call a contract
type CallContractRequest struct {
	Sender          []byte `json:"sender"`
	Chain           string `json:"chain"`
	ContractAddress string `json:"contractAddress"`
	Payload         []byte `json:"payload"`
	Fee             *Fee   `json:"fee,omitempty"`
}

// RouteMessageRequest represents a request to route a message
type RouteMessageRequest struct {
	Sender  []byte `json:"sender"`
	ID      string `json:"id"`
	Payload []byte `json:"payload"`
}

// SignCommandsRequest represents a request to sign commands
type SignCommandsRequest struct {
	Sender []byte `json:"sender"`
	Chain  string `json:"chain"`
}

type PsbtParams struct {
	ScalarTag       []byte
	Version         uint8
	ProtocolTag     []byte
	NetworkKind     types.NetworkKind //mainnet, testnet
	NetworkType     string            //mainnet, testnet, testnet4, regtest
	CustodianPubKey []types.PublicKey
	CustodianQuorum uint8
	CustodianScript []byte
}

//	type SignPsbtsRequest struct {
//		ChainName string
//		Psbts     []covExported.Psbt
//	}
type CreatePsbtRequest struct {
	Outpoints []CommandOutPoint
	Params    PsbtParams
}

// Calculate taproot address from covenant pubkey and network
func (p *PsbtParams) GetTaprootAddress() (btcutil.Address, error) {
	return btc.ScriptPubKeyToAddress(p.CustodianScript, p.NetworkType)
}

//	type RedeemTx struct {
//		BlockHeight uint64
//		TxHash      string
//		LogIndex    uint
//	}
type Session struct {
	Sequence uint64
	Phase    uint8
}

func (s *Session) Cmp(other *Session) int64 {
	if other == nil {
		return math.MaxInt64
	}
	var diffSeq, diffPhase int64
	if s.Sequence >= other.Sequence {
		diffSeq = int64(s.Sequence - other.Sequence)
	} else {
		diffSeq = -int64(other.Sequence - s.Sequence)
	}

	if s.Phase >= other.Phase {
		diffPhase = int64(s.Phase - other.Phase)
	} else {
		diffPhase = -int64(other.Phase - s.Phase)
	}

	return diffSeq*2 + diffPhase
}

// Eache chainId container an array of SwitchPhaseEvents,
// with the first element is switch to Preparing phase
type GroupRedeemSessions struct {
	GroupUid          string
	MaxSession        Session
	MinSession        Session
	SwitchPhaseEvents map[string][]*contracts.IScalarGatewaySwitchPhase //Map by chainId
	RedeemTokenEvents map[string][]*contracts.IScalarGatewayRedeemToken
}

/*
* For each custodian group, maximum difference between the session of evms is a phase
 */
func (s *GroupRedeemSessions) Construct() {
	s.MinSession.Sequence = math.MaxInt64
	//Find the max, min session
	for _, switchPhaseEvent := range s.SwitchPhaseEvents {
		lastEvent := switchPhaseEvent[len(switchPhaseEvent)-1]
		if s.MaxSession.Sequence < lastEvent.Sequence {
			s.MaxSession.Sequence = lastEvent.Sequence
			s.MaxSession.Phase = lastEvent.To
		} else if s.MaxSession.Sequence == lastEvent.Sequence && s.MaxSession.Phase < lastEvent.To {
			s.MaxSession.Phase = lastEvent.To
		}
		if s.MinSession.Sequence > lastEvent.Sequence {
			s.MinSession.Sequence = lastEvent.Sequence
			s.MinSession.Phase = lastEvent.To
		} else if s.MinSession.Sequence == lastEvent.Sequence && s.MinSession.Phase > lastEvent.To {
			s.MinSession.Phase = lastEvent.To
		}
	}
	diff := s.MaxSession.Cmp(&s.MinSession)
	log.Info().Str("groupUid", s.GroupUid).Int64("diff", diff).
		Any("maxSession", s.MaxSession).
		Any("minSession", s.MinSession).
		Msg("[GroupRedeemSessions] [ConstructPreparingPhase]")

	if s.MaxSession.Phase == uint8(covExported.Preparing) {
		s.ConstructPreparingPhase()
	} else if s.MaxSession.Phase == uint8(covExported.Executing) {
		s.ConstructExecutingPhase()
	}
}

func (s *GroupRedeemSessions) ConstructPreparingPhase() {
	diff := s.MaxSession.Cmp(&s.MinSession)
	if diff == 0 {
		log.Warn().Str("groupUid", s.GroupUid).Msg("[GroupRedeemSessions] [ConstructPreparingPhase] max session and min session are the same")
		//Each chain keep only one switch phase event to Preparing phase
		for chainId, switchPhaseEvent := range s.SwitchPhaseEvents {
			if len(switchPhaseEvent) == 0 {
				continue
			}
			if len(switchPhaseEvent) == 2 {
				s.SwitchPhaseEvents[chainId] = switchPhaseEvent[1:]
			}
		}
		//Keep all redeem token events of the max session's sequence
		for chainId, redeemTokenEvents := range s.RedeemTokenEvents {
			currentSessionEvents := make([]*contracts.IScalarGatewayRedeemToken, 0)
			for _, redeemTokenEvent := range redeemTokenEvents {
				if redeemTokenEvent.Sequence == s.MaxSession.Sequence {
					currentSessionEvents = append(currentSessionEvents, redeemTokenEvent)
				}
			}
			s.RedeemTokenEvents[chainId] = currentSessionEvents
		}
	} else {
		//These are some chains switch to the preparing phase, and some other is in execution phase from previous session
		//We don't need to recreate Redeem transaction to btc,
		//show we don't need to send RedeemEvent for confirmation
		s.RedeemTokenEvents = make(map[string][]*contracts.IScalarGatewayRedeemToken)
		//Remove old switch phase event
		//Find all chains with 2 events [Preparing, Executing], remove the first event
		for chainId, switchPhaseEvent := range s.SwitchPhaseEvents {
			if len(switchPhaseEvent) == 2 && switchPhaseEvent[0].To == uint8(covExported.Preparing) && switchPhaseEvent[1].To == uint8(covExported.Executing) {
				s.SwitchPhaseEvents[chainId] = switchPhaseEvent[1:]
			}
		}
	}
}

func (s *GroupRedeemSessions) ConstructExecutingPhase() {
	//For both case diff == 0 and diff = 1, we need to resend the redeem transaction to the scalar network
	//Expecting all chains are switching to the executing phase
	for chainId, switchPhaseEvent := range s.SwitchPhaseEvents {
		if switchPhaseEvent[0].Sequence < s.MaxSession.Sequence {
			log.Warn().Str("chainId", chainId).Any("First preparing event", switchPhaseEvent[0]).
				Msg("[Relayer] [RecoverRedeemSessions] Session is too low. Some thing wrong")
		}
	}
	//We resend to the scalar network onlye the redeem transaction of the last session
	for chainId, redeemTokenEvent := range s.RedeemTokenEvents {
		for _, event := range redeemTokenEvent {
			if event.Sequence < s.MaxSession.Sequence {
				log.Warn().Str("chainId", chainId).Any("Redeem transaction", event).
					Msg("[Relayer] [RecoverRedeemSessions] Redeem transaction is too low. Some thing wrong")
			}
		}
	}
}

// Each chain store switch phase events array with 1 or 2 elements of the form:
// 1. [Preparing]
// 2. [Preparing, Executing] in the same sequence
type ChainRedeemSessions struct {
	SwitchPhaseEvents map[string][]*contracts.IScalarGatewaySwitchPhase //Map by custodian group uid
	RedeemTokenEvents map[string][]*contracts.IScalarGatewayRedeemToken
}

// Return number of events added
func (s *ChainRedeemSessions) AppendSwitchPhaseEvent(groupUid string, event *contracts.IScalarGatewaySwitchPhase) int {
	//Put switch phase event in the first position
	switchPhaseEvents, ok := s.SwitchPhaseEvents[groupUid]
	if !ok {
		s.SwitchPhaseEvents[groupUid] = []*contracts.IScalarGatewaySwitchPhase{event}
		return 1
	}
	if len(switchPhaseEvents) >= 2 {
		log.Warn().Str("groupUid", groupUid).Msg("[ChainRedeemSessions] [AppendSwitchPhaseEvent] switch phase events already has 2 elements")
		return 2
	}
	currentPhase := switchPhaseEvents[0]
	log.Warn().Str("groupUid", groupUid).Any("current element", currentPhase).
		Any("incomming element", event).
		Msg("[ChainRedeemSessions] [AppendSwitchPhaseEvent] switch phase event has the same sequence")
	if currentPhase.Sequence == event.Sequence {
		if currentPhase.To == uint8(covExported.Preparing) && event.To == uint8(covExported.Executing) {
			s.SwitchPhaseEvents[groupUid] = append(switchPhaseEvents, event)
			return 2
		} else if currentPhase.To == uint8(covExported.Executing) && event.To == uint8(covExported.Preparing) {
			s.SwitchPhaseEvents[groupUid] = []*contracts.IScalarGatewaySwitchPhase{event, currentPhase}
			return 2
		} else {
			log.Warn().Msg("[ChainRedeemSessions] [AppendSwitchPhaseEvent] event is already in the list")
			return 1
		}
	} else if event.Sequence < currentPhase.Sequence {
		if event.Sequence == currentPhase.Sequence-1 && event.To == uint8(covExported.Executing) && currentPhase.To == uint8(covExported.Preparing) {
			s.SwitchPhaseEvents[groupUid] = []*contracts.IScalarGatewaySwitchPhase{event, currentPhase}
			return 2
		}
		log.Warn().Msg("[ChainRedeemSessions] [AppendSwitchPhaseEvent] incomming event is too old")
		return 1
	} else {
		//event.Sequence > currentPhase.Sequence
		if currentPhase.Sequence == event.Sequence-1 && currentPhase.To == uint8(covExported.Executing) && event.To == uint8(covExported.Preparing) {
			s.SwitchPhaseEvents[groupUid] = []*contracts.IScalarGatewaySwitchPhase{currentPhase, event}
			return 2
		}
		log.Warn().Msg("[ChainRedeemSessions] [AppendSwitchPhaseEvent] Incomming event is too high, set it as an unique item in the list")
		s.SwitchPhaseEvents[groupUid] = []*contracts.IScalarGatewaySwitchPhase{event}
		return 1
	}
}

// Put redeem token event in the first position
// we keep only the redeem transaction of the max session's sequence
func (s *ChainRedeemSessions) AppendRedeemTokenEvent(groupUid string, event *contracts.IScalarGatewayRedeemToken) {
	redeemEvents, ok := s.RedeemTokenEvents[groupUid]
	if !ok {
		s.RedeemTokenEvents[groupUid] = []*contracts.IScalarGatewayRedeemToken{event}
	} else if len(redeemEvents) > 0 {
		lastInsertedEvent := redeemEvents[0]
		if event.Sequence > lastInsertedEvent.Sequence {
			s.RedeemTokenEvents[groupUid] = []*contracts.IScalarGatewayRedeemToken{event}
		} else if lastInsertedEvent.Sequence == event.Sequence {
			s.RedeemTokenEvents[groupUid] = append([]*contracts.IScalarGatewayRedeemToken{event}, redeemEvents...)
		} else {
			log.Warn().Str("groupUid", groupUid).Any("last inserted event", lastInsertedEvent).
				Any("incomming event", event).
				Msg("[ChainRedeemSessions] [AppendRedeemTokenEvent] ignore redeem token tx with lower sequence")
		}
	}
}
