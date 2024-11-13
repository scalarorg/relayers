package events

import (
	"github.com/scalarorg/relayers/config"
)

var eventBus *EventBus

type Channels []chan<- *EventEnvelope

// Store array of channels by destination chain
type EventBus struct {
	channels map[string]Channels
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
func (eb *EventBus) filterChannels(destinationChain string) Channels {
	return eb.channels[destinationChain]
}

func (eb *EventBus) BroadcastEvent(event *EventEnvelope) {
	channels := eb.filterChannels(event.DestinationChain)
	for _, channel := range channels {
		channel <- event
	}
}

func (eb *EventBus) Subscribe(destinationChain string) <-chan *EventEnvelope {
	sender := make(chan *EventEnvelope)
	channels := eb.channels[destinationChain]
	channels = append(channels, sender)
	return sender
}
