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
	Component string      // Which component should handle this
	Handler   string      // Which method should handle this
	Data      interface{} // The actual event data
}

type FindCosmosToEvmCallContractApproved struct {
	ID      string
	Payload []byte
}

type HandleCosmosToEvmCallContractCompleteEventData struct {
	Event      *EvmEvent[contracts.IAxelarGatewayContractCallApproved]
	RelayDatas []FindCosmosToEvmCallContractApproved
}

type HandleCosmosToEvmCallContractCompleteEventExecuteResult struct {
	ID     string
	Status Status
}
