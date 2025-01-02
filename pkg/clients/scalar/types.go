package scalar

import (
	"github.com/gogo/protobuf/proto"
	"github.com/scalarorg/scalar-core/x/chains/types"
)

// Add this new type definition

const (
	EventTypeDestCallApproved               = "scalar.chains.v1beta1.DestCallApproved"
	EventTypeEVMEventCompleted              = "scalar.chains.v1beta1.EVMEventCompleted"
	EventTypeTokenSent                      = "scalar.scalarnet.v1beta1.TokenSent"
	EventTypeContractCallSubmitted          = "scalar.scalarnet.v1beta1.ContractCallSubmitted"
	EventTypeContractCallWithTokenSubmitted = "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted"
	TokenSentEventTopicId                   = "tm.event='NewBlock' AND scalar.scalarnet.v1beta1.TokenSent.event_id EXISTS"
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
type TokenSent struct {
}
type DestCallApproved struct {
	MessageID        string `json:"messageId"`
	Sender           string `json:"sender"`
	SourceChain      string `json:"sourceChain"`
	DestinationChain string `json:"destinationChain"`
	ContractAddress  string `json:"contractAddress"`
	CommandID        string `json:"commandId"`
	Payload          string `json:"payload"`
	PayloadHash      string `json:"payloadHash"`
}

type ContractCallSubmitted struct {
	MessageID        string `json:"messageId"`
	Sender           string `json:"sender"`
	SourceChain      string `json:"sourceChain"`
	DestinationChain string `json:"destinationChain"`
	ContractAddress  string `json:"contractAddress"`
	CommandID        string `json:"commandId"`
	Payload          string `json:"payload"`
	PayloadHash      string `json:"payloadHash"`
}

type ContractCallApproved = ContractCallSubmitted

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

type EVMEventCompleted = ExecuteRequest

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
	DestCallApprovedEvent = ListenerEvent[*types.DestCallApproved]{
		TopicId: DestCallApprovedEventTopicId,
		Type:    EventTypeDestCallApproved,
		Parser:  ParseDestCallApprovedEvent,
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
		// Parser:  ParseEvmEventCompletedEvent,
	}
	AllNewBlockEvent = ListenerEvent[proto.Message]{
		TopicId: "tm.event='NewBlock'",
		Type:    "All",
		Parser:  ParseIBCEvent[proto.Message],
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
