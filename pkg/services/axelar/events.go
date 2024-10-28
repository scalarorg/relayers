package axelar

// Define event types
const (
	EvmEventCompletedType              = "axelar.evm.v1beta1.EVMEventCompleted"
	ContractCallSubmittedType          = "axelar.axelarnet.v1beta1.ContractCallSubmitted"
	ContractCallApprovedType           = "axelar.evm.v1beta1.ContractCallApproved"
	ContractCallWithTokenSubmittedType = "axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted"
	ExecuteMessageType                 = "ExecuteMessage"
)

// Define topic IDs
const (
	AxelarEVMCompletedEventTopicID                = "tm.event='NewBlock' AND axelar.evm.v1beta1.EVMEventCompleted.event_id EXISTS"
	AxelarCosmosContractCallEventTopicID          = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallSubmitted.message_id EXISTS"
	AxelarCosmosContractCallApprovedEventTopicID  = "tm.event='NewBlock' AND axelar.evm.v1beta1.ContractCallApproved.event_id EXISTS"
	AxelarCosmosContractCallWithTokenEventTopicID = "tm.event='Tx' AND axelar.axelarnet.v1beta1.ContractCallWithTokenSubmitted.message_id EXISTS"
	AxelarIBCCompleteEventTopicID                 = "tm.event='Tx' AND message.action='ExecuteMessage'"
)

// Define AxelarListenerEvent instances with proper types
var (
	AxelarEVMCompletedEvent = AxelarListenerEvent[*ExecuteRequest]{
		Type:       EvmEventCompletedType,
		TopicID:    AxelarEVMCompletedEventTopicID,
		ParseEvent: ParseEvmEventCompletedEvent,
	}

	AxelarCosmosContractCallEvent = AxelarListenerEvent[*IBCEvent[ContractCallSubmitted]]{
		Type:       ContractCallSubmittedType,
		TopicID:    AxelarCosmosContractCallEventTopicID,
		ParseEvent: ParseContractCallSubmittedEvent,
	}

	AxelarCosmosContractCallApprovedEvent = AxelarListenerEvent[*IBCEvent[ContractCallSubmitted]]{
		Type:       ContractCallApprovedType,
		TopicID:    AxelarCosmosContractCallApprovedEventTopicID,
		ParseEvent: ParseContractCallApprovedEvent,
	}

	AxelarCosmosContractCallWithTokenEvent = AxelarListenerEvent[*IBCEvent[ContractCallWithTokenSubmitted]]{
		Type:       ContractCallWithTokenSubmittedType,
		TopicID:    AxelarCosmosContractCallWithTokenEventTopicID,
		ParseEvent: ParseContractCallWithTokenSubmittedEvent,
	}

	AxelarIBCCompleteEvent = AxelarListenerEvent[*IBCPacketEvent]{
		Type:       ExecuteMessageType,
		TopicID:    AxelarIBCCompleteEventTopicID,
		ParseEvent: ParseIBCCompleteEvent,
	}
)
