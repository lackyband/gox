package bitrueSpot

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
				if msg.Event == "ping" {
					pong := WebSocketMessage{
						Event:     "pong",
						Timestamp: msg.Timestamp,
					}
					if err := w.conn.WriteJSON(pong); err != nil {
						fmt.Printf("Failed to send pong: %v\n", err)
					}
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

// BitrueClient represents a client for the Bitrue Spot API
type BitrueClient struct {
	baseURL        string
	wsMarketURL    string
	wsUserURL      string
	apiKey         string
	secretKey      string
	httpClient     *http.Client
	wsMarketClient *WebSocketClient
	wsUserClient   *WebSocketClient
	wsMarketMutex  sync.Mutex
	wsUserMutex    sync.Mutex
}

// NewBitrueClient initializes a new BitrueClient
func NewBitrueClient(apiKey, secretKey string) *BitrueClient {
	return &BitrueClient{
		baseURL:        "https://open.bitrue.com",
		wsMarketURL:    "wss://ws.bitrue.com/market/ws",
		wsUserURL:      "wss://wsapi.bitrue.com",
		apiKey:         apiKey,
		secretKey:      secretKey,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		wsMarketClient: nil,
		wsUserClient:   nil,
	}
}

// StartWebSocketMarket starts the market data WebSocket client
func (c *BitrueClient) StartWebSocketMarket() error {
	c.wsMarketMutex.Lock()
	defer c.wsMarketMutex.Unlock()

	if c.wsMarketClient != nil {
		return nil
	}

	c.wsMarketClient = NewWebSocketClient(c.wsMarketURL)
	if err := c.wsMarketClient.Connect(); err != nil {
		return err
	}
	go c.wsMarketClient.ReadMessages(true)
	return nil
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
	if err := c.wsUserClient.Connect(); err != nil {
		return err
	}
	go c.wsUserClient.ReadMessages(false)
	return nil
}

// StopWebSocketMarket stops the market data WebSocket client
func (c *BitrueClient) StopWebSocketMarket() {
	c.wsMarketMutex.Lock()
	defer c.wsMarketMutex.Unlock()

	if c.wsMarketClient != nil {
		c.wsMarketClient.Close()
		c.wsMarketClient = nil
	}
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

// SubscribeDepth subscribes to the depth websocket channel
func (c *BitrueClient) SubscribeDepth(symbol string, callback WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_simple_depth_step0"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(symbol), -1)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   strings.ToLower(symbol),
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeUserOrderUpdate subscribes to the user order update websocket channel
func (c *BitrueClient) SubscribeUserOrderUpdate(callback WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": "user_order_update",
		},
	}
	return c.wsUserClient.conn.WriteJSON(msg)
}

// SubscribeUserBalanceUpdate subscribes to the user balance update websocket channel
func (c *BitrueClient) SubscribeUserBalanceUpdate(callback WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": "user_balance_update",
		},
	}
	return c.wsUserClient.conn.WriteJSON(msg)
}

// SubscribeWebSocket subscribes to a WebSocket channel
func (c *BitrueClient) SubscribeWebSocket(channel, symbol, interval string, isMarket bool) error {
	client := c.wsMarketClient
	if !isMarket {
		client = c.wsUserClient
	}
	if client == nil || client.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}

	if interval != "" {
		channel = strings.Replace(channel, "$interval", interval, -1)
	}
	channel = strings.Replace(channel, "$symbol", strings.ToLower(symbol), -1)

	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   strings.ToLower(symbol),
		},
	}
	return client.conn.WriteJSON(msg)
}

// UnsubscribeWebSocket unsubscribes from a WebSocket channel
func (c *BitrueClient) UnsubscribeWebSocket(channel, symbol, interval string, isMarket bool) error {
	client := c.wsMarketClient
	if !isMarket {
		client = c.wsUserClient
	}
	if client == nil || client.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}

	if interval != "" {
		channel = strings.Replace(channel, "$interval", interval, -1)
	}
	channel = strings.Replace(channel, "$symbol", strings.ToLower(symbol), -1)

	msg := WebSocketMessage{
		Event: "unsub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   strings.ToLower(symbol),
		},
	}
	return client.conn.WriteJSON(msg)
}

// GetWebSocketMessages returns the channel for receiving WebSocket messages
func (c *BitrueClient) GetWebSocketMessages(isMarket bool) chan []byte {
	if isMarket {
		return c.wsMarketClient.messageChan
	}
	return c.wsUserClient.messageChan
}

// DoRequest executes an HTTP request with headers and signature
func (c *BitrueClient) doRequest(method, endpoint string, params url.Values, body url.Values, signed bool) ([]byte, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	var respBody []byte
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		u, err := url.Parse(c.baseURL + endpoint)
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
		if c.apiKey != "" {
			req.Header.Set("X-MBX-APIKEY", c.apiKey)
		}

		if signed {
			if params == nil {
				params = url.Values{}
			}
			timestamp := time.Now().UnixMilli()
			params.Set("timestamp", strconv.FormatInt(timestamp, 10))
			signingString := params.Encode()
			if body != nil {
				signingString += "&" + body.Encode()
			}
			fmt.Printf("DEBUG SIGNING STRING: %s\n", signingString)
			signature := c.GenerateSignature(signingString)
			fmt.Printf("DEBUG SIGNATURE: %s\n", signature)
			signedQuery := params.Encode() + "&signature=" + signature
			u.RawQuery = signedQuery
			req.URL = u
		}

		fmt.Printf("\nDEBUG REQUEST URL: %s\n", req.URL.String())
		fmt.Printf("DEBUG REQUEST HEADERS:\n")
		for k, v := range req.Header {
			fmt.Printf("  %s: %s\n", k, v)
		}
		if body != nil {
			fmt.Printf("DEBUG REQUEST BODY: %s\n", body.Encode())
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
			return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// GenerateSignature creates an HMAC SHA256 signature
func (c *BitrueClient) GenerateSignature(totalParams string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(totalParams))
	return hex.EncodeToString(mac.Sum(nil))
}

// REST API Endpoints

// Ping tests connectivity to the API
func (c *BitrueClient) Ping() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/api/v1/ping", nil, nil, false)
}

// ServerTime retrieves the server time
func (c *BitrueClient) ServerTime() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/api/v1/time", nil, nil, false)
}

// ExchangeInfo retrieves trading pairs and rules
func (c *BitrueClient) ExchangeInfo() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/api/v1/exchangeInfo", nil, nil, false)
}

// Klines retrieves Kline/Candlestick data
func (c *BitrueClient) Klines(symbol, scale string, fromIdx, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("scale", scale)
	if fromIdx > 0 {
		params.Set("fromIdx", strconv.Itoa(fromIdx))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.doRequest(http.MethodGet, "/api/v1/market/kline", params, nil, false)
}

// Depth retrieves the order book
func (c *BitrueClient) Depth(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.doRequest(http.MethodGet, "/api/v1/depth", params, nil, false)
}

// Trades retrieves recent trades
func (c *BitrueClient) Trades(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.doRequest(http.MethodGet, "/api/v1/trades", params, nil, false)
}

// HistoricalTrades retrieves older trades
func (c *BitrueClient) HistoricalTrades(symbol string, limit int, fromId int64) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if fromId > 0 {
		params.Set("fromId", strconv.FormatInt(fromId, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/historicalTrades", params, nil, false)
}

// AggTrades retrieves compressed/aggregate trades
func (c *BitrueClient) AggTrades(params struct {
	Symbol    string
	FromID    int64
	StartTime int64
	EndTime   int64
	Limit     int
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	if params.FromID > 0 {
		query.Set("fromId", strconv.FormatInt(params.FromID, 10))
	}
	if params.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	return c.doRequest(http.MethodGet, "/api/v1/aggTrades", query, nil, false)
}

// Ticker24hr retrieves 24-hour price change statistics
func (c *BitrueClient) Ticker24hr(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return c.doRequest(http.MethodGet, "/api/v1/ticker/24hr", params, nil, false)
}

// TickerPrice retrieves the latest price for a symbol
func (c *BitrueClient) TickerPrice(symbol string) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	return c.doRequest(http.MethodGet, "/api/v1/ticker/price", params, nil, false)
}

// TickerBook retrieves the best price/qty on the order book
func (c *BitrueClient) TickerBook(symbol string) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	return c.doRequest(http.MethodGet, "/api/v1/ticker/bookTicker", params, nil, false)
}

// ETFNetValue retrieves ETF net value for a symbol
func (c *BitrueClient) ETFNetValue(symbol string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/v1/etf/net-value/%s", symbol)
	return c.doRequest(http.MethodGet, endpoint, nil, nil, true)
}

// NewOrder places a new spot order
func (c *BitrueClient) NewOrder(params struct {
	Symbol        string
	Side          string
	Type          string
	TimeInForce   string
	Quantity      float64
	Price         float64
	ClientOrderID string
	StopPrice     float64
	IcebergQty    float64
	RecvWindow    int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	query.Set("side", params.Side)
	query.Set("type", params.Type)
	query.Set("quantity", fmt.Sprintf("%f", params.Quantity))
	if params.TimeInForce != "" {
		query.Set("timeInForce", params.TimeInForce)
	}
	if params.Price > 0 {
		query.Set("price", fmt.Sprintf("%f", params.Price))
	}
	if params.ClientOrderID != "" {
		query.Set("newClientOrderId", params.ClientOrderID)
	}
	if params.StopPrice > 0 {
		query.Set("stopPrice", fmt.Sprintf("%f", params.StopPrice))
	}
	if params.IcebergQty > 0 {
		query.Set("icebergQty", fmt.Sprintf("%f", params.IcebergQty))
	}
	if params.RecvWindow > 0 {
		query.Set("recvWindow", strconv.FormatInt(params.RecvWindow, 10))
	}
	return c.doRequest(http.MethodPost, "/api/v1/order", query, nil, true)
}

// QueryOrder checks an order's status
func (c *BitrueClient) QueryOrder(params struct {
	Symbol            string
	OrderID           int64
	OrigClientOrderID string
	RecvWindow        int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	if params.OrderID > 0 {
		query.Set("orderId", strconv.FormatInt(params.OrderID, 10))
	}
	if params.OrigClientOrderID != "" {
		query.Set("origClientOrderId", params.OrigClientOrderID)
	}
	if params.RecvWindow > 0 {
		query.Set("recvWindow", strconv.FormatInt(params.RecvWindow, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/order", query, nil, true)
}

// CancelOrder cancels an active order
func (c *BitrueClient) CancelOrder(params struct {
	Symbol            string
	OrderID           int64
	OrigClientOrderID string
	NewClientOrderID  string
	RecvWindow        int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	if params.OrderID > 0 {
		query.Set("orderId", strconv.FormatInt(params.OrderID, 10))
	}
	if params.OrigClientOrderID != "" {
		query.Set("origClientOrderId", params.OrigClientOrderID)
	}
	if params.NewClientOrderID != "" {
		query.Set("newClientOrderId", params.NewClientOrderID)
	}
	if params.RecvWindow > 0 {
		query.Set("recvWindow", strconv.FormatInt(params.RecvWindow, 10))
	}
	return c.doRequest(http.MethodDelete, "/api/v1/order", query, nil, true)
}

// OpenOrders retrieves all open orders
func (c *BitrueClient) OpenOrders(symbol string, recvWindow int64) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if recvWindow > 0 {
		params.Set("recvWindow", strconv.FormatInt(recvWindow, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/openOrders", params, nil, true)
}

// AllOrders retrieves all account orders
func (c *BitrueClient) AllOrders(params struct {
	Symbol     string
	FromID     int64
	StartTime  int64
	EndTime    int64
	Limit      int
	RecvWindow int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	if params.FromID > 0 {
		query.Set("fromId", strconv.FormatInt(params.FromID, 10))
	}
	if params.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.RecvWindow > 0 {
		query.Set("recvWindow", strconv.FormatInt(params.RecvWindow, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/allOrders", query, nil, true)
}

// Account retrieves account information
func (c *BitrueClient) Account(recvWindow int64) ([]byte, error) {
	params := url.Values{}
	if recvWindow > 0 {
		params.Set("recvWindow", strconv.FormatInt(recvWindow, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/account", params, nil, true)
}

// MyTrades retrieves trade history
func (c *BitrueClient) MyTrades(params struct {
	Symbol     string
	StartTime  int64
	EndTime    int64
	FromID     int64
	Limit      int
	RecvWindow int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("symbol", params.Symbol)
	if params.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	if params.FromID > 0 {
		query.Set("fromId", strconv.FormatInt(params.FromID, 10))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.RecvWindow > 0 {
		query.Set("recvWindow", strconv.FormatInt(params.RecvWindow, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v2/myTrades", query, nil, true)
}

// WithdrawCommit commits a withdrawal request
func (c *BitrueClient) WithdrawCommit(params struct {
	Coin        string
	Amount      float64
	AddressTo   string
	ChainName   string
	AddressMark string
	AddrType    string
	Tag         string
}) ([]byte, error) {
	body := url.Values{}
	body.Set("coin", params.Coin)
	body.Set("amount", fmt.Sprintf("%f", params.Amount))
	body.Set("addressTo", params.AddressTo)
	body.Set("chainName", params.ChainName)
	if params.AddressMark != "" {
		body.Set("addressMark", params.AddressMark)
	}
	if params.AddrType != "" {
		body.Set("addrType", params.AddrType)
	}
	if params.Tag != "" {
		body.Set("tag", params.Tag)
	}
	return c.doRequest(http.MethodPost, "/api/v1/withdraw/commit", nil, body, true)
}

// WithdrawHistory queries withdrawal history
func (c *BitrueClient) WithdrawHistory(params struct {
	Coin      string
	Status    int
	Offset    int
	Limit     int
	StartTime int64
	EndTime   int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("coin", params.Coin)
	if params.Status > 0 {
		query.Set("status", strconv.Itoa(params.Status))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/withdraw/history", query, nil, true)
}

// DepositHistory queries deposit history
func (c *BitrueClient) DepositHistory(params struct {
	Coin      string
	Status    int
	Offset    int
	Limit     int
	StartTime int64
	EndTime   int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("coin", params.Coin)
	if params.Status > 0 {
		query.Set("status", strconv.Itoa(params.Status))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(params.StartTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	return c.doRequest(http.MethodGet, "/api/v1/deposit/history", query, nil, true)
}

// User Stream Endpoints

// CreateListenKey creates a new user data stream listenKey
func (c *BitrueClient) CreateListenKey() ([]byte, error) {
	return c.doRequest(http.MethodPost, "/api/v3/userDataStream", nil, nil, true)
}

// KeepAliveListenKey extends the validity of a listenKey
func (c *BitrueClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return c.doRequest(http.MethodPut, "/api/v3/userDataStream", params, nil, true)
}

// CloseListenKey closes a user data stream
func (c *BitrueClient) CloseListenKey(listenKey string) ([]byte, error) {
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return c.doRequest(http.MethodDelete, "/api/v3/userDataStream", params, nil, true)
}
