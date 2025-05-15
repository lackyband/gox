package tester

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"github.com/gorilla/websocket"
)

func genDepthChannel(symbol string) string {
	normalized := strings.ToLower(symbol)
	normalized = strings.Replace(normalized, "-usdt", "usdt", -1)
	normalized = strings.Replace(normalized, "-", "_", -1)
	return fmt.Sprintf("market_%s_depth_step0", normalized)
}

func createDepthSubscribeMessage(symbol string) []byte {
	normalized := strings.ToLower(symbol)
	normalized = strings.Replace(normalized, "-usdt", "usdt", -1)
	normalized = strings.Replace(normalized, "-", "_", -1)
	channel := genDepthChannel(normalized)
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

// BitrueDepthData represents the order book depth data
// Note: The exchange may send numbers or strings for price/amount
// Using json.Number to handle both cases
// Format: [ ["price", "amount"], ... ]
type BitrueDepthData struct {
	Asks [][]json.RawMessage `json:"asks"`
	Bids [][]json.RawMessage `json:"bids"`
}

// parseDepthLevel parses a single price/amount level from the depth data
func parseDepthLevel(level []json.RawMessage) (price, amount string, err error) {
	if len(level) < 2 {
		return "", "", fmt.Errorf("invalid depth level format")
	}

	// Parse price (can be string or number)
	var priceVal interface{}
	if err := json.Unmarshal(level[0], &priceVal); err != nil {
		return "", "", fmt.Errorf("failed to parse price: %v", err)
	}
	switch v := priceVal.(type) {
	case string:
		price = v
	case float64:
		price = strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		price = v.String()
	default:
		return "", "", fmt.Errorf("unexpected price type: %T", v)
	}

	// Parse amount (can be string or number)
	var amountVal interface{}
	if err := json.Unmarshal(level[1], &amountVal); err != nil {
		return "", "", fmt.Errorf("failed to parse amount: %v", err)
	}
	switch v := amountVal.(type) {
	case string:
		amount = v
	case float64:
		amount = strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		amount = v.String()
	default:
		return "", "", fmt.Errorf("unexpected amount type: %T", v)
	}

	return price, amount, nil
}

type BitrueDepthMessage struct {
	Channel string         `json:"channel"`
	Tick    BitrueDepthData `json:"tick"`
	Ts      int64          `json:"ts"`
	Status  string         `json:"status"`
}

func TestBitrueFuturesDepthPool(t *testing.T) {
	t.Log("=== Starting Bitrue Futures Depth Pool Test ===")
	symbols := fetchFuturesContracts(t)
	if len(symbols) > 100 {
		symbols = symbols[:100]
		t.Logf("Limited to first 100 symbols for testing")
	}
	var symbolGroups [][]string
	for i := 0; i < len(symbols); i += maxSubsPerConn {
		end := i + maxSubsPerConn
		if end > len(symbols) {
			end = len(symbols)
		}
		group := symbols[i:end]
		symbolGroups = append(symbolGroups, group)
		t.Logf("  Group %d: %d symbols (first: %s, last: %s)", len(symbolGroups), len(group), group[0], group[len(group)-1])
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

			t.Logf("[%s] Subscribing to %d symbols...", connID, len(symbols))
			for i, symbol := range symbols {
				msg := createDepthSubscribeMessage(symbol)
				t.Logf("[%s] [%d/%d] Subscribing to %s", connID, i+1, len(symbols), symbol)
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					t.Errorf("[%s] Failed to subscribe to %s: %v", connID, symbol, err)
					return
				}
			}

			gotDepth := make(map[string]bool)
			deadline := time.Now().Add(30 * time.Second)
			conn.SetReadDeadline(deadline)
			t.Logf("[%s] Starting to read messages (timeout: %s)", connID, deadline.Format(time.RFC3339))
			for len(gotDepth) < len(symbols) && time.Now().Before(deadline) {
				msgType, message, err := conn.ReadMessage()
				if err != nil {
					t.Logf("[%s] Read error: %v", connID, err)
					break
				}
				var processedMessage []byte
				if msgType == websocket.BinaryMessage {
					processedMessage, err = decompressGzip(message)
					if err != nil {
						t.Logf("[%s] Failed to decompress: %v", connID, err)
						continue
					}
				} else {
					processedMessage = message
				}
				var depthMsg BitrueDepthMessage
				if err := json.Unmarshal(processedMessage, &depthMsg); err != nil {
					t.Logf("[%s] Failed to unmarshal message: %v", connID, err)
					continue
				}
				ch := depthMsg.Channel
				parts := strings.Split(ch, "_")
				if len(parts) >= 4 && parts[0] == "market" && parts[len(parts)-2] == "depth" {
					sym := strings.ToUpper(parts[1] + "_" + parts[2])

					// Process asks
					for _, ask := range depthMsg.Tick.Asks {
						price, amount, err := parseDepthLevel(ask)
						if err != nil {
							t.Logf("[%s] Failed to parse ask level: %v", connID, err)
							continue
						}
						t.Logf("[%s] %s ASK: %s @ %s", connID, sym, amount, price)
					}

					// Process bids
					for _, bid := range depthMsg.Tick.Bids {
						price, amount, err := parseDepthLevel(bid)
						if err != nil {
							t.Logf("[%s] Failed to parse bid level: %v", connID, err)
							continue
						}
						t.Logf("[%s] %s BID: %s @ %s", connID, sym, amount, price)
					}

					if !gotDepth[sym] {
						gotDepth[sym] = true
						results <- sym
						t.Logf("[%s] ✓ New depth for %s (total: %d/%d)", connID, sym, len(gotDepth), len(symbols))
					} else {
						t.Logf("[%s] Duplicate depth for %s", connID, sym)
					}
				} else {
					t.Logf("[%s] Non-depth message: %+v", connID, depthMsg)
				}
			}
		}(groupID, group)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		t.Log("All WebSocket handlers completed")
	}()

	depths := make(map[string]bool)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case sym := <-results:
			depths[sym] = true
			t.Logf("=== Progress: %d/%d symbols received depth ===", len(depths), len(symbols))
			if len(depths) == len(symbols) {
				t.Log("=== SUCCESS: Received depth for all symbols ===")
				return
			}
		case <-ticker.C:
			t.Logf("Waiting... (%d/%d symbols received so far)", len(depths), len(symbols))
		case <-done:
			t.Logf("All WebSocket handlers completed with %d/%d symbols received", len(depths), len(symbols))
			if len(depths) != len(symbols) {
				t.Errorf("Did not receive depth for all symbols (got %d/%d)", len(depths), len(symbols))
			}
			return
		case <-timeout:
			t.Logf("=== TEST TIMEOUT ===")
			t.Logf("Received depth for %d/%d symbols:", len(depths), len(symbols))
			for sym := range depths {
				t.Logf("  ✓ %s", sym)
			}
			t.Logf("Missing depth for %d symbols:", len(symbols)-len(depths))
			for _, sym := range symbols {
				if !depths[sym] {
					t.Logf("  ✗ %s", sym)
				}
			}
			t.Fatalf("Timeout: Did not receive depth data for all symbols (got %d/%d)", len(depths), len(symbols))
		}
	}
}
