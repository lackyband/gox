package tester

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"github.com/gorilla/websocket"
)

// This test runs long enough to exercise ping/pong and keepalive handling on the Bitrue Futures WebSocket.
func TestBitrueFuturesTradePool_Keepalive(t *testing.T) {
	t.Log("=== Starting Bitrue Futures Trade Pool Keepalive Test ===")
	symbols := fetchFuturesContracts(t)
	if len(symbols) > 10 {
		symbols = symbols[:10] // Use a small set for long-duration test
	}
	t.Logf("Testing keepalive for %d symbols", len(symbols))

	var wg sync.WaitGroup
	results := make(chan string, len(symbols))
	const keepaliveDuration = 5 * time.Minute
	timeout := time.After(keepaliveDuration + 30*time.Second)

	for groupID, group := range [][]string{symbols} {
		wg.Add(1)
		go func(groupID int, symbols []string) {
			defer wg.Done()
			connID := fmt.Sprintf("conn-%d", groupID+1)
			t.Logf("[%s] Connecting to WebSocket at %s", connID, bitrueFuturesWSURL)
			conn, _, err := websocket.DefaultDialer.Dial(bitrueFuturesWSURL, nil)
			if err != nil {
				t.Errorf("[%s] WebSocket dial error: %v", connID, err)
				return
			}
			defer func() {
				conn.Close()
				t.Logf("[%s] WebSocket connection closed", connID)
			}()
			t.Logf("[%s] Connected successfully", connID)

			// Subscribe to all symbols
			for i, symbol := range symbols {
				msg := createSubscribeMessage(symbol)
				t.Logf("[%s] [%d/%d] Subscribing to %s", connID, i+1, len(symbols), symbol)
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					t.Errorf("[%s] Failed to subscribe to %s: %v", connID, symbol, err)
					return
				}
			}

			deadline := time.Now().Add(keepaliveDuration)
			conn.SetReadDeadline(deadline)
			t.Logf("[%s] Starting to read messages (keepalive test, deadline: %s)", connID, deadline.Format(time.RFC3339))
			var lastPong time.Time
			conn.SetPongHandler(func(appData string) error {
				lastPong = time.Now()
				t.Logf("[%s] Received pong at %s", connID, lastPong.Format(time.RFC3339))
				return nil
			})

			for time.Now().Before(deadline) {
				msgType, message, err := conn.ReadMessage()
				if err != nil {
					t.Logf("[%s] Read error: %v", connID, err)
					break
				}
				if msgType == websocket.PingMessage {
					t.Logf("[%s] Received ping, sending pong", connID)
					conn.WriteMessage(websocket.PongMessage, nil)
					continue
				}
				if msgType == websocket.PongMessage {
					lastPong = time.Now()
					t.Logf("[%s] Received pong message at %s", connID, lastPong.Format(time.RFC3339))
					continue
				}
				var tradeMsg BitrueTradeMessage
				if err := json.Unmarshal(message, &tradeMsg); err == nil && len(tradeMsg.Tick.Data) > 0 {
					ch := tradeMsg.Channel
					parts := strings.Split(ch, "_")
					if len(parts) >= 4 && parts[0] == "market" && parts[len(parts)-2] == "trade" {
						sym := strings.ToUpper(parts[1] + "_" + parts[2])
						results <- sym
					}
				}
			}
			t.Logf("[%s] Finished keepalive read loop", connID)
		}(groupID, group)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
			t.Log("All WebSocket handlers completed (keepalive)")
	}()

	select {
	case <-done:
		t.Log("Keepalive test completed before timeout")
	case <-timeout:
		t.Logf("Keepalive test reached timeout (%v)", keepaliveDuration)
	}
}
