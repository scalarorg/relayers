package events

import (
	"github.com/rs/zerolog/log"
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
	return &EventBus{
		channels: make(map[string]Channels),
	}
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
	log.Debug().Msgf("Broadcasting event to %d listeners for %s", len(channels), event.DestinationChain)
	for _, channel := range channels {
		channel <- event
	}
}

func (eb *EventBus) Subscribe(destinationChain string) <-chan *EventEnvelope {
	log.Debug().Msgf("Subscribing to %s", destinationChain)
	sender := make(chan *EventEnvelope)
	if eb.channels[destinationChain] == nil {
		eb.channels[destinationChain] = []chan<- *EventEnvelope{sender}
	} else {
		eb.channels[destinationChain] = append(eb.channels[destinationChain], sender)
	}
	return sender
}
