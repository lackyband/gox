package wsmanager

import (
	"sync"
)

type WebSocketMessageHandler func(message []byte)

type SubscriptionCallback struct {
	ID       string
	Callback WebSocketMessageHandler
}

type WebSocketClientIface interface {
	Subscribe(channel, subscriptionType, symbol, interval string, callback WebSocketMessageHandler) error
	Unsubscribe(subID string) error
	Close() error
	// The following methods are required for testing with the pool manager's processMessages.
	MessageChan() interface{}
	Done() interface{}
}

type PooledWebSocket struct {
	client        WebSocketClientIface
	subscriptions map[string]SubscriptionCallback // subscription ID -> callback
	messageRouter *MessageRouter
	lastUsed      int64
	mu            sync.RWMutex
}

// Subscriptions returns all active subscriptions (for testing/demo only).
func (p *PooledWebSocket) Subscriptions() map[string]SubscriptionCallback {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.subscriptions
}

// Client returns the underlying WebSocketClientIface (for testing/demo only).
func (p *PooledWebSocket) Client() WebSocketClientIface {
	return p.client
}

type WebSocketPool struct {
	connections    []*PooledWebSocket
	maxSubsPerConn int
	mu             sync.RWMutex
	clientFactory  func() (WebSocketClientIface, error)
}

// Connections returns all pooled websocket connections (for testing/demo only).
func (p *WebSocketPool) Connections() []*PooledWebSocket {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connections
}


