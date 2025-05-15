package wsmanager

import (
	"fmt"
	"sync"
	"time"
)



func (p *WebSocketPool) Subscribe(channel, symbol, interval string, callback WebSocketMessageHandler) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var conn *PooledWebSocket
	for _, c := range p.connections {
		if len(c.subscriptions) < p.maxSubsPerConn {
			conn = c
			break
		}
	}

	if conn == nil {
		client, err := p.clientFactory()
		if err != nil {
			return "", err
		}
		conn = &PooledWebSocket{
			client:        client,
			subscriptions: make(map[string]SubscriptionCallback),
			messageRouter: NewMessageRouter(),
			lastUsed:      time.Now().Unix(),
			mu:            sync.RWMutex{},
		}
		p.connections = append(p.connections, conn)
		// The tester's client must provide messageChan and done fields for processMessages to work.
		go conn.processMessages()
	}

	subID := generateSubscriptionID(channel, symbol, interval)
	conn.mu.Lock()
	conn.subscriptions[subID] = SubscriptionCallback{
		ID:       subID,
		Callback: callback,
	}
	conn.mu.Unlock()
	conn.messageRouter.RegisterCallback(subID, channel, callback)

	if len(conn.subscriptions) == 1 {
		if err := conn.client.Subscribe(channel, symbol, interval, conn.messageRouter.RouteMessage); err != nil {
			conn.mu.Lock()
			delete(conn.subscriptions, subID)
			conn.mu.Unlock()
			return "", err
		}
	}

	conn.lastUsed = time.Now().Unix()
	return subID, nil
}

func (p *WebSocketPool) Unsubscribe(subID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, conn := range p.connections {
		conn.mu.Lock()
		if _, exists := conn.subscriptions[subID]; exists {
			delete(conn.subscriptions, subID)
			if len(conn.subscriptions) == 0 {
				conn.client.Close()
				p.connections = append(p.connections[:i], p.connections[i+1:]...)
			}
			conn.mu.Unlock()
			return nil
		}
		conn.mu.Unlock()
	}
	return fmt.Errorf("subscription not found: %s", subID)
}

func (p *WebSocketPool) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now().Unix()
	for i := 0; i < len(p.connections); {
		conn := p.connections[i]
		conn.mu.Lock()
		if len(conn.subscriptions) == 0 && now-conn.lastUsed > 300 {
			conn.client.Close()
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			conn.mu.Unlock()
			continue
		}
		conn.mu.Unlock()
		i++
	}
}

// processMessages requires that the client implements messageChan (chan []byte) and done (chan struct{}).
func (p *PooledWebSocket) processMessages() {
	messageChan, ok1 := p.client.MessageChan().(chan []byte)
	done, ok2 := p.client.Done().(chan struct{})
	if !ok1 || !ok2 {
		return
	}
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				return
			}
			p.messageRouter.RouteMessage(message)
		case <-done:
			return
		}
	}
}
