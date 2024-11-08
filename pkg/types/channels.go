package types

import contracts "github.com/scalarorg/relayers/pkg/contracts/generated"

type EventChan chan *contracts.IAxelarGatewayContractCallApproved

type ChannelManager struct {
	evmClientChan     EventChan
	evmListenerChan   EventChan
	postgresEventChan EventChan
	mongoEventChan    EventChan
}
