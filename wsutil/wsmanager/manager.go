package wsmanager

import (
	"sync"
)

type WebSocketPoolManager struct {
	maxSubscriptionsPerConnection int
	pools                         map[string]*WebSocketPool
	clientFactory                 func() (WebSocketClientIface, error)
	mu                            sync.RWMutex
}

func NewWebSocketPoolManager(maxSubsPerConn int, clientFactory func() (WebSocketClientIface, error)) *WebSocketPoolManager {
	return &WebSocketPoolManager{
		maxSubscriptionsPerConnection: maxSubsPerConn,
		pools:                         make(map[string]*WebSocketPool),
		clientFactory:                 clientFactory,
	}
}

// subscriptionType is required for correct callback grouping
func (m *WebSocketPoolManager) Subscribe(channel, subscriptionType, symbol, interval string, callback WebSocketMessageHandler) (string, error) {

	m.mu.Lock()
	pool, exists := m.pools[channel]
	if !exists {
		pool = NewWebSocketPool(m.maxSubscriptionsPerConnection, m.clientFactory)
		m.pools[channel] = pool
	}
	m.mu.Unlock()

	return pool.Subscribe(channel, subscriptionType, symbol, interval, callback)
}

func (m *WebSocketPoolManager) Unsubscribe(channel, subID string) error {
	m.mu.RLock()
	pool, exists := m.pools[channel]
	m.mu.RUnlock()
	if !exists {
		return nil
	}
	return pool.Unsubscribe(subID)
}

// Pools exposes the internal pools map for testing/demo purposes ONLY.
func (m *WebSocketPoolManager) Pools() map[string]*WebSocketPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pools
}

func (m *WebSocketPoolManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pool := range m.pools {
		pool.Cleanup()
	}
}

func NewWebSocketPool(maxSubsPerConn int, clientFactory func() (WebSocketClientIface, error)) *WebSocketPool {
	return &WebSocketPool{
		connections:    []*PooledWebSocket{},
		maxSubsPerConn: maxSubsPerConn,
		clientFactory:  clientFactory,
	}
}

func generateSubscriptionID(channel, symbol, interval string) string {
	return channel + ":" + symbol + ":" + interval
}
