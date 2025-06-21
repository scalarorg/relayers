package scalar

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/pkg/utils"
	chainsExported "github.com/scalarorg/scalar-core/x/chains/exported"
	"github.com/scalarorg/scalar-core/x/chains/types"
	covExported "github.com/scalarorg/scalar-core/x/covenant/exported"
	covtypes "github.com/scalarorg/scalar-core/x/covenant/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Add this new type definition

const (
	BroadcasterBatchSize                     = 1
	BlockEventsBase64Encoded                 = false
	EventTypeMintCommand                     = "scalar.chains.v1beta1.MintCommand"
	EventTypeContractCallApproved            = "scalar.chains.v1beta1.ContractCallApproved"
	EventTypeContractCallWithMintApproved    = "scalar.chains.v1beta1.ContractCallWithMintApproved"
	EventTypeTokenSent                       = "scalar.chains.v1beta1.EventTokenSent"
	EventTypeEVMEventCompleted               = "scalar.chains.v1beta1.EVMEventCompleted"
	EventTypeCommandBatchSigned              = "scalar.chains.v1beta1.CommandBatchSigned"
	EventTypeRedeemTokenApproved             = "scalar.chains.v1beta1.EventRedeemTokenApproved"
	EventTypeContractCallSubmitted           = "scalar.scalarnet.v1beta1.ContractCallSubmitted"
	EventTypeContractCallWithTokenSubmitted  = "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted"
	EventTypeSwitchPhaseStarted              = "scalar.covenant.v1beta1.SwitchPhaseStarted"
	EventTypeSwitchPhaseCompleted            = "scalar.covenant.v1beta1.SwitchPhaseCompleted"
	TokenSentEventTopicId                    = "tm.event='NewBlock' AND scalar.chains.v1beta1.EventTokenSent.event_id EXISTS"
	MintCommandEventTopicId                  = "tm.event='NewBlock' AND scalar.chains.v1beta1.MintCommand.event_id EXISTS"
	ContractCallApprovedEventTopicId         = "tm.event='NewBlock' AND scalar.chains.v1beta1.ContractCallApproved.event_id EXISTS"
	ContractCallWithMintApprovedEventTopicId = "tm.event='NewBlock' AND scalar.chains.v1beta1.ContractCallWithMintApproved.event_id EXISTS"
	CommandBatchSignedEventTopicId           = "tm.event='NewBlock' AND scalar.chains.v1beta1.CommandBatchSigned.event_id EXISTS"
	EVMCompletedEventTopicId                 = "tm.event='NewBlock' AND scalar.chains.v1beta1.EVMEventCompleted.event_id EXISTS"
	//For future use
	ContractCallSubmittedEventTopicId = "tm.event='Tx' AND scalar.scalarnet.v1beta1.ContractCallSubmitted.message_id EXISTS"
	ContractCallWithTokenEventTopicId = "tm.event='Tx' AND scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted.message_id EXISTS"
	ExecuteMessageEventTopicId        = "tm.event='Tx' AND message.action='ExecuteMessage'"
)

type EventHandlerCallBack[T any] func(events []IBCEvent[T])
type ListenerEvent[T any] struct {
	TopicId string
	Type    string
	Parser  func(events map[string][]string) ([]IBCEvent[T], error)
}

type IBCEvent[T any] struct {
	Hash        string `json:"hash"`
	SrcChannel  string `json:"srcChannel,omitempty"`
	DestChannel string `json:"destChannel,omitempty"`
	Args        T      `json:"args"`
}
type ScalarMessage interface {
	proto.Message
	UnmarshalJson(map[string]string) error
}

// var _ proto.Message = &EventTokenSent{}
func UnmarshalJson(jsonData map[string]string, e proto.Message) error {
	switch e := e.(type) {
	case *types.EventTokenSent:
		return UnmarshalTokenSent(jsonData, e)
	case *types.ContractCallApproved:
		return UnmarshalContractCallApproved(jsonData, e)
	case *types.EventContractCallWithMintApproved:
		return UnmarshalContractCallWithMintApproved(jsonData, e)
	case *types.CommandBatchSigned:
		return UnmarshalCommandBatchSigned(jsonData, e)
	case *types.ChainEventCompleted:
		return UnmarshalChainEventCompleted(jsonData, e)
	case *covtypes.SwitchPhaseStarted:
		return UnmarshalSwitchPhaseStarted(jsonData, e)
	default:
		return fmt.Errorf("unsupport type %T", e)
	}
}
func UnamrshalAsset(jsonData string) (sdk.Coin, error) {
	var rawCoin map[string]string
	err := json.Unmarshal([]byte(jsonData), &rawCoin)
	if err != nil {
		log.Debug().Err(err).Msg("Cannot unmarshalling coin data")
		return sdk.NewCoin("", sdk.NewInt(0)), err
	}
	denom := removeQuote(rawCoin["denom"])
	amount, ok := sdk.NewIntFromString(removeQuote(rawCoin["amount"]))
	if !ok {
		amount = sdk.NewInt(0)
	}
	return sdk.NewCoin(denom, amount), nil
}
func UnmarshalTokenSent(jsonData map[string]string, e *types.EventTokenSent) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	e.Sender = removeQuote(jsonData["sender"])
	e.DestinationAddress = removeQuote(jsonData["destination_address"])
	e.DestinationChain = exported.ChainName(removeQuote(jsonData["destination_chain"]))
	eventId := utils.NormalizeHash(removeQuote(jsonData["event_id"]))
	e.EventID = types.EventID(eventId)
	transferId, ok := sdk.NewIntFromString(removeQuote(jsonData["transfer_id"]))
	if ok {
		e.TransferID = exported.TransferID(transferId.Uint64())
	}
	e.CommandID = removeQuote(jsonData["command_id"])
	assetData, ok := jsonData["asset"]
	if ok {
		e.Asset, _ = UnamrshalAsset(assetData)
	}
	return nil
}

// {"asset":"{\"denom\":\"pBtc\",\"amount\":\"10000\"}","chain":"\"evm|11155111\"","destination_address":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"","destination_chain":"\"evm|97\"","event_id":"\"0x620bc60a616248eaf0a9f5b7e45db3f96eca31420c581034a6c59669cefb7de1-240\"","sender":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"","transfer_id":"\"0\""}

func UnmarshalChainEventCompleted(jsonData map[string]string, e *types.ChainEventCompleted) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	eventId := utils.NormalizeHash(removeQuote(jsonData["event_id"]))
	e.EventID = types.EventID(eventId)
	e.Type = removeQuote(jsonData["type"])
	return nil
}

func UnmarshalSwitchPhaseStarted(jsonData map[string]string, e *covtypes.SwitchPhaseStarted) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	sequence, err := strconv.ParseUint(removeQuote(jsonData["sequence"]), 10, 64)
	if err != nil {
		log.Warn().Msgf("Failed to decode sequence: %v, error: %v", jsonData["sequence"], err)
	}
	e.Sequence = sequence
	e.ExecuteData = removeQuote(jsonData["execute_data"])
	e.Phase = covExported.Phase(covExported.Phase_value[removeQuote(jsonData["phase"])])
	return nil
}

func UnmarshalContractCallApproved(jsonData map[string]string, e *types.ContractCallApproved) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	eventId := utils.NormalizeHash(removeQuote(jsonData["event_id"]))
	e.EventID = types.EventID(eventId)
	commandIDHex, err := DecodeIntArrayToHexString(jsonData["command_id"])
	if err != nil {
		log.Warn().Msgf("Failed to decode command ID: %v, error: %v", jsonData["command_id"], err)
	}
	e.CommandID, _ = types.HexToCommandID(commandIDHex)
	e.Sender = removeQuote(jsonData["sender"])
	e.DestinationChain = exported.ChainName(removeQuote(jsonData["destination_chain"]))
	e.ContractAddress = removeQuote(jsonData["contract_address"])

	payloadHex, err := DecodeIntArrayToHexString(removeQuote(jsonData["payload_hash"]))
	if err != nil {
		log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", jsonData["payload_hash"], err)
	}
	e.PayloadHash = chainsExported.Hash(common.HexToHash(payloadHex))
	log.Debug().Any("JsonData", jsonData).Msg("Input data")
	log.Debug().Any("result", e).Msg("Resut data")
	return nil
}

func UnmarshalContractCallWithMintApproved(jsonData map[string]string, e *types.EventContractCallWithMintApproved) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	eventId := utils.NormalizeHash(removeQuote(jsonData["event_id"]))
	e.EventID = types.EventID(eventId)
	commandIDHex, err := DecodeIntArrayToHexString(jsonData["command_id"])
	if err != nil {
		log.Warn().Msgf("Failed to decode command ID: %v, error: %v", jsonData["command_id"], err)
	}
	e.CommandID, _ = types.HexToCommandID(commandIDHex)
	e.Sender = removeQuote(jsonData["sender"])
	e.DestinationChain = exported.ChainName(removeQuote(jsonData["destination_chain"]))
	e.ContractAddress = removeQuote(jsonData["contract_address"])

	payloadHex, err := DecodeIntArrayToHexString(jsonData["payload_hash"])
	if err != nil {
		log.Warn().Msgf("Failed to decode payload hash: %v, error: %v", jsonData["payload_hash"], err)
	}
	e.PayloadHash = chainsExported.Hash(common.HexToHash(payloadHex))
	assetData, ok := jsonData["asset"]
	if ok {
		e.Asset, _ = UnamrshalAsset(assetData)
	}
	log.Debug().Any("JsonData", jsonData).Msg("Input data")
	log.Debug().Any("result", e).Msg("Resut data")
	return nil
}

func UnmarshalCommandBatchSigned(jsonData map[string]string, e *types.CommandBatchSigned) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	batchCommandId := removeQuote(jsonData["command_batch_id"])
	commandBatch, err := base64.StdEncoding.DecodeString(batchCommandId)
	if err != nil {
		log.Warn().Msgf("Failed to decode command ID: %v, error: %v", batchCommandId, err)
		return err
	}
	e.CommandBatchID = commandBatch
	return nil
}

type ContractCallWithTokenSubmitted struct {
	MessageID        string `json:"messageId"`
	Sender           string `json:"sender"`
	SourceChain      string `json:"sourceChain"`
	DestinationChain string `json:"destinationChain"`
	ContractAddress  string `json:"contractAddress"`
	Payload          string `json:"payload"`
	PayloadHash      string `json:"payloadHash"`
	Symbol           string `json:"symbol"`
	Amount           string `json:"amount"`
}

// ExecuteRequest represents an execute request
type ExecuteRequest struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

type EVMEventCompleted = types.ChainEventCompleted

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

var (
	TokenSentEvent = ListenerEvent[*types.EventTokenSent]{
		TopicId: TokenSentEventTopicId,
		Type:    EventTypeTokenSent,
		Parser:  ParseIBCEvent[*types.EventTokenSent],
		//Parser:  ParseTokenSentEvent,
	}
	MintCommandEvent = ListenerEvent[*types.MintCommand]{
		TopicId: MintCommandEventTopicId,
		Type:    EventTypeMintCommand,
		Parser:  ParseIBCEvent[*types.MintCommand],
		//Parser:  ParseTokenSentEvent,
	}
	ContractCallWithMintApprovedEvent = ListenerEvent[*types.EventContractCallWithMintApproved]{
		TopicId: ContractCallWithMintApprovedEventTopicId,
		Type:    EventTypeContractCallWithMintApproved,
		Parser:  ParseIBCEvent[*types.EventContractCallWithMintApproved],
		//Parser:  ParseTokenSentEvent,
	}
	ContractCallApprovedEvent = ListenerEvent[*types.ContractCallApproved]{
		TopicId: ContractCallApprovedEventTopicId,
		Type:    EventTypeContractCallApproved,
		Parser:  ParseIBCEvent[*types.ContractCallApproved],
		//Parser:  ParseContractCallApprovedEvent,
	}
	BatchCommandSignedEvent = ListenerEvent[*types.CommandBatchSigned]{
		TopicId: CommandBatchSignedEventTopicId,
		Type:    EventTypeCommandBatchSigned,
		Parser:  ParseIBCEvent[*types.CommandBatchSigned],
	}

	EVMCompletedEvent = ListenerEvent[*types.ChainEventCompleted]{
		TopicId: EVMCompletedEventTopicId,
		Type:    EventTypeEVMEventCompleted,
		Parser:  ParseIBCEvent[*types.ChainEventCompleted],
	}
	AllNewBlockEvent = ListenerEvent[ScalarMessage]{
		TopicId: "tm.event='NewBlock'",
		Type:    "All",
		Parser:  ParseIBCEvent[ScalarMessage],
		//Parser: ParseAllNewBlockEvent,
	}
	//For future use
	// ContractCallSubmittedEvent = ListenerEvent[IBCEvent[ContractCallSubmitted]]{
	// 	TopicId: ContractCallSubmittedEventTopicId,
	// 	Type:    EventTypeContractCallSubmitted,
	// 	Parser:  ParseIBCEvent[types.ContractCallSubmitted],
	// 	// Parser:  ParseContractCallSubmittedEvent,
	// }
	// ContractCallWithTokenSubmittedEvent = ListenerEvent[IBCEvent[ContractCallWithTokenSubmitted]]{
	// 	TopicId: ContractCallWithTokenEventTopicId,
	// 	Type:    EventTypeContractCallWithTokenSubmitted,
	// 	Parser:  ParseContractCallWithTokenSubmittedEvent,
	// }
	// ExecuteMessageEvent = ListenerEvent[IBCPacketEvent]{
	// 	TopicId: ExecuteMessageEventTopicId,
	// 	Type:    "ExecuteMessage",
	// 	Parser:  ParseExecuteMessageEvent,
	// }
)

type BatchedCommand struct {
	BatchCommandId string
	Chain          string
	CommandIDs     []string
}
type PendingCommands struct {
	//Store SignCommandRequest txHash: chain => txHash
	//There case where multiple chains have the same txHash, so we need to store the txHash for each chain
	SignRequestTxsMutex sync.Mutex
	SignRequestTxs      sync.Map
	//Store psbt for pooling model
	PsbtsMutex sync.Mutex
	Psbts      sync.Map
	//If a chain have pending commands, we need to store number of command in the map
	// UpcPendingCommands      sync.Map
	// UpcPendingCommandsMutex sync.Mutex
	//Store batch commands
	BatchCommandsMutex sync.Mutex
	BatchCommands      sync.Map
}

func NewPendingCommands() *PendingCommands {
	return &PendingCommands{
		SignRequestTxs:      sync.Map{},
		SignRequestTxsMutex: sync.Mutex{},
		BatchCommands:       sync.Map{},
		BatchCommandsMutex:  sync.Mutex{},
		Psbts:               sync.Map{},
		PsbtsMutex:          sync.Mutex{},
	}
}

//	func (p *PendingCommands) LoadSignRequest(chain string) (value any, ok bool) {
//		p.SignRequestTxsMutex.Lock()
//		defer p.SignRequestTxsMutex.Unlock()
//		return p.SignRequestTxs.Load(chain)
//	}
func (p *PendingCommands) StoreSignRequest(chain string, txHash string) {
	p.SignRequestTxsMutex.Lock()
	defer p.SignRequestTxsMutex.Unlock()
	p.SignRequestTxs.Store(chain, txHash)
}

func (p *PendingCommands) DeleteSignRequestTx(txHash string) {
	p.SignRequestTxsMutex.Lock()
	defer p.SignRequestTxsMutex.Unlock()
	chains := []string{}
	p.SignRequestTxs.Range(func(key, value any) bool {
		if value.(string) == txHash {
			chains = append(chains, key.(string))
		}
		return true
	})
	for _, chain := range chains {
		p.SignRequestTxs.Delete(chain)
	}
}

// SignRequestTxs stores map chain=>txHash
// Return map of txHash => []chain
func (p *PendingCommands) GetAllSignRequestTxs() map[string][]string {
	p.SignRequestTxsMutex.Lock()
	defer p.SignRequestTxsMutex.Unlock()
	requests := make(map[string][]string)
	p.SignRequestTxs.Range(func(key, value any) bool {
		txHash := value.(string)
		chain := key.(string)
		requests[txHash] = append(requests[txHash], chain)
		return true
	})
	return requests
}
func (p *PendingCommands) StoreBatchCommand(batchCommandId string, chain string) {
	p.BatchCommandsMutex.Lock()
	defer p.BatchCommandsMutex.Unlock()
	p.BatchCommands.Store(batchCommandId, chain)
}
func (p *PendingCommands) DeleteBatchCommand(batchCommandId string) {
	p.BatchCommandsMutex.Lock()
	defer p.BatchCommandsMutex.Unlock()
	p.BatchCommands.Delete(batchCommandId)
}
func (p *PendingCommands) GetAlllBatchCommands() map[string]string {
	p.BatchCommandsMutex.Lock()
	defer p.BatchCommandsMutex.Unlock()
	requests := make(map[string]string)
	p.BatchCommands.Range(func(key, value any) bool {
		requests[key.(string)] = value.(string)
		return true
	})
	return requests
}

func (p *PendingCommands) StorePsbt(chain string, psbt covExported.Psbt) {
	p.PsbtsMutex.Lock()
	defer p.PsbtsMutex.Unlock()
	pendingPsbt, ok := p.Psbts.Load(chain)
	if ok {
		newPsbts := append(pendingPsbt.([]covExported.Psbt), psbt)
		p.Psbts.Store(chain, newPsbts)
	} else {
		p.Psbts.Store(chain, []covExported.Psbt{psbt})
	}
}

func (p *PendingCommands) StorePsbts(chain string, psbts []covExported.Psbt) {
	p.PsbtsMutex.Lock()
	defer p.PsbtsMutex.Unlock()
	pendingPsbt, ok := p.Psbts.Load(chain)
	if ok {
		newPsbts := append(pendingPsbt.([]covExported.Psbt), psbts...)
		p.Psbts.Store(chain, newPsbts)
	} else {
		p.Psbts.Store(chain, psbts)
	}
}

func (p *PendingCommands) RemovePsbt(chain string, psbt covExported.Psbt) {
	p.PsbtsMutex.Lock()
	defer p.PsbtsMutex.Unlock()
	if value, ok := p.Psbts.Load(chain); ok && value != nil {
		psbts := value.([]covExported.Psbt)
		if len(psbts) > 0 && bytes.Equal(psbts[0], psbt) {
			p.Psbts.Store(chain, psbts[1:])
		}
	}
}

func (p *PendingCommands) DeleteFirstPsbt(chain string) {
	p.PsbtsMutex.Lock()
	defer p.PsbtsMutex.Unlock()
	if value, ok := p.Psbts.Load(chain); ok && value != nil {
		psbts := value.([]covExported.Psbt)
		if len(psbts) > 0 {
			p.Psbts.Store(chain, psbts[1:])
		}
	}
}

// Get first psbt for each chain
func (p *PendingCommands) GetFirstPsbts() map[string]covExported.Psbt {
	p.PsbtsMutex.Lock()
	defer p.PsbtsMutex.Unlock()
	psbts := make(map[string]covExported.Psbt)
	p.Psbts.Range(func(key, value any) bool {
		psbtList := value.([]covExported.Psbt)
		if len(psbtList) > 0 {
			psbts[key.(string)] = psbtList[0]
		}
		return true
	})
	return psbts
}

// Check if a chain has pending commands either psbt or upc
// func (p *PendingCommands) HasPendingCommands(chain string) bool {
// 	p.UpcPendingCommandsMutex.Lock()
// 	defer p.UpcPendingCommandsMutex.Unlock()
// 	count, ok := p.UpcPendingCommands.Load(chain)
// 	if ok && count.(int) > 0 {
// 		log.Debug().Str("Chain", chain).Msg("[PendingCommands] There is a pending upc command")
// 		return true
// 	}
// 	p.PsbtsMutex.Lock()
// 	defer p.PsbtsMutex.Unlock()
// 	value, ok := p.Psbts.Load(chain)
// 	if !ok {
// 		return false
// 	}
// 	psbts := value.([]covtypes.Psbt)
// 	if len(psbts) > 0 {
// 		log.Debug().Str("Chain", chain).Msg("[PendingCommands] There is a pending psbt command")
// 		return true
// 	}
// 	return false
// }

// func (p *PendingCommands) GetUpcPendingCommands() map[string]int {
// 	p.UpcPendingCommandsMutex.Lock()
// 	defer p.UpcPendingCommandsMutex.Unlock()
// 	count := make(map[string]int)
// 	p.UpcPendingCommands.Range(func(key, value any) bool {
// 		count[key.(string)] = value.(int)
// 		return true
// 	})
// 	return count
// }

// func (p *PendingCommands) StoreUpcPendingCommands(chain string, count int) {
// 	p.UpcPendingCommandsMutex.Lock()
// 	defer p.UpcPendingCommandsMutex.Unlock()
// 	p.UpcPendingCommands.Store(chain, count)
// }
// func (p *PendingCommands) DeleteUpcPendingCommands(chain string) {
// 	p.UpcPendingCommandsMutex.Lock()
// 	defer p.UpcPendingCommandsMutex.Unlock()
// 	p.UpcPendingCommands.Delete(chain)
// }

type MessageBuffer struct {
	mutex           sync.Mutex
	txBuffers       []sdk.Msg
	failedMsgs      []sdk.Msg
	signCommandReqs sync.Map
}

func NewMessageBuffer() *MessageBuffer {
	return &MessageBuffer{
		txBuffers:       []sdk.Msg{},
		failedMsgs:      []sdk.Msg{},
		signCommandReqs: sync.Map{},
	}
}

func (b *MessageBuffer) AddNewMsg(msgs ...sdk.Msg) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.txBuffers = append(b.txBuffers, msgs...)
}
func (b *MessageBuffer) GetMsgBufferSize() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.txBuffers)
}
func (b *MessageBuffer) AddFailedMsg(msgs ...sdk.Msg) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.failedMsgs = append(b.failedMsgs, msgs...)
}

func (b *MessageBuffer) AddSignCommandReq(chain string, msg sdk.Msg) error {
	_, ok := b.signCommandReqs.Load(chain)
	if ok {
		log.Debug().Msgf("[Broadcaster] [QueueSignCommandReq] signCommandReq %T for chain %s is already in buffer. Skip adding new one", msg, chain)
		return fmt.Errorf("signCommandReq message %T for chain %s is already in buffer", msg, chain)
	} else {
		log.Debug().Msgf("[Broadcaster] [QueueSignCommandReq] enqueue signCommandReq message %T for chain %s", msg, chain)
		b.signCommandReqs.Store(chain, msg)
		return nil
	}
}

func (b *MessageBuffer) RetrieveNewMsgs(batchSize int) []sdk.Msg {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.txBuffers) > batchSize {
		msgs := b.txBuffers[:batchSize]
		b.txBuffers = b.txBuffers[batchSize:]
		return msgs
	} else {
		msgs := b.txBuffers
		b.txBuffers = []sdk.Msg{}
		return msgs
	}
}

func (b *MessageBuffer) RetrieveFailedMsgs() []sdk.Msg {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	msgs := b.failedMsgs
	b.failedMsgs = []sdk.Msg{}
	return msgs
}

func (b *MessageBuffer) RetrieveAllSignCommandReqs() map[string]sdk.Msg {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	requests := make(map[string]sdk.Msg)
	b.signCommandReqs.Range(func(key, value any) bool {
		requests[key.(string)] = value.(sdk.Msg)
		return true
	})
	b.signCommandReqs.Clear()
	return requests
}
