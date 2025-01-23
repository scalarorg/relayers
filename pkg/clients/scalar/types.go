package scalar

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Add this new type definition

const (
	EventTypeMintCommand                     = "scalar.chains.v1beta1.MintCommand"
	EventTypeContractCallApproved            = "scalar.chains.v1beta1.ContractCallApproved"
	EventTypeContractCallWithMintApproved    = "scalar.chains.v1beta1.ContractCallWithMintApproved"
	EventTypeTokenSent                       = "scalar.chains.v1beta1.EventTokenSent"
	EventTypeEVMEventCompleted               = "scalar.chains.v1beta1.EVMEventCompleted"
	EventTypeCommandBatchSigned              = "scalar.chains.v1beta1.CommandBatchSigned"
	EventTypeContractCallSubmitted           = "scalar.scalarnet.v1beta1.ContractCallSubmitted"
	EventTypeContractCallWithTokenSubmitted  = "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted"
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
	eventId := strings.TrimPrefix(removeQuote(jsonData["event_id"]), "0x")
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
	eventId := strings.TrimPrefix(removeQuote(jsonData["event_id"]), "0x")
	e.EventID = types.EventID(eventId)
	e.Type = removeQuote(jsonData["type"])
	return nil
}

func UnmarshalContractCallApproved(jsonData map[string]string, e *types.ContractCallApproved) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	eventId := strings.TrimPrefix(removeQuote(jsonData["event_id"]), "0x")
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
	e.PayloadHash = types.Hash(common.HexToHash(payloadHex))
	log.Debug().Any("JsonData", jsonData).Msg("Input data")
	log.Debug().Any("result", e).Msg("Resut data")
	return nil
}

func UnmarshalContractCallWithMintApproved(jsonData map[string]string, e *types.EventContractCallWithMintApproved) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	eventId := strings.TrimPrefix(removeQuote(jsonData["event_id"]), "0x")
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
	e.PayloadHash = types.Hash(common.HexToHash(payloadHex))
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
