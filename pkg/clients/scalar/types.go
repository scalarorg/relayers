package scalar

// Add this new type definition

const (
	SCALAR_NETWORK_NAME           = "Scalar.Network"
	SCALAR_CONTRACT_CALL_APPROVED = "Scalar.ContractCallApproved"
	SCALAR_COMMAND_EXECUTED       = "Scalar.CommandExecuted"

	ContractCallApprovedEventTopicId = "tm.event='NewBlock' AND scalar.evm.v1beta1.ContractCallApproved.event_id EXISTS"
	SignCommandsEventTopicId         = "tm.event='NewBlock' AND sign.batchedCommandID EXISTS"
	EVMCompletedEventTopicId         = "tm.event='NewBlock' AND scalar.evm.v1beta1.EVMEventCompleted.event_id EXISTS"
	//For future use
	ContractCallSubmittedEventTopicId = "tm.event='Tx' AND scalar.scalarnet.v1beta1.ContractCallSubmitted.message_id EXISTS"
	ContractCallWithTokenEventTopicId = "tm.event='Tx' AND scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted.message_id EXISTS"
	ExecuteMessageEventTopicId        = "tm.event='Tx' AND message.action='ExecuteMessage'"
)

type EventHandlerCallBack[T any] func(events []T)
type ListenerEvent[T any] struct {
	TopicId string
	Type    string
	Parser  func(events map[string][]string) ([]T, error)
}

type IBCEvent[T any] struct {
	Hash        string `json:"hash"`
	SrcChannel  string `json:"srcChannel,omitempty"`
	DestChannel string `json:"destChannel,omitempty"`
	Args        T      `json:"args"`
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
	DestCallApprovedEvent = ListenerEvent[IBCEvent[DestCallApproved]]{
		TopicId: ContractCallApprovedEventTopicId,
		Type:    "scalar.evm.v1beta1.ContractCallApproved",
		Parser:  ParseDestCallApprovedEvent,
	}
	ContractCallApprovedEvent = ListenerEvent[IBCEvent[ContractCallApproved]]{
		TopicId: ContractCallApprovedEventTopicId,
		Type:    "scalar.evm.v1beta1.ContractCallApproved",
		Parser:  ParseContractCallApprovedEvent,
	}
	SignCommandsEvent = ListenerEvent[IBCEvent[SignCommands]]{
		TopicId: SignCommandsEventTopicId,
		Type:    "sign",
		Parser:  ParseSignCommandsEvent,
	}

	EVMCompletedEvent = ListenerEvent[IBCEvent[EVMEventCompleted]]{
		TopicId: EVMCompletedEventTopicId,
		Type:    "scalar.evm.v1beta1.EVMEventCompleted",
		Parser:  ParseEvmEventCompletedEvent,
	}
	AllNewBlockEvent = ListenerEvent[IBCEvent[any]]{
		TopicId: "tm.event='NewBlock'",
		Type:    "All",
		Parser:  ParseAllNewBlockEvent,
	}
	//For future use
	ContractCallSubmittedEvent = ListenerEvent[IBCEvent[ContractCallSubmitted]]{
		TopicId: ContractCallSubmittedEventTopicId,
		Type:    "scalar.scalarnet.v1beta1.ContractCallSubmitted",
		Parser:  ParseContractCallSubmittedEvent,
	}
	ContractCallWithTokenSubmittedEvent = ListenerEvent[IBCEvent[ContractCallWithTokenSubmitted]]{
		TopicId: ContractCallWithTokenEventTopicId,
		Type:    "scalar.scalarnet.v1beta1.ContractCallWithTokenSubmitted",
		Parser:  ParseContractCallWithTokenSubmittedEvent,
	}
	ExecuteMessageEvent = ListenerEvent[IBCPacketEvent]{
		TopicId: ExecuteMessageEventTopicId,
		Type:    "ExecuteMessage",
		Parser:  ParseExecuteMessageEvent,
	}
)

//For example
