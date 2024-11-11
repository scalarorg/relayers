package scalar

import "github.com/scalarorg/relayers/pkg/types"

// Add this new type definition

const (
	EVMCompletedEventTopicId          = "tm.event='NewBlock' AND axelar.evm.v1beta1.EVMEventCompleted.event_id EXISTS"
	ContractCallEventTopicId          = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallSubmitted.message_id EXISTS"
	ContractCallApprovedEventTopicId  = "tm.event='NewBlock' AND axelar.evm.v1beta1.ContractCallApproved.event_id EXISTS"
	ContractCallWithTokenEventTopicId = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted.message_id EXISTS"
	IBCCompleteEventTopicId           = "tm.event='Tx' AND message.action='ExecuteMessage'"
)

type ListenerEvent[T any] struct {
	TopicId string
	Type    string
	Parser  func(events map[string][]string) (T, error)
}

var (
	ContractCallApprovedEvent = ListenerEvent[*types.IBCEvent[types.ContractCallApproved]]{
		TopicId: ContractCallApprovedEventTopicId,
		Type:    "axelar.evm.v1beta1.ContractCallApproved",
		Parser:  ParseContractCallApprovedEvent,
	}
	EVMCompletedEvent = ListenerEvent[*types.ExecuteRequest]{
		TopicId: EVMCompletedEventTopicId,
		Type:    "axelar.evm.v1beta1.EVMEventCompleted",
		Parser:  ParseEvmEventCompletedEvent,
	}

	//For future use
	ContractCallSubmittedEvent = ListenerEvent[*types.IBCEvent[types.ContractCallSubmitted]]{
		TopicId: ContractCallEventTopicId,
		Type:    "axelar.axelarnet.v1beta1.ContractCallSubmitted",
		Parser:  ParseContractCallSubmittedEvent,
	}
	ContractCallWithTokenSubmittedEvent = ListenerEvent[*types.IBCEvent[types.ContractCallWithTokenSubmitted]]{
		TopicId: ContractCallWithTokenEventTopicId,
		Type:    "axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted",
		Parser:  ParseContractCallWithTokenSubmittedEvent,
	}
	ExecuteMessageEvent = ListenerEvent[*types.IBCPacketEvent]{
		TopicId: IBCCompleteEventTopicId,
		Type:    "ExecuteMessage",
		Parser:  ParseIBCCompleteEvent,
	}
)

//For example
