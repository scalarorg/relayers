package types

import (
	contracts "github.com/scalarorg/relayers/pkg/contracts/generated"
)

type Status int

const (
	PENDING Status = iota
	APPROVED
	SUCCESS
	FAILED
)

type EventEnvelope struct {
	DestinationChain string      // The source chain of the event
	EventType        string      // The name of the event in format "ComponentName.EventName"
	Data             interface{} // The actual event data
}

type FindCosmosToEvmCallContractApproved struct {
	ID      string
	Payload []byte
}

type HandleCosmosToEvmCallContractCompleteEventData struct {
	Event      *EvmEvent[*contracts.IAxelarGatewayContractCallApproved]
	RelayDatas []FindCosmosToEvmCallContractApproved
}

type HandleCosmosToEvmCallContractCompleteEventExecuteResult struct {
	ID     string
	Status Status
}

type WaitForTransactionData struct {
	Hash  string
	Event *EvmEvent[*contracts.IAxelarGatewayContractCall]
}
