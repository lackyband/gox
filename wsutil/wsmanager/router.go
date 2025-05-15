package wsmanager

import (
	"encoding/json"
	"sync"
)

type MessageRouter struct {
	// callbacks[channel][subscriptionType] = callback
	callbacks map[string]map[string]WebSocketMessageHandler
	mu        sync.RWMutex
}

func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		callbacks: make(map[string]map[string]WebSocketMessageHandler),
	}
}

// subscriptionType could be "trade", "depth", "account", etc.
func (r *MessageRouter) RegisterCallback(channel, subscriptionType string, callback WebSocketMessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.callbacks[channel]; !exists {
		r.callbacks[channel] = make(map[string]WebSocketMessageHandler)
	}
	r.callbacks[channel][subscriptionType] = callback
}

func (r *MessageRouter) RouteMessage(message []byte) {
	var msg struct {
		Channel string `json:"channel"`
		Event   string `json:"event"`
		Type    string `json:"type"` // e.g., "trade", "depth", etc.
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if typeMap, exists := r.callbacks[msg.Channel]; exists {
		// Try event first, then type, then fallback (for compatibility)
		if cb, ok := typeMap[msg.Event]; ok {
			go cb(message)
			return
		}
		if cb, ok := typeMap[msg.Type]; ok {
			go cb(message)
			return
		}
		if cb, ok := typeMap[""]; ok {
			go cb(message)
		}
	}
	// Optionally handle pings, etc.
}
