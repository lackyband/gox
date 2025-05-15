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

type BitrueContract struct {
	Symbol         string `json:"symbol"`
	PricePrecision int    `json:"pricePrecision"`
	Status         int    `json:"status"`
}

type BitrueTradeData struct {
	Amount string `json:"amount"`
	Ds     string `json:"ds"`
	Price  string `json:"price"`
	Side   string `json:"side"`
	Ts     int64  `json:"ts"`
	Vol    string `json:"vol"`
}

type BitrueTradeMessage struct {
	EventRep string        `json:"event_rep"`
	Channel  string        `json:"channel"`
	Data     []interface{} `json:"data"`
	Tick     struct {
		Data []BitrueTradeData `json:"data"`
		Ts   int64             `json:"ts"`
	} `json:"tick"`
	Ts     int64  `json:"ts"`
	Status string `json:"status"`
}

// fetchFuturesContracts fetches all active contracts from Bitrue REST API

// genTradeChannel generates the trade channel name for a symbol
func genTradeChannel(symbol string) string {
	normalized := strings.ToLower(symbol)
	normalized = strings.Replace(normalized, "-usdt", "usdt", -1)
	normalized = strings.Replace(normalized, "-", "_", -1)
	return fmt.Sprintf("market_%s_trade_ticker", normalized)
}

// createSubscribeMessage generates a subscription message
func createSubscribeMessage(symbol string) []byte {
	normalized := strings.ToLower(symbol)
	normalized = strings.Replace(normalized, "-usdt", "usdt", -1)
	normalized = strings.Replace(normalized, "-", "_", -1)
	channel := genTradeChannel(normalized)
	msg := map[string]interface{}{
		"event": "sub",
		"params": map[string]string{
			"cb_id":   normalized,
			"channel": channel,
		},
	}
	data, _ := json.Marshal(msg)
	return data
}

// decompressGzip decompresses gzip-compressed data

func TestBitrueFuturesTradePool(t *testing.T) {
	t.Log("=== Starting Bitrue Futures Trade Pool Test ===")
	t.Logf("Fetching active futures contracts from %s", bitrueFuturesRestAPI)
	symbols := fetchFuturesContracts(t)
	if len(symbols) > 100 {
		symbols = symbols[:100] // Limit for test speed
		t.Logf("Limited to first 100 symbols for testing")
	}
	t.Logf("Found %d active symbols for testing", len(symbols))

	// Pool symbols into groups
	t.Logf("Grouping %d symbols into connections (max %d symbols per connection)", len(symbols), maxSubsPerConn)
	var symbolGroups [][]string
	for i := 0; i < len(symbols); i += maxSubsPerConn {
		end := i + maxSubsPerConn
		if end > len(symbols) {
			end = len(symbols)
		}
		group := symbols[i:end]
		symbolGroups = append(symbolGroups, group)
		t.Logf("  Group %d: %d symbols (first: %s, last: %s)", 
			len(symbolGroups), len(group), group[0], group[len(group)-1])
	}

	var wg sync.WaitGroup
	results := make(chan string, len(symbols))
	timeout := time.After(60 * time.Second)
	for groupID, group := range symbolGroups {
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

			// Subscribe to all symbols in this group
			t.Logf("[%s] Subscribing to %d symbols...", connID, len(symbols))
			for i, symbol := range symbols {
				msg := createSubscribeMessage(symbol)
				t.Logf("[%s] [%d/%d] Subscribing to %s", connID, i+1, len(symbols), symbol)
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					t.Errorf("[%s] Failed to subscribe to %s: %v", connID, symbol, err)
					return
				}
			}

			// Track which symbols got a trade
			gotTrade := make(map[string]bool)
			deadline := time.Now().Add(30 * time.Second)
			conn.SetReadDeadline(deadline)
			t.Logf("[%s] Starting to read messages (timeout: %s)", connID, deadline.Format(time.RFC3339))
			
			for len(gotTrade) < len(symbols) && time.Now().Before(deadline) {
				msgType, message, err := conn.ReadMessage()
				if err != nil {
					t.Logf("[%s] Read error: %v", connID, err)
					break
				}
				t.Logf("[%s] Received message (type: %d, len: %d)", connID, msgType, len(message))

				var processedMessage []byte
				if msgType == websocket.BinaryMessage {
					t.Logf("[%s] Decompressing gzipped message", connID)
					processedMessage, err = decompressGzip(message)
					if err != nil {
						t.Logf("[%s] Failed to decompress: %v", connID, err)
						continue
					}
					t.Logf("[%s] Decompressed message: %d -> %d bytes", connID, len(message), len(processedMessage))
				} else {
					processedMessage = message
					t.Logf("[%s] Text message: %s", connID, string(processedMessage))
				}

				var tradeMsg BitrueTradeMessage
				if err := json.Unmarshal(processedMessage, &tradeMsg); err != nil {
					t.Logf("[%s] Failed to unmarshal message: %v", connID, err)
					t.Logf("[%s] Message content: %s", connID, string(processedMessage))
					continue
				}

				t.Logf("[%s] Message received - Event: %s, Channel: %s, Status: %s", 
					connID, tradeMsg.EventRep, tradeMsg.Channel, tradeMsg.Status)

				// Check if this is a trade message (has Tick.Data)
				if len(tradeMsg.Tick.Data) > 0 {
					// Extract symbol from channel (format: market_e_btcusdt_trade_ticker)
					ch := tradeMsg.Channel
					parts := strings.Split(ch, "_")
					if len(parts) >= 4 && parts[0] == "market" && parts[len(parts)-2] == "trade" {
						sym := strings.ToUpper(parts[1] + "_" + parts[2])
						
						// Log the trade details
						tradeData := tradeMsg.Tick.Data[0]
						t.Logf("[%s] TRADE %s: %s %s @ %s (vol: %s, ts: %d)", 
							connID, sym, tradeData.Side, tradeData.Amount, 
							tradeData.Price, tradeData.Vol, tradeData.Ts)
						
						// Track unique symbols with trades
						if !gotTrade[sym] {
							gotTrade[sym] = true
							results <- sym
							t.Logf("[%s] ✓ New trade for %s (total: %d/%d)", 
								connID, sym, len(gotTrade), len(symbols))
						} else {
							t.Logf("[%s] Duplicate trade for %s", connID, sym)
						}
					} else {
						t.Logf("[%s] Unexpected channel format: %s", connID, ch)
					}
				} else {
					t.Logf("[%s] Non-trade message: %+v", connID, tradeMsg)
				}
			}
		}(groupID, group)
	}

	// Wait for all groups or timeout
	t.Log("=== Starting to wait for trade data ===")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		t.Log("All WebSocket handlers completed")
	}()

	trades := make(map[string]bool)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case sym := <-results:
			trades[sym] = true
			t.Logf("=== Progress: %d/%d symbols received trades ===", len(trades), len(symbols))
			if len(trades) == len(symbols) {
				t.Log("=== SUCCESS: Received trades for all symbols ===")
				return // All received
			}

		case <-ticker.C:
			t.Logf("Waiting... (%d/%d symbols received so far)", len(trades), len(symbols))

		case <-done:
			t.Logf("All WebSocket handlers completed with %d/%d symbols received", len(trades), len(symbols))
			if len(trades) != len(symbols) {
				t.Errorf("Did not receive trades for all symbols (got %d/%d)", len(trades), len(symbols))
			}
			return

		case <-timeout:
			t.Logf("=== TEST TIMEOUT ===")
			t.Logf("Received trades for %d/%d symbols:", len(trades), len(symbols))
			for sym := range trades {
				t.Logf("  ✓ %s", sym)
			}
			t.Logf("Missing trades for %d symbols:", len(symbols)-len(trades))
			for _, sym := range symbols {
				if !trades[sym] {
					t.Logf("  ✗ %s", sym)
				}
			}
			t.Fatalf("Timeout: Did not receive trade data for all symbols (got %d/%d)", len(trades), len(symbols))
		}
	}
}
