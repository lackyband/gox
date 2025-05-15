package bitrueFutures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lackyband/gox/common"
)

// BitrueClient represents a client for the Bitrue Futures API (REST and WebSocket)
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

// SubscribeDepth subscribes to the depth websocket channel for a contract
func (c *BitrueClient) SubscribeDepth(contractName string, callback common.WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_depth_step0"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	msg := common.WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeKline subscribes to the kline websocket channel for a contract and interval
func (c *BitrueClient) SubscribeKline(contractName, interval string, callback common.WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_kline_$interval"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	channel = strings.Replace(channel, "$interval", interval, -1)
	msg := common.WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeTrade subscribes to the trade websocket channel for a contract
func (c *BitrueClient) SubscribeTrade(contractName string, callback common.WebSocketMessageHandler) error {
	if c.wsMarketClient == nil || c.wsMarketClient.conn == nil {
		return fmt.Errorf("WebSocket client not started")
	}
	c.wsMarketClient.SetCallback(callback)
	channel := "market_$symbol_trade_ticker"
	channel = strings.Replace(channel, "$symbol", strings.ToLower(contractName), -1)
	msg := common.WebSocketMessage{
		Event: "sub",
		Params: map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	return c.wsMarketClient.conn.WriteJSON(msg)
}

// SubscribeUserData subscribes to the user data websocket channel (requires listenKey)
func (c *BitrueClient) SubscribeUserData(listenKey string, callback common.WebSocketMessageHandler) error {
	if c.wsUserClient == nil || c.wsUserClient.conn == nil {
		return fmt.Errorf("User WebSocket client not started")
	}
	c.wsUserClient.SetCallback(callback)
	// Bitrue user data streams are established on connection, no extra sub needed
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

	msg := common.WebSocketMessage{
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

	msg := common.WebSocketMessage{
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

	msg := common.WebSocketMessage{
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

// REST API Endpoints

// Ping tests connectivity to the API
func (c *BitrueClient) Ping() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/ping", nil, nil, c.apiKey, c.secretKey, false)
}

// ServerTime retrieves the server time
func (c *BitrueClient) ServerTime() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/time", nil, nil, c.apiKey, c.secretKey, false)
}

// Contracts retrieves current open contracts
func (c *BitrueClient) Contracts() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/contracts", nil, nil, c.apiKey, c.secretKey, false)
}

// Depth retrieves the order book
func (c *BitrueClient) Depth(contractName string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/depth", params, nil, c.apiKey, c.secretKey, false)
}

// Ticker retrieves 24-hour price statistics
func (c *BitrueClient) Ticker(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/ticker", params, nil, c.apiKey, c.secretKey, false)
}

// Klines retrieves Kline/Candlestick data
func (c *BitrueClient) Klines(contractName, interval string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v1/klines", params, nil, c.apiKey, c.secretKey, false)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/myTrades", query, nil, c.apiKey, c.secretKey, true)
}

// ModifyPositionMargin modifies isolated position margin
func (c *BitrueClient) ModifyPositionMargin(contractName string, positionMargin float64) ([]byte, error) {
	body := url.Values{}
	body.Set("contractName", contractName)
	body.Set("positionMargin", fmt.Sprintf("%f", positionMargin))
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/fapi/v2/positionMargin", nil, body, c.apiKey, c.secretKey, true)
}

// ChangeLeverage changes initial leverage
func (c *BitrueClient) ChangeLeverage(contractName string, leverage int) ([]byte, error) {
	body := url.Values{}
	body.Set("contractName", contractName)
	body.Set("leverage", strconv.Itoa(leverage))
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/fapi/v2/level_edit", nil, body, c.apiKey, c.secretKey, true)
}

// OpenOrders retrieves all open orders
func (c *BitrueClient) OpenOrders(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/openOrders", params, nil, c.apiKey, c.secretKey, true)
}

// CancelOrder cancels an order
func (c *BitrueClient) CancelOrder(params struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}) ([]byte, error) {
	body := url.Values{}
	body.Set("contractName", params.ContractName)
	body.Set("conditionOrder", strconv.FormatBool(params.ConditionOrder))
	if params.ClientOrderID != "" {
		body.Set("clientOrderId", params.ClientOrderID)
	}
	if params.OrderID > 0 {
		body.Set("orderId", strconv.FormatInt(params.OrderID, 10))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/fapi/v2/cancel", nil, body, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/order", query, nil, c.apiKey, c.secretKey, true)
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
	body := url.Values{}
	body.Set("contractName", params.ContractName)
	body.Set("side", params.Side)
	body.Set("type", params.Type)
	body.Set("positionType", strconv.Itoa(params.PositionType))
	body.Set("open", params.Open)
	body.Set("volume", fmt.Sprintf("%f", params.Volume))
	body.Set("amount", fmt.Sprintf("%f", params.Amount))
	body.Set("price", fmt.Sprintf("%f", params.Price))
	body.Set("leverage", strconv.Itoa(params.Leverage))
	body.Set("conditionOrder", strconv.FormatBool(params.ConditionOrder))
	if params.ClientOrderID != "" {
		body.Set("clientOrderId", params.ClientOrderID)
	}
	if params.TriggerOrderType > 0 {
		body.Set("triggerOrderType", strconv.Itoa(params.TriggerOrderType))
	}
	if params.TriggerType > 0 {
		body.Set("triggerType", strconv.Itoa(params.TriggerType))
	}
	if params.TriggerPriceType > 0 {
		body.Set("triggerPriceType", strconv.Itoa(params.TriggerPriceType))
	}
	if params.TriggerPrice > 0 {
		body.Set("triggerPrice", fmt.Sprintf("%f", params.TriggerPrice))
	}
	if params.PositionID > 0 {
		body.Set("positionId", strconv.Itoa(params.PositionID))
	}
	if len(params.TriggerOrderParams) > 0 {
		triggerParams, _ := json.Marshal(params.TriggerOrderParams)
		body.Set("triggerOrderCreateParams", string(triggerParams))
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/fapi/v2/order", nil, body, c.apiKey, c.secretKey, true)
}

// Account retrieves account information
func (c *BitrueClient) Account() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/account", nil, nil, c.apiKey, c.secretKey, true)
}

// LeverageBracket retrieves leverage brackets
func (c *BitrueClient) LeverageBracket(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/leverageBracket", params, nil, c.apiKey, c.secretKey, true)
}

// CommissionRate retrieves commission rates
func (c *BitrueClient) CommissionRate(contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/commissionRate", params, nil, c.apiKey, c.secretKey, true)
}

// FuturesTransfer transfers funds between wallet and futures account
func (c *BitrueClient) FuturesTransfer(coinSymbol string, amount float64, transferType, unionID string) ([]byte, error) {
	body := url.Values{}
	body.Set("coinSymbol", coinSymbol)
	body.Set("amount", fmt.Sprintf("%f", amount))
	body.Set("transferType", transferType)
	if unionID != "" {
		body.Set("unionId", unionID)
	}
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/fapi/v2/futures_transfer", nil, body, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/futures_transfer_history", query, nil, c.apiKey, c.secretKey, true)
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
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodGet, "/fapi/v2/forceOrdersHistory", query, nil, c.apiKey, c.secretKey, true)
}

// User Stream Endpoints

// CreateListenKey creates a new user data stream listenKey
func (c *BitrueClient) CreateListenKey() ([]byte, error) {
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPost, "/user_stream/api/v1/listenKey", nil, nil, c.apiKey, c.secretKey, true)
}

// KeepAliveListenKey extends the validity of a listenKey
func (c *BitrueClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodPut, endpoint, nil, nil, c.apiKey, c.secretKey, true)
}

// CloseListenKey closes a user data stream
func (c *BitrueClient) CloseListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return common.DoRequest(c.httpClient, c.baseURL, http.MethodDelete, endpoint, nil, nil, c.apiKey, c.secretKey, true)
}
