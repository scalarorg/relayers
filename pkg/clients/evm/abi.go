package evm

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rs/zerolog/log"
	contracts "github.com/scalarorg/relayers/pkg/clients/evm/contracts/generated"
)

var (
	scalarGatewayAbi *abi.ABI
	mapEvents        = map[string]*abi.Event{}
)

func init() {
	log.Info().Msg("Initializing ABI")
	scalarGatewayAbi, _ = contracts.IScalarGatewayMetaData.GetAbi()
	for _, event := range scalarGatewayAbi.Events {
		mapEvents[event.Name] = &event
	}
}

func getScalarGatewayAbi() (*abi.ABI, error) {
	if scalarGatewayAbi == nil {
		var err error
		scalarGatewayAbi, err = contracts.IScalarGatewayMetaData.GetAbi()
		if err != nil {
			return nil, err
		}
	}
	return scalarGatewayAbi, nil
}

func GetMapEvents() map[string]*abi.Event {
	return mapEvents
}

func GetEventByName(name string) (*abi.Event, bool) {
	event, ok := mapEvents[name]
	return event, ok
}
