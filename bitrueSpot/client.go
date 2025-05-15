package bitrueSpot

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lackyband/gox/common"
)

// BitrueClient represents a client for the Bitrue Spot API (REST and WebSocket)
type BitrueClient struct {
	baseURL        string
	wsMarketURL    string
	wsUserURL      string
	apiKey         string
	secretKey      string
	httpClient     *http.Client
	wsMarketClient *common.WebSocketClient
	wsUserClient   *common.WebSocketClient
	wsMarketMutex  sync.Mutex
	wsUserMutex    sync.Mutex
}

// NewBitrueClient initializes a new BitrueClient
func NewBitrueClient(apiKey, secretKey string) *BitrueClient {
	return &BitrueClient{
		baseURL:        "https://openapi.bitrue.com",
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

	c.wsMarketClient = common.NewWebSocketClient(c.wsMarketURL)
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
	c.wsUserClient = common.NewWebSocketClient(wsURL)
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

// SubscribeDepth subscribes to the depth websocket channel for a symbol
func (c *BitrueClient) SubscribeDepth(symbol string, callback common.WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_simple_depth_step0"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(symbol), -1)
	msg := common.WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   strings.ToLower(symbol),
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeUserOrderUpdate subscribes to the user order update websocket channel
func (c *BitrueClient) SubscribeUserOrderUpdate(callback common.WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	msg := common.WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": "user_order_update",
		},
	}
	return c.wsUserClient.conn.WriteJSON(msg)
}

// SubscribeUserBalanceUpdate subscribes to the user balance update websocket channel
func (c *BitrueClient) SubscribeUserBalanceUpdate(callback common.WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	msg := common.WebSocketMessage{
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

	msg := common.WebSocketMessage{
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

	msg := common.WebSocketMessage{
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

// REST API Endpoints

// Ping tests connectivity to the API
func (c *BitrueClient) Ping() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/ping", nil, nil, c.apiKey, c.secretKey, false)
}

// ServerTime retrieves the server time
func (c *BitrueClient) ServerTime() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/time", nil, nil, c.apiKey, c.secretKey, false)
}

// ExchangeInfo retrieves trading pairs and rules
func (c *BitrueClient) ExchangeInfo() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/exchangeInfo", nil, nil, c.apiKey, c.secretKey, false)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/market/kline", params, nil, c.apiKey, c.secretKey, false)
}

// Depth retrieves the order book
func (c *BitrueClient) Depth(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/depth", params, nil, c.apiKey, c.secretKey, false)
}

// Trades retrieves recent trades
func (c *BitrueClient) Trades(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/trades", params, nil, c.apiKey, c.secretKey, false)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/historicalTrades", params, nil, c.apiKey, c.secretKey, false)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/aggTrades", query, nil, c.apiKey, c.secretKey, false)
}

// Ticker24hr retrieves 24-hour price change statistics
func (c *BitrueClient) Ticker24hr(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/ticker/24hr", params, nil, c.apiKey, c.secretKey, false)
}

// TickerPrice retrieves the latest price for a symbol
func (c *BitrueClient) TickerPrice(symbol string) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/ticker/price", params, nil, c.apiKey, c.secretKey, false)
}

// TickerBook retrieves the best price/qty on the order book
func (c *BitrueClient) TickerBook(symbol string) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/ticker/bookTicker", params, nil, c.apiKey, c.secretKey, false)
}

// ETFNetValue retrieves ETF net value for a symbol
func (c *BitrueClient) ETFNetValue(symbol string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/v1/etf/net-value/%s", symbol)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, endpoint, nil, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/api/v1/order", query, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/order", query, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodDelete, "/api/v1/order", query, nil, c.apiKey, c.secretKey, true)
}

// OpenOrders retrieves all open orders
func (c *BitrueClient) OpenOrders(symbol string, recvWindow int64) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if recvWindow > 0 {
		params.Set("recvWindow", strconv.FormatInt(recvWindow, 10))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/openOrders", params, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/allOrders", query, nil, c.apiKey, c.secretKey, true)
}

// Account retrieves account information
func (c *BitrueClient) Account(recvWindow int64) ([]byte, error) {
	params := url.Values{}
	if recvWindow > 0 {
		params.Set("recvWindow", strconv.FormatInt(recvWindow, 10))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/account", params, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v2/myTrades", query, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/api/v1/withdraw/commit", nil, body, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/withdraw/history", query, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/api/v1/deposit/history", query, nil, c.apiKey, c.secretKey, true)
}

// User Stream Endpoints

// CreateListenKey creates a new user data stream listenKey
func (c *BitrueClient) CreateListenKey() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/poseidon/api/v1/listenKey", nil, nil, c.apiKey, c.secretKey, true)
}

// KeepAliveListenKey extends the validity of a listenKey
func (c *BitrueClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/poseidon/api/v1/listenKey/%s", listenKey)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPut, endpoint, nil, nil, c.apiKey, c.secretKey, true)
}

// CloseListenKey closes a user data stream
func (c *BitrueClient) CloseListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/poseidon/api/v1/listenKey/%s", listenKey)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodDelete, endpoint, nil, nil, c.apiKey, c.secretKey, true)
}
