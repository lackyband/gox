package wsmanager

import (
	"encoding/json"
	"sync"
)

type MessageRouter struct {
	callbacks    map[string]WebSocketMessageHandler
	channelIndex map[string]map[string]struct{}
	mu           sync.RWMutex
}

func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		callbacks:    make(map[string]WebSocketMessageHandler),
		channelIndex: make(map[string]map[string]struct{}),
	}
}

func (r *MessageRouter) RegisterCallback(subID, channel string, callback WebSocketMessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callbacks[subID] = callback

	if _, exists := r.channelIndex[channel]; !exists {
		r.channelIndex[channel] = make(map[string]struct{})
	}
	r.channelIndex[channel][subID] = struct{}{}
}

func (r *MessageRouter) RouteMessage(message []byte) {
	var msg struct {
		Channel string `json:"channel"`
		Event   string `json:"event"`
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if subs, exists := r.channelIndex[msg.Channel]; exists {
		for subID := range subs {
			if cb, ok := r.callbacks[subID]; ok {
				go cb(message)
			}
		}
	}
	// Optionally handle pings, etc.
}
