package events

type Config struct {
}

var eventBus *EventBus

type EventBus struct {
}

// Todo: Add some event bus config
// For example:
// - Event buffer size
// - Event receiver buffer size
func NewEventBus(config *Config) *EventBus {
	return &EventBus{}
}

func GetEventBus(config *Config) *EventBus {
	if eventBus == nil {
		eventBus = NewEventBus(config)
	}
	return eventBus
}
