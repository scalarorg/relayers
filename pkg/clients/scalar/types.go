package scalar

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/scalar-core/x/chains/types"
	"github.com/scalarorg/scalar-core/x/nexus/exported"
)

// Add this new type definition

const (
	EventTypeDestCallApproved               = "scalar.chains.v1beta1.DestCallApproved"
	EventTypeEVMEventCompleted              = "scalar.chains.v1beta1.EVMEventCompleted"
	EventTypeTokenSent                      = "scalar.chains.v1beta1.EventTokenSent"
	EventTypeMintCommand                    = "scalar.chains.v1beta1.MintCommand"
	EventTypeContractCallSubmitted          = "scalar.scalarnet.v1beta1.ContractCallSubmitted"
	EventTypeContractCallWithTokenSubmitted = "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted"
	TokenSentEventTopicId                   = "tm.event='NewBlock' AND scalar.chains.v1beta1.EventTokenSent.event_id EXISTS"
	MintCommandEventTopicId                 = "tm.event='NewBlock' AND scalar.chains.v1beta1.MintCommand.event_id EXISTS"
	DestCallApprovedEventTopicId            = "tm.event='NewBlock' AND scalar.chains.v1beta1.DestCallApproved.event_id EXISTS"
	SignCommandsEventTopicId                = "tm.event='NewBlock' AND sign.batchedCommandID EXISTS"
	EVMCompletedEventTopicId                = "tm.event='NewBlock' AND scalar.chains.v1beta1.EVMEventCompleted.event_id EXISTS"
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
	case *types.DestCallApproved:
		return UnmarshalDestCallApproved(jsonData, e)
	case *types.ChainEventCompleted:
		return UnmarshalChainEventCompleted(jsonData, e)
	default:
		return fmt.Errorf("unsupport type %T", e)
	}
}
func UnmarshalTokenSent(jsonData map[string]string, e *types.EventTokenSent) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	e.Sender = removeQuote(jsonData["sender"])
	e.DestinationAddress = removeQuote(jsonData["destination_address"])
	e.DestinationChain = exported.ChainName(removeQuote(jsonData["destination_chain"]))
	e.EventID = types.EventID(removeQuote(jsonData["event_id"]))
	transferId, ok := sdk.NewIntFromString(jsonData["transfer_id"])
	if ok {
		e.TransferID = exported.TransferID(transferId.Uint64())
	}
	assetData, ok := jsonData["asset"]
	if ok {
		var rawCoin map[string]string
		fmt.Printf("assetData %s\n", assetData)
		err := json.Unmarshal([]byte(assetData), &rawCoin)
		if err != nil {
			log.Debug().Err(err).Msg("Cannot unmarshalling coin data")
		} else {
			denom := rawCoin["denom"]
			amount, ok := sdk.NewIntFromString(rawCoin["amount"])
			if ok {
				e.Asset = sdk.NewCoin(denom, amount)
			}
		}
	}
	return nil
}

// {"asset":"{\"denom\":\"pBtc\",\"amount\":\"10000\"}","chain":"\"evm|11155111\"","destination_address":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"","destination_chain":"\"evm|97\"","event_id":"\"0x620bc60a616248eaf0a9f5b7e45db3f96eca31420c581034a6c59669cefb7de1-240\"","sender":"\"0x982321eb5693cdbAadFfe97056BEce07D09Ba49f\"","transfer_id":"\"0\""}

func UnmarshalChainEventCompleted(jsonData map[string]string, e *types.ChainEventCompleted) error {
	e.Chain = exported.ChainName(jsonData["chain"])
	e.EventID = types.EventID(jsonData["event_id"])
	e.Type = jsonData["type"]
	return nil
}

func UnmarshalDestCallApproved(jsonData map[string]string, e *types.DestCallApproved) error {
	e.Chain = exported.ChainName(removeQuote(jsonData["chain"]))
	e.EventID = types.EventID(removeQuote(jsonData["event_id"]))
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
	log.Debug().Any("JsonData", jsonData).Msg("Input data")
	log.Debug().Any("result", e).Msg("Resut data")
	return nil
}

// type DestCallApproved struct {
// 	MessageID        string `json:"messageId"`
// 	Sender           string `json:"sender"`
// 	SourceChain      string `json:"sourceChain"`
// 	DestinationChain string `json:"destinationChain"`
// 	ContractAddress  string `json:"contractAddress"`
// 	CommandID        string `json:"commandId"`
// 	Payload          string `json:"payload"`
// 	PayloadHash      string `json:"payloadHash"`
// }

// type ContractCallSubmitted struct {
// 	MessageID        string `json:"messageId"`
// 	Sender           string `json:"sender"`
// 	SourceChain      string `json:"sourceChain"`
// 	DestinationChain string `json:"destinationChain"`
// 	ContractAddress  string `json:"contractAddress"`
// 	CommandID        string `json:"commandId"`
// 	Payload          string `json:"payload"`
// 	PayloadHash      string `json:"payloadHash"`
// }

// type ContractCallApproved = ContractCallSubmitted

type SignCommands struct {
	DestinationChain string `json:"destinationChain"`
	TxHash           string `json:"txHash"`
	MessageID        string `json:"messageId"`
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
	DestCallApprovedEvent = ListenerEvent[*types.DestCallApproved]{
		TopicId: DestCallApprovedEventTopicId,
		Type:    EventTypeDestCallApproved,
		Parser:  ParseIBCEvent[*types.DestCallApproved],
		//Parser:  ParseDestCallApprovedEvent,
	}
	SignCommandsEvent = ListenerEvent[SignCommands]{
		TopicId: SignCommandsEventTopicId,
		Type:    "sign",
		Parser:  ParseSignCommandsEvent,
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

//For example
