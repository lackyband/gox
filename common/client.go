package common

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessageHandler defines a callback for WebSocket messages
type WebSocketMessageHandler func(message []byte)

// WebSocketClient manages a WebSocket connection
type WebSocketClient struct {
	conn        *websocket.Conn
	url         string
	messageChan chan []byte
	done        chan struct{}
	mutex       sync.Mutex
	callback    WebSocketMessageHandler
}

// WebSocketMessage represents a WebSocket message structure
type WebSocketMessage struct {
	Event     string                 `json:"event"`
	Params    map[string]interface{} `json:"params"`
	Channel   string                 `json:"channel"`
	Timestamp int64                  `json:"ts"`
	Status    string                 `json:"status"`
	Tick      interface{}            `json:"tick"`
	Data      interface{}            `json:"data"`
	EventRep  string                 `json:"event_rep"`
	Ping      string                 `json:"ping"`
	Pong      string                 `json:"pong"`
}

// NewWebSocketClient initializes a WebSocket client
func NewWebSocketClient(wsURL string) *WebSocketClient {
	return &WebSocketClient{
		url:         wsURL,
		messageChan: make(chan []byte, 100),
		done:        make(chan struct{}),
		callback:    nil,
	}
}

// SetCallback sets a callback function for message handling
func (w *WebSocketClient) SetCallback(cb WebSocketMessageHandler) {
	w.callback = cb
}

// Connect establishes a WebSocket connection
func (w *WebSocketClient) Connect() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.conn != nil {
		return nil
	}

	conn, _, err := websocket.DefaultDialer.Dial(w.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	w.conn = conn
	return nil
}

// Close closes the WebSocket connection
func (w *WebSocketClient) Close() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
	close(w.done)
}

// ReadMessages handles incoming WebSocket messages
func (w *WebSocketClient) ReadMessages(isMarket bool) {
	for {
		select {
		case <-w.done:
			return
		default:
			if w.conn == nil {
				return
			}
			_, message, err := w.conn.ReadMessage()
			if err != nil {
				fmt.Printf("WebSocket read error: %v\n", err)
				w.Close()
				return
			}

			// Decompress market data (Gzip)
			if isMarket {
				reader, err := gzip.NewReader(bytes.NewReader(message))
				if err != nil {
					fmt.Printf("Failed to decompress message: %v\n", err)
					continue
				}
				decompressed, err := io.ReadAll(reader)
				reader.Close()
				if err != nil {
					fmt.Printf("Failed to read decompressed message: %v\n", err)
					continue
				}
				message = decompressed
			}

			// Handle ping/pong
			var msg WebSocketMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg.Ping != "" {
					pong := WebSocketMessage{Pong: msg.Ping}
					w.conn.WriteJSON(pong)
					continue
				}
			}

			// Send message to callback if registered, else to channel
			if w.callback != nil {
				w.callback(message)
			} else {
				select {
				case w.messageChan <- message:
				default:
					fmt.Println("Message channel full, dropping message")
				}
			}
		}
	}
}

// Reconnect attempts to reconnect with exponential backoff
func (w *WebSocketClient) Reconnect(isMarket bool) error {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		if err := w.Connect(); err != nil {
			attempts++
			backoff := time.Duration(1<<uint(attempts)) * time.Second
			fmt.Printf("Reconnect attempt %d failed: %v, retrying in %v\n", attempts, err, backoff)
			time.Sleep(backoff)
			continue
		}
		go w.ReadMessages(isMarket)
		return nil
	}
	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}

// DoRequest executes an HTTP request with headers and signature (if needed)
func DoRequest(client *http.Client, baseURL, method, endpoint string, params url.Values, body url.Values, apiKey, secretKey string, signed bool) ([]byte, error) {
	u, err := url.Parse(baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	var totalParams string
	var bodyBytes []byte
	if body != nil {
		bodyBytes = []byte(body.Encode())
		totalParams = body.Encode()
	}
	if params != nil {
		u.RawQuery = params.Encode()
		if totalParams == "" {
			totalParams = u.RawQuery
		} else {
			totalParams = u.RawQuery + "&" + totalParams
		}
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if apiKey != "" {
		req.Header.Set("X-MBX-APIKEY", apiKey)
	}

	if signed {
		timestamp := time.Now().UnixMilli()
		if params == nil {
			params = url.Values{}
		}
		params.Set("timestamp", strconv.FormatInt(timestamp, 10))
		u.RawQuery = params.Encode()
		totalParams = u.RawQuery
		if body != nil {
			totalParams += "&" + body.Encode()
		}
		signature := GenerateSignature(totalParams, secretKey)
		params.Set("signature", signature)
		u.RawQuery = params.Encode()
		req.URL = u
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(respBody))
	}

	return respBody, nil
}

// GenerateSignature creates an HMAC SHA256 signature
func GenerateSignature(totalParams, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(totalParams))
	return hex.EncodeToString(mac.Sum(nil))
}
