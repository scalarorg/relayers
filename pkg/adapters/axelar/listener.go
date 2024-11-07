package axelar

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/scalarorg/relayers/config"
	"github.com/scalarorg/relayers/pkg/types"
)

type AxelarListener struct {
	wsURL      string
	wsMap      map[string]*websocket.Conn
	wsMapMux   sync.Mutex
	maxRetries int
}

func NewAxelarListener() (*AxelarListener, error) {
	if config.GlobalConfig.Axelar.RPCUrl == "" {
		return nil, fmt.Errorf("axelar RPC URL is not configured")
	}

	return &AxelarListener{
		wsURL:      config.GlobalConfig.Axelar.RPCUrl,
		wsMap:      make(map[string]*websocket.Conn),
		maxRetries: -1, // infinite retries, like TypeScript version
	}, nil
}

func (al *AxelarListener) reconnect(topicID string) error {
	al.wsMapMux.Lock()
	defer al.wsMapMux.Unlock()

	// Close existing connection if any
	if ws, exists := al.wsMap[topicID]; exists {
		ws.Close()
		delete(al.wsMap, topicID)
	}

	// Create new connection
	ws, _, err := websocket.DefaultDialer.Dial(al.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to reconnect WebSocket: %w", err)
	}

	al.wsMap[topicID] = ws
	return nil
}

func (al *AxelarListener) initWS(topicID string) (*websocket.Conn, error) {
	al.wsMapMux.Lock()
	defer al.wsMapMux.Unlock()

	if ws, ok := al.wsMap[topicID]; ok {
		return ws, nil
	}

	ws, _, err := websocket.DefaultDialer.Dial(al.wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	al.wsMap[topicID] = ws
	return ws, nil
}

func (al *AxelarListener) Listen(ctx context.Context, event types.AxelarListenerEvent[any], ch chan<- interface{}) error {
	ws, err := al.initWS(event.TopicID)
	if err != nil {
		return err
	}

	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "subscribe",
		"params":  []string{event.TopicID},
	}

	err = ws.WriteJSON(subscribeMsg)
	if err != nil {
		return fmt.Errorf("failed to subscribe to event: %w", err)
	}

	log.Info().Str("type", event.Type).Msg("[AxelarListener] Listening to Axelar event")

	go func() {
		defer ws.Close()
		for {
			select {
			case <-ctx.Done():
				log.Debug().Str("type", event.Type).Msg("[AxelarListener] Stopping Axelar listener")
				return
			default:
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Error().Err(err).Str("type", event.Type).Msg("[AxelarListener] Error reading WebSocket message")
					return
				}

				var wsEvent struct {
					Result struct {
						Query  string              `json:"query"`
						Events map[string][]string `json:"events"`
					} `json:"result"`
				}

				err = json.Unmarshal(message, &wsEvent)
				if err != nil {
					log.Error().Err(err).Str("type", event.Type).Msg("[AxelarListener] Error unmarshalling WebSocket message")
					continue
				}

				if wsEvent.Result.Query != event.TopicID {
					log.Debug().Str("type", event.Type).Str("query", wsEvent.Result.Query).Msg("[AxelarListener] Unmatched message")
					continue
				}

				parsedEvent, err := event.ParseEvent(wsEvent.Result.Events)
				if err != nil {
					log.Error().Err(err).Str("type", event.Type).Msg("[AxelarListener] Error parsing event")
					continue
				}

				ch <- parsedEvent
			}
		}
	}()

	return nil
}
