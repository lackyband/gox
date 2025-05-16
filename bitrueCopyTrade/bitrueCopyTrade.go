package bitrueCopyTrade

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// QueryOrderParams represents parameters for querying a copy trading order
type QueryOrderParams struct {
	ContractName string `json:"contractName"`
	OrderID      string `json:"orderId,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
}

// OrderParams represents parameters for placing a new copy trading order
type OrderParams struct {
	ContractName           string  `json:"contractName"`
	ClientOrderID          int64   `json:"clientOrderId"`
	Side                   string  `json:"side"`
	Type                   string  `json:"type"`
	PositionType           int     `json:"positionType"`
	Open                   string  `json:"open"`
	Volume                 float64 `json:"volume"`
	Amount                 float64 `json:"amount"`
	Price                  float64 `json:"price"`
	Leverage               int     `json:"leverage"`
	TriggerOrderType       int     `json:"triggerOrderType"`
	TriggerType            int     `json:"triggerType"`
	TriggerPriceType       int     `json:"triggerPriceType"`
	TriggerPrice           float64 `json:"triggerPrice"`
	ConditionOrder         bool    `json:"conditionOrder"`
	PositionID             int64   `json:"positionId,omitempty"`
	TriggerOrderCreateParams []struct {
		ClientOrderID    int64   `json:"clientOrderId"`
		TriggerType      int     `json:"triggerType"`
		TriggerPriceType int     `json:"triggerPriceType"`
		TriggerPrice     float64 `json:"triggerPrice"`
		Type             string  `json:"type"`
		Price            float64 `json:"price"`
	} `json:"triggerOrderCreateParams,omitempty"`
}

// CancelOrderParams represents parameters for canceling a copy trading order
type CancelOrderParams struct {
	ContractName   string `json:"contractName"`
	OrderID        int64  `json:"orderId,omitempty"`
	ClientOrderID  string `json:"clientOrderId,omitempty"`
	ConditionOrder bool   `json:"conditionOrder"`
}

// Connect establishes a WebSocket connection
func (w *WebSocketClient) Connect(apiKey string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.conn != nil {
		return nil
	}

	const maxRetries = 3
	const retryDelay = 2 * time.Second

	headers := http.Header{}
	if apiKey != "" {
		headers.Set("X-CH-APIKEY", apiKey)
	}

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, _, err := websocket.DefaultDialer.Dial(w.url, headers)
		if err == nil {
			w.conn = conn
			return nil
		}
		fmt.Printf("WebSocket connect attempt %d failed: %v, retrying in %v...\n", attempt, err, retryDelay)
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("failed to connect to WebSocket after %d attempts: %v", maxRetries, err)
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
func (w *WebSocketClient) ReadMessages() {
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

			// Handle ping/pong
			var msg WebSocketMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				if msg.Event == "ping" {
					pong := map[string]interface{}{
						"event": "pong",
						"ts":    msg.Timestamp,
					}
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
func (w *WebSocketClient) Reconnect() error {
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		if err := w.Connect(""); err != nil {
			attempts++
			backoff := time.Duration(1<<uint(attempts)) * time.Second
			fmt.Printf("Reconnect attempt %d failed: %v, retrying in %v\n", attempts, err, backoff)
			time.Sleep(backoff)
			continue
		}
		go w.ReadMessages()
		return nil
	}
	return fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
}

// BitrueClient represents a client for the Bitrue Copy Trading API
type BitrueClient struct {
	baseURL       string
	wsUserURL     string
	userStreamURL string
	apiKey        string
	secretKey     string
	httpClient    *http.Client
	wsUserClient  *WebSocketClient
	wsUserMutex   sync.Mutex
}

// NewBitrueClient initializes a new BitrueClient
func NewBitrueClient(apiKey, secretKey string) *BitrueClient {
	return &BitrueClient{
		baseURL:       "https://fapi.bitrue.com",
		userStreamURL: "https://fapiws-auth.bitrue.com",
		wsUserURL:     "wss://fapiws.bitrue.com",
		apiKey:        apiKey,
		secretKey:     secretKey,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		wsUserClient:  nil,
	}
}

// StartWebSocketUser starts the user data WebSocket client
func (c *BitrueClient) StartWebSocketUser(listenKey string) error {
	c.wsUserMutex.Lock()
	defer c.wsUserMutex.Unlock()

	if c.wsUserClient != nil {
		return nil
	}

	wsURL := fmt.Sprintf("%s/stream?listenKey=%s", c.wsUserURL, listenKey)
	c.wsUserClient = NewWebSocketClient(wsURL)
	if err := c.wsUserClient.Connect(c.apiKey); err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	go c.wsUserClient.ReadMessages()
	return nil
}

// StopWebSocketUser stops the user data WebSocket client
func (c *BitrueClient) StopWebSocketUser() {
	c.wsUserMutex.Lock()
	defer c.wsUserMutex.Unlock()

	if c.wsUserClient != nil {
		c.wsUserClient.Close()
		c.wsUserClient = nil
	}
}

// SubscribeUserData subscribes to the user data websocket channel
func (c *BitrueClient) SubscribeUserData(listenKey string, callback WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	// Copy trading user data streams are established on connection
	return nil
}

// GetWebSocketMessages returns the channel for receiving WebSocket messages
func (c *BitrueClient) GetWebSocketMessages() chan []byte {
	return c.wsUserClient.messageChan
}

// DoRequest executes an HTTP request with headers and signature
func (c *BitrueClient) DoRequest(method, baseURL, endpoint string, params url.Values, body interface{}, signed bool) ([]byte, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var respBody []byte
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		u, err := url.Parse(baseURL + endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %v", err)
		}

		var bodyBytes []byte
		if body != nil {
			bodyBytes, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %v", err)
			}
		}

		if params != nil && method == http.MethodGet {
			u.RawQuery = params.Encode()
		}

		req, err := http.NewRequest(method, u.String(), bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("X-CH-APIKEY", c.apiKey)
		}

		if signed {
			timestamp := time.Now().UnixMilli()
			path := u.Path
			if u.RawQuery != "" {
				path += "?" + u.RawQuery
			}
			signingString := fmt.Sprintf("%d%s%s%s", timestamp, strings.ToUpper(method), path, string(bodyBytes))
			signature := c.GenerateSignature(signingString)
			req.Header.Set("X-CH-SIGN", signature)
			req.Header.Set("X-CH-TS", strconv.FormatInt(timestamp, 10))
		}



		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %v", err)
			time.Sleep(retryDelay)
			continue
		}
		defer resp.Body.Close()

		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		if resp.StatusCode == http.StatusServiceUnavailable {
			fmt.Printf("Attempt %d: 503 Service Unavailable, retrying in %v...\n", attempt, retryDelay)
			time.Sleep(retryDelay)
			lastErr = fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return respBody, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// GenerateSignature creates an HMAC SHA256 signature
func (c *BitrueClient) GenerateSignature(signingString string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(signingString))
	return hex.EncodeToString(mac.Sum(nil))
}

// OpenOrders retrieves all open copy trading orders
func (c *BitrueClient) OpenOrders(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return c.DoRequest(http.MethodGet, c.baseURL, "/fapi/v2/copyTrading/openOrders", params, nil, true)
}

// CancelOrder cancels a copy trading order
func (c *BitrueClient) CancelOrder(params struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}) ([]byte, error) {
	body := map[string]interface{}{
		"contractName":   params.ContractName,
		"conditionOrder": params.ConditionOrder,
	}
	if params.ClientOrderID != "" {
		body["clientOrderId"] = params.ClientOrderID
	}
	if params.OrderID > 0 {
		body["orderId"] = params.OrderID
	}
	return c.DoRequest(http.MethodPost, c.baseURL, "/fapi/v2/copyTrading/cancel", nil, body, true)
}

// QueryOrder queries a copy trading order
func (c *BitrueClient) QueryOrder(params struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}) ([]byte, error) {
	query := url.Values{}
	query.Set("contractName", params.ContractName)
	if params.ClientOrderID != "" {
		query.Set("clientOrderId", params.ClientOrderID)
	}
	if params.OrderID > 0 {
		query.Set("orderId", strconv.FormatInt(params.OrderID, 10))
	}
	if params.ConditionOrder {
		query.Set("conditionOrder", "true")
	}
	return c.DoRequest(http.MethodGet, c.baseURL, "/fapi/v2/copyTrading/order", query, nil, true)
}

// NewOrder places a new copy trading order
func (c *BitrueClient) NewOrder(params struct {
	ContractName       string
	ClientOrderID      string
	Side               string
	Type               string
	PositionType       int
	Open               string
	Volume             float64
	Amount             float64
	Price              float64
	Leverage           int
	TriggerOrderType   int
	TriggerType        int
	TriggerPriceType   int
	TriggerPrice       float64
	ConditionOrder     bool
	PositionID         int
	TriggerOrderParams []map[string]interface{}
}) ([]byte, error) {
	body := map[string]interface{}{
		"contractName":   params.ContractName,
		"side":           params.Side,
		"type":           params.Type,
		"positionType":   params.PositionType,
		"open":           params.Open,
		"volume":         params.Volume,
		"amount":         params.Amount,
		"price":          params.Price,
		"leverage":       params.Leverage,
		"conditionOrder": params.ConditionOrder,
	}
	if params.ClientOrderID != "" {
		body["clientOrderId"] = params.ClientOrderID
	}
	if params.TriggerOrderType > 0 {
		body["triggerOrderType"] = params.TriggerOrderType
	}
	if params.TriggerType > 0 {
		body["triggerType"] = params.TriggerType
	}
	if params.TriggerPriceType > 0 {
		body["triggerPriceType"] = params.TriggerPriceType
	}
	if params.TriggerPrice > 0 {
		body["triggerPrice"] = params.TriggerPrice
	}
	if params.PositionID > 0 {
		body["positionId"] = params.PositionID
	}
	if len(params.TriggerOrderParams) > 0 {
		body["triggerOrderCreateParams"] = params.TriggerOrderParams
	}
	return c.DoRequest(http.MethodPost, c.baseURL, "/fapi/v2/copyTrading/order", nil, body, true)
}

// Account retrieves copy trading account information
func (c *BitrueClient) Account() ([]byte, error) {
	return c.DoRequest(http.MethodGet, c.baseURL, "/fapi/v2/copyTrading/account", nil, nil, true)
}

// Positions retrieves copy trading positions
func (c *BitrueClient) Positions() ([]byte, error) {
	return c.DoRequest(http.MethodGet, c.baseURL, "/fapi/v2/copyTrading/positions", nil, nil, true)
}

// ConfigQuerySymbols retrieves copy trading symbol configuration
func (c *BitrueClient) ConfigQuerySymbols() ([]byte, error) {
	return c.DoRequest(http.MethodGet, c.baseURL, "/fapi/v2/copyTrading/configQuerySymbols", nil, nil, true)
}

// Status retrieves copy trading status
func (c *BitrueClient) Status() ([]byte, error) {
	return c.DoRequest(http.MethodPost, c.baseURL, "/fapi/v2/copyTrading/status", nil, nil, true)
}

// User Stream Endpoints

// CreateListenKey creates a new user data stream listenKey
func (c *BitrueClient) CreateListenKey() ([]byte, error) {
	return c.DoRequest(http.MethodPost, c.userStreamURL, "/user_stream/api/v1/listenKey", nil, nil, true)
}

// KeepAliveListenKey extends the validity of a listenKey
func (c *BitrueClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.DoRequest(http.MethodPut, c.userStreamURL, endpoint, nil, nil, true)
}

// CloseListenKey closes a user data stream
func (c *BitrueClient) CloseListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.DoRequest(http.MethodDelete, c.userStreamURL, endpoint, nil, nil, true)
}
