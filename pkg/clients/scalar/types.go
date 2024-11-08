package scalar

// Add this new type definition

const (
	EVMCompletedEventTopicId          = "tm.event='NewBlock' AND axelar.evm.v1beta1.EVMEventCompleted.event_id EXISTS"
	ContractCallEventTopicId          = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallSubmitted.message_id EXISTS"
	ContractCallApprovedEventTopicId  = "tm.event='NewBlock' AND axelar.evm.v1beta1.ContractCallApproved.event_id EXISTS"
	ContractCallWithTokenEventTopicId = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted.message_id EXISTS"
	IBCCompleteEventTopicId           = "tm.event='Tx' AND message.action='ExecuteMessage'"
)

type Config struct {
	ChainID       string
	Denom         string
	RpcEndpoint   string
	GasPrice      string
	LcdEndpoint   string
	WsEndpoint    string
	Mnemonic      string
	BroadcastMode string //"sync" or "async" or "block"
}

type Event struct {
	TopicId string
	Type    string
}

var (
	ContractCallApprovedEvent = Event{
		TopicId: ContractCallApprovedEventTopicId,
		Type:    "axelar.evm.v1beta1.ContractCallApproved",
	}
	EVMCompletedEvent = Event{
		TopicId: EVMCompletedEventTopicId,
		Type:    "axelar.evm.v1beta1.EVMEventCompleted",
	}
)

//For example
