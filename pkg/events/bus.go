package events

import (
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/types"
)

var eventBus *EventBus

type EventBus struct {
}

// Todo: Add some event bus config
// For example:
// - Event buffer size
// - Event receiver buffer size
func NewEventBus(config *config.EventBusConfig) *EventBus {
	return &EventBus{}
}

func GetEventBus(config *config.EventBusConfig) *EventBus {
	if eventBus == nil {
		eventBus = NewEventBus(config)
	}
	return eventBus
}

func (eb *EventBus) BroadcastEvent(event *types.EventEnvelope) {

}

func (eb *EventBus) Subscribe(eventType string) (<-chan *types.EventEnvelope, error) {
	sender := make(chan *types.EventEnvelope)
	return sender, nil
}
