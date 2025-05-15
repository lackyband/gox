package bitrueFutures

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

// BitrueClient represents a client for the Bitrue Futures API
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
		baseURL:        "https://fapi.bitrue.com",
		wsMarketURL:    "wss://fmarket-ws.bitrue.com/kline-api/ws",
		wsUserURL:      "wss://fapiws.bitrue.com",
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
func (c *BitrueClient) SubscribeDepth(contractName string, callback WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_depth_step0"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeKline subscribes to the kline websocket channel
func (c *BitrueClient) SubscribeKline(contractName, interval string, callback WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_kline_$interval"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	channel = strings.Replace(channel, "$interval", interval, -1)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeTrade subscribes to the trade websocket channel
func (c *BitrueClient) SubscribeTrade(contractName string, callback WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_trade_ticker"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	msg := WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeUserData subscribes to the user data websocket channel
func (c *BitrueClient) SubscribeUserData(listenKey string, callback WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	// Futures user data streams are established on connection
	return nil
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
			"cb_id":   "",
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
			"cb_id":   "",
		},
	}
	return client.conn.WriteJSON(msg)
}

// RequestWebSocketData requests historical data from a WebSocket channel
func (c *BitrueClient) RequestWebSocketData(channel, symbol, interval string, top int, endIdx string, isMarket bool) error {
	client := c.wsMarketClient
	if !isMarket {
		return fmt.Errorf("historical data requests are only supported for market streams")
	}
	if client == nil || client.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}

	if interval != "" {
		channel = strings.Replace(channel, "$interval", interval, -1)
	}
	channel = strings.Replace(channel, "$symbol", strings.ToLower(symbol), -1)

	params := map[string]interface{}{
		"channel": channel,
		"cb_id":   "",
	}
	if top > 0 {
		params["top"] = top
	}
	if endIdx != "" {
		params["endIdx"] = endIdx
	}

	msg := WebSocketMessage{
		Event:  "req",
		Params: params,
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
func (c *BitrueClient) doRequest(method, endpoint string, params url.Values, body interface{}, signed bool) ([]byte, error) {
	u, err := url.Parse(c.baseURL + endpoint)
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
		signature := c.generateSignature(timestamp, strings.ToUpper(method), path, string(bodyBytes))
		req.Header.Set("X-CH-SIGN", signature)
		req.Header.Set("X-CH-TS", strconv.FormatInt(timestamp, 10))
	}

	resp, err := c.httpClient.Do(req)
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
func (c *BitrueClient) generateSignature(timestamp int64, method, path, body string) string {
	message := fmt.Sprintf("%d%s%s%s", timestamp, method, path, body)
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// REST API Endpoints

// Ping tests connectivity to the API
func (c *BitrueClient) Ping() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/fapi/v1/ping", nil, nil, false)
}

// ServerTime retrieves the server time
func (c *BitrueClient) ServerTime() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/fapi/v1/time", nil, nil, false)
}

// Contracts retrieves current open contracts
func (c *BitrueClient) Contracts() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/fapi/v1/contracts", nil, nil, false)
}

// Depth retrieves the order book
func (c *BitrueClient) Depth(contractName string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.doRequest(http.MethodGet, "/fapi/v1/depth", params, nil, false)
}

// Ticker retrieves 24-hour price statistics
func (c *BitrueClient) Ticker(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return c.doRequest(http.MethodGet, "/fapi/v1/ticker", params, nil, false)
}

// Klines retrieves Kline/Candlestick data
func (c *BitrueClient) Klines(contractName, interval string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.doRequest(http.MethodGet, "/fapi/v1/klines", params, nil, false)
}

// MyTrades retrieves trade history
func (c *BitrueClient) MyTrades(params struct {
	ContractName string
	FromID       int64
	Limit        int
	StartTime    int64
	EndTime      int64
}) ([]byte, error) {
	query := url.Values{}
	query.Set("contractName", params.ContractName)
	if params.FromID > 0 {
		query.Set("fromId", strconv.FormatInt(params.FromID, 10))
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
	return c.doRequest(http.MethodGet, "/fapi/v2/myTrades", query, nil, true)
}

// ModifyPositionMargin modifies isolated position margin
func (c *BitrueClient) ModifyPositionMargin(contractName string, positionMargin float64) ([]byte, error) {
	body := map[string]interface{}{
		"contractName":   contractName,
		"positionMargin": positionMargin,
	}
	return c.doRequest(http.MethodPost, "/fapi/v2/positionMargin", nil, body, true)
}

// ChangeLeverage changes initial leverage
func (c *BitrueClient) ChangeLeverage(contractName string, leverage int) ([]byte, error) {
	body := map[string]interface{}{
		"contractName": contractName,
		"leverage":     leverage,
	}
	return c.doRequest(http.MethodPost, "/fapi/v2/level_edit", nil, body, true)
}

// OpenOrders retrieves all open orders
func (c *BitrueClient) OpenOrders(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return c.doRequest(http.MethodGet, "/fapi/v2/openOrders", params, nil, true)
}

// CancelOrder cancels an order
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
	return c.doRequest(http.MethodPost, "/fapi/v2/cancel", nil, body, true)
}

// QueryOrder queries an order
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
	return c.doRequest(http.MethodGet, "/fapi/v2/order", query, nil, true)
}

// NewOrder places a new order
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
	return c.doRequest(http.MethodPost, "/fapi/v2/order", nil, body, true)
}

// Account retrieves account information
func (c *BitrueClient) Account() ([]byte, error) {
	return c.doRequest(http.MethodGet, "/fapi/v2/account", nil, nil, true)
}

// LeverageBracket retrieves leverage brackets
func (c *BitrueClient) LeverageBracket(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return c.doRequest(http.MethodGet, "/fapi/v2/leverageBracket", params, nil, true)
}

// CommissionRate retrieves commission rates
func (c *BitrueClient) CommissionRate(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return c.doRequest(http.MethodGet, "/fapi/v2/commissionRate", params, nil, true)
}

// FuturesTransfer transfers funds between wallet and futures account
func (c *BitrueClient) FuturesTransfer(coinSymbol string, amount float64, transferType, unionID string) ([]byte, error) {
	body := map[string]interface{}{
		"coinSymbol":   coinSymbol,
		"amount":       amount,
		"transferType": transferType,
	}
	if unionID != "" {
		body["unionId"] = unionID
	}
	return c.doRequest(http.MethodPost, "/fapi/v2/futures_transfer", nil, body, true)
}

// FuturesTransferHistory retrieves transfer history
func (c *BitrueClient) FuturesTransferHistory(params struct {
	CoinSymbol   string
	BeginTime    int64
	EndTime      int64
	TransferType string
	Page         int
	Limit        int
}) ([]byte, error) {
	query := url.Values{}
	if params.CoinSymbol != "" {
		query.Set("coinSymbol", params.CoinSymbol)
	}
	if params.BeginTime > 0 {
		query.Set("beginTime", strconv.FormatInt(params.BeginTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	query.Set("transferType", params.TransferType)
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	return c.doRequest(http.MethodGet, "/fapi/v2/futures_transfer_history", query, nil, true)
}

// ForceOrdersHistory retrieves forced liquidation order history
func (c *BitrueClient) ForceOrdersHistory(params struct {
	ContractName  string
	BeginTime     int64
	EndTime       int64
	AutoCloseType string
	Page          int
	Limit         int
}) ([]byte, error) {
	query := url.Values{}
	query.Set("contractName", params.ContractName)
	if params.BeginTime > 0 {
		query.Set("beginTime", strconv.FormatInt(params.BeginTime, 10))
	}
	if params.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(params.EndTime, 10))
	}
	if params.AutoCloseType != "" {
		query.Set("autoCloseType", params.AutoCloseType)
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	return c.doRequest(http.MethodGet, "/fapi/v2/forceOrdersHistory", query, nil, true)
}

// User Stream Endpoints

// CreateListenKey creates a new user data stream listenKey
func (c *BitrueClient) CreateListenKey() ([]byte, error) {
	return c.doRequest(http.MethodPost, "/user_stream/api/v1/listenKey", nil, nil, true)
}

// KeepAliveListenKey extends the validity of a listenKey
func (c *BitrueClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.doRequest(http.MethodPut, endpoint, nil, nil, true)
}

// CloseListenKey closes a user data stream
func (c *BitrueClient) CloseListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.doRequest(http.MethodDelete, endpoint, nil, nil, true)
}
