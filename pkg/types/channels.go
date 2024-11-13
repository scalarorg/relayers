package types

import contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"

type EventChan chan *contracts.IAxelarGatewayContractCallApproved

type ChannelManager struct {
	evmClientChan     EventChan
	evmListenerChan   EventChan
	postgresEventChan EventChan
	mongoEventChan    EventChan
}
