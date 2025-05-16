package main

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
	"time"

	"github.com/gorilla/websocket"
)

// APIError represents an error returned by the Bitrue API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// BaseClient handles common functionalities like HTTP requests and signatures
type BaseClient struct {
	apiKey     string
	secretKey  string
	httpClient *http.Client
}

// NewBaseClient creates a new BaseClient instance
func NewBaseClient(apiKey, secretKey string) *BaseClient {
	return &BaseClient{
		apiKey:     apiKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// DoRequest performs an HTTP request with optional signature
func (c *BaseClient) DoRequest(method, baseURL, endpoint string, params url.Values, body interface{}, signed bool) ([]byte, error) {
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

	if method == "GET" && params != nil {
		u.RawQuery = params.Encode()
	}

	path := u.Path
	if method == "GET" && u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	timestamp := time.Now().UnixMilli()
	signingString := fmt.Sprintf("%d%s%s%s", timestamp, strings.ToUpper(method), path, string(bodyBytes))
	var signature string
	if signed {
		signature = c.GenerateSignature(signingString)
	}

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	if method == "POST" || method == "PUT" || method == "DELETE" {
		req.Header.Set("Content-Type", "application/json")
		if bodyBytes != nil {
			req.Body = http.NoBody
			req.ContentLength = int64(len(bodyBytes))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(bodyBytes)), nil
			}
		}
	}

	req.Header.Set("X-CH-APIKEY", c.apiKey)
	if signed {
		req.Header.Set("X-CH-TS", strconv.FormatInt(timestamp, 10))
		req.Header.Set("X-CH-SIGN", signature)
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
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			return nil, fmt.Errorf("API error %d: %s", apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GenerateSignature creates an HMAC SHA256 signature
func (c *BaseClient) GenerateSignature(data string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// WebSocketMessageHandler defines the callback for WebSocket messages
type WebSocketMessageHandler func([]byte)

// WebSocketClient handles WebSocket connections
type WebSocketClient struct {
	conn          *websocket.Conn
	url           string
	subscriptions map[string]WebSocketMessageHandler
	callback      WebSocketMessageHandler
}

// NewWebSocketClient creates a new WebSocketClient
func NewWebSocketClient(url string) *WebSocketClient {
	return &WebSocketClient{
		url:           url,
		subscriptions: make(map[string]WebSocketMessageHandler),
	}
}

// Connect establishes a WebSocket connection
func (w *WebSocketClient) Connect(apiKey string) error {
	headers := http.Header{}
	if apiKey != "" {
		headers.Set("X-CH-APIKEY", apiKey)
	}
	conn, _, err := websocket.DefaultDialer.Dial(w.url, headers)
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %v", err)
	}
	w.conn = conn
	return nil
}

// Close terminates the WebSocket connection
func (w *WebSocketClient) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// Subscribe adds a subscription to a channel
func (w *WebSocketClient) Subscribe(channel string, callback WebSocketMessageHandler) {
	w.subscriptions[channel] = callback
}

// SetCallback sets a single callback for user data streams
func (w *WebSocketClient) SetCallback(callback WebSocketMessageHandler) {
	w.callback = callback
}

// ReadMessages processes incoming WebSocket messages
func (w *WebSocketClient) ReadMessages(multiChannel bool) {
	defer w.Close()
	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			fmt.Printf("WebSocket read error: %v\n", err)
			return
		}
		if string(message) == "pong" {
			continue
		}
		if multiChannel {
			var msg struct {
				Channel string `json:"channel"`
			}
			if err := json.Unmarshal(message, &msg); err == nil {
				if callback, ok := w.subscriptions[msg.Channel]; ok {
					callback(message)
				}
			}
		} else if w.callback != nil {
			w.callback(message)
		}
	}
}

// FuturesClient for USD-Margin Futures trading
type FuturesClient struct {
	*BaseClient
	baseURL     string
	wsMarketURL string
	wsUserURL   string
	marketWS    *WebSocketClient
	userWS      *WebSocketClient
}

// NewFuturesClient creates a new FuturesClient
func NewFuturesClient(apiKey, secretKey string) *FuturesClient {
	return &FuturesClient{
		BaseClient:  NewBaseClient(apiKey, secretKey),
		baseURL:     "https://fapi.bitrue.com",
		wsMarketURL: "wss://fmarket-ws.bitrue.com/kline-api/ws",
		wsUserURL:   "wss://fuser-ws.bitrue.com",
	}
}

// Ping tests connectivity
func (c *FuturesClient) Ping() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/ping", nil, nil, false)
}

// ServerTime retrieves server time
func (c *FuturesClient) ServerTime() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/time", nil, nil, false)
}

// Contracts retrieves contract information
func (c *FuturesClient) Contracts() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/contracts", nil, nil, false)
}

// Ticker retrieves ticker data
func (c *FuturesClient) Ticker(contractName string) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/ticker", params, nil, false)
}

// Depth retrieves order book depth
func (c *FuturesClient) Depth(contractName string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/depth", params, nil, false)
}

// Klines retrieves candlestick data
func (c *FuturesClient) Klines(contractName, interval string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("contractName", contractName)
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v1/klines", params, nil, false)
}

// NewOrder places a new order
func (c *FuturesClient) NewOrder(params interface{}) ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/fapi/v2/order", nil, params, true)
}

// QueryOrder queries an order
func (c *FuturesClient) QueryOrder(orderId, contractName string) ([]byte, error) {
	params := url.Values{}
	params.Set("orderId", orderId)
	params.Set("contractName", contractName)
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/order", params, nil, true)
}

// CancelOrder cancels an order
func (c *FuturesClient) CancelOrder(params interface{}) ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/fapi/v2/cancel", nil, params, true)
}

// OpenOrders retrieves open orders
func (c *FuturesClient) OpenOrders(contractName string) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/openOrders", params, nil, true)
}

// Account retrieves account information
func (c *FuturesClient) Account() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/account", nil, nil, true)
}

// MyTrades retrieves trade history
func (c *FuturesClient) MyTrades(contractName string, limit int) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/myTrades", params, nil, true)
}

// LeverageBracket retrieves leverage bracket
func (c *FuturesClient) LeverageBracket(contractName string) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/leverageBracket", params, nil, true)
}

// CommissionRate retrieves commission rate
func (c *FuturesClient) CommissionRate(contractName string) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/commissionRate", params, nil, true)
}

// FuturesTransfer performs a futures transfer
func (c *FuturesClient) FuturesTransfer(params interface{}) ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/fapi/v2/futures_transfer", nil, params, true)
}

// FuturesTransferHistory retrieves transfer history
func (c *FuturesClient) FuturesTransferHistory(startTime, endTime int64, limit int) ([]byte, error) {
	params := url.Values{}
	if startTime > 0 {
		params.Set("startTime", strconv.FormatInt(startTime, 10))
	}
	if endTime > 0 {
		params.Set("endTime", strconv.FormatInt(endTime, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/futures_transfer_history", params, nil, true)
}

// ForceOrdersHistory retrieves forced orders history
func (c *FuturesClient) ForceOrdersHistory(contractName string, limit int) ([]byte, error) {
	params := url.Values{}
	if contractName != "" {
		params.Set("contractName", contractName)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/fapi/v2/forceOrdersHistory", params, nil, true)
}

// CreateListenKey creates a user data stream listen key
func (c *FuturesClient) CreateListenKey() ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/user_stream/api/v1/listenKey", nil, nil, true)
}

// KeepAliveListenKey keeps the listen key alive
func (c *FuturesClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.DoRequest("PUT", c.baseURL, endpoint, nil, nil, true)
}

// CloseListenKey closes the listen key
func (c *FuturesClient) CloseListenKey(listenKey string) ([]byte, error) {
	endpoint := fmt.Sprintf("/user_stream/api/v1/listenKey/%s", listenKey)
	return c.DoRequest("DELETE", c.baseURL, endpoint, nil, nil, true)
}

// StartMarketWebSocket starts the market data WebSocket for futures trading
func (c *FuturesClient) StartMarketWebSocket() error {
	if c.marketWS != nil {
		return nil
	}
	c.marketWS = NewWebSocketClient(c.wsMarketURL)
	if err := c.marketWS.Connect(""); err != nil {
		return err
	}
	go c.marketWS.ReadMessages(true)
	return nil
}

// SubscribeDepth subscribes to depth updates for futures
func (c *FuturesClient) SubscribeDepth(contractName string, callback WebSocketMessageHandler) error {
	if c.marketWS == nil {
		return fmt.Errorf("market WebSocket not started")
	}
	channel := fmt.Sprintf("market_%s_depth_step0", strings.ToLower(contractName))
	msg := map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	if err := c.marketWS.conn.WriteJSON(msg); err != nil {
		return err
	}
	c.marketWS.Subscribe(channel, callback)
	return nil
}

// StartUserWebSocket starts the user data WebSocket specifically for futures trading
// Note: This method is distinct from SpotClient's StartUserWebSocket due to different WebSocket URLs and trading contexts
func (c *FuturesClient) StartUserWebSocket(listenKey string, callback WebSocketMessageHandler) error {
	if c.userWS != nil {
		return nil
	}
	wsURL := fmt.Sprintf("%s/stream?listenKey=%s", c.wsUserURL, listenKey)
	c.userWS = NewWebSocketClient(wsURL)
	if err := c.userWS.Connect(c.apiKey); err != nil {
		return err
	}
	c.userWS.SetCallback(callback)
	go c.userWS.ReadMessages(false)
	return nil
}

// SpotClient for Spot trading
type SpotClient struct {
	*BaseClient
	baseURL     string
	wsMarketURL string
	wsUserURL   string
	marketWS    *WebSocketClient
	userWS      *WebSocketClient
}

// NewSpotClient creates a new SpotClient
func NewSpotClient(apiKey, secretKey string) *SpotClient {
	return &SpotClient{
		BaseClient:  NewBaseClient(apiKey, secretKey),
		baseURL:     "https://open.bitrue.com",
		wsMarketURL: "wss://market-ws.bitrue.com/kline-api/ws",
		wsUserURL:   "wss://user-ws.bitrue.com",
	}
}

// Ping tests connectivity for spot trading
func (c *SpotClient) Ping() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/ping", nil, nil, false)
}

// ServerTime retrieves server time for spot trading
func (c *SpotClient) ServerTime() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/time", nil, nil, false)
}

// ExchangeInfo retrieves exchange information for spot trading
func (c *SpotClient) ExchangeInfo() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/exchangeInfo", nil, nil, false)
}

// Trades retrieves recent trades for spot trading
func (c *SpotClient) Trades(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/trades", params, nil, false)
}

// HistoricalTrades retrieves historical trades for spot trading
func (c *SpotClient) HistoricalTrades(symbol string, limit int, fromId int64) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if fromId > 0 {
		params.Set("fromId", strconv.FormatInt(fromId, 10))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/historicalTrades", params, nil, false)
}

// AggTrades retrieves aggregated trades for spot trading
func (c *SpotClient) AggTrades(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/aggTrades", params, nil, false)
}

// Klines retrieves candlestick data for spot trading
func (c *SpotClient) Klines(symbol, interval string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/klines", params, nil, false)
}

// Ticker24hr retrieves 24hr ticker data for spot trading
func (c *SpotClient) Ticker24hr(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/ticker/24hr", params, nil, false)
}

// TickerPrice retrieves ticker price for spot trading
func (c *SpotClient) TickerPrice(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/ticker/price", params, nil, false)
}

// TickerBook retrieves order book ticker for spot trading
func (c *SpotClient) TickerBook(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/ticker/bookTicker", params, nil, false)
}

// Depth retrieves order book depth for spot trading
func (c *SpotClient) Depth(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/depth", params, nil, false)
}

// NewOrder places a new order for spot trading
func (c *SpotClient) NewOrder(params interface{}) ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/api/v1/order", nil, params, true)
}

// QueryOrder queries an order for spot trading
func (c *SpotClient) QueryOrder(params interface{}) ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/order", params, nil, true)
}

// CancelOrder cancels an order for spot trading
func (c *SpotClient) CancelOrder(params interface{}) ([]byte, error) {
	return c.DoRequest("DELETE", c.baseURL, "/api/v1/order", params, nil, true)
}

// OpenOrders retrieves open orders for spot trading
func (c *SpotClient) OpenOrders(symbol string) ([]byte, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/openOrders", params, nil, true)
}

// Account retrieves account information for spot trading
func (c *SpotClient) Account() ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/account", nil, nil, true)
}

// MyTrades retrieves trade history for spot trading
func (c *SpotClient) MyTrades(symbol string, limit int) ([]byte, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return c.DoRequest("GET", c.baseURL, "/api/v1/myTrades", params, nil, true)
}

// WithdrawCommit initiates a withdrawal for spot trading
func (c *SpotClient) WithdrawCommit(params interface{}) ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/api/v1/withdraw/commit", nil, params, true)
}

// WithdrawHistory retrieves withdrawal history for spot trading
func (c *SpotClient) WithdrawHistory(params interface{}) ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/withdraw/history", params, nil, true)
}

// DepositHistory retrieves deposit history for spot trading
func (c *SpotClient) DepositHistory(params interface{}) ([]byte, error) {
	return c.DoRequest("GET", c.baseURL, "/api/v1/deposit/history", params, nil, true)
}

// CreateListenKey creates a user data stream listen key for spot trading
func (c *SpotClient) CreateListenKey() ([]byte, error) {
	return c.DoRequest("POST", c.baseURL, "/api/v1/userDataStream", nil, nil, true)
}

// KeepAliveListenKey keeps the listen key alive for spot trading
func (c *SpotClient) KeepAliveListenKey(listenKey string) ([]byte, error) {
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return c.DoRequest("PUT", c.baseURL, "/api/v1/userDataStream", params, nil, true)
}

// CloseListenKey closes the listen key for spot trading
func (c *SpotClient) CloseListenKey(listenKey string) ([]byte, error) {
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return c.DoRequest("DELETE", c.baseURL, "/api/v1/userDataStream", params, nil, true)
}

// StartMarketWebSocket starts the market data WebSocket for spot trading
func (c *SpotClient) StartMarketWebSocket() error {
	if c.marketWS != nil {
		return nil
	}
	c.marketWS = NewWebSocketClient(c.wsMarketURL)
	if err := c.marketWS.Connect(""); err != nil {
		return err
	}
	go c.marketWS.ReadMessages(true)
	return nil
}

// SubscribeDepth subscribes to depth updates for spot trading
func (c *SpotClient) SubscribeDepth(symbol string, callback WebSocketMessageHandler) error {
	if c.marketWS == nil {
		return fmt.Errorf("market WebSocket not started")
	}
	channel := fmt.Sprintf("market_%s_depth_step0", strings.ToLower(symbol))
	msg := map[string]interface{}{
		"event": "sub",
		"params": map[string]interface{}{
			"channel": channel,
			"cb_id":   "",
		},
	}
	if err := c.marketWS.conn.WriteJSON(msg); err != nil {
		return err
	}
	c.marketWS.Subscribe(channel, callback)
	return nil
}

// StartUserWebSocket starts the user data WebSocket specifically for spot trading
// Note: This method is distinct from FuturesClient's StartUserWebSocket due to different WebSocket URLs and trading contexts
func (c *SpotClient) StartUserWebSocket(listenKey string, callback WebSocketMessageHandler) error {
	if c.userWS != nil {
		return nil
	}
	wsURL := fmt.Sprintf("%s/stream?listenKey=%s", c.wsUserURL, listenKey)
	c.userWS = NewWebSocketClient(wsURL)
	if err := c.userWS.Connect(c.apiKey); err != nil {
		return err
	}
	c.userWS.SetCallback(callback)
	go c.userWS.ReadMessages(false)
	return nil
}

func main() {
	// Example usage for FuturesClient
	futuresClient := NewFuturesClient("your-api-key", "your-secret-key")
	resp, err := futuresClient.Ping()
	if err != nil {
		fmt.Printf("Futures Ping Error: %v\n", err)
	} else {
		fmt.Printf("Futures Ping Response: %s\n", resp)
	}

	// Example usage for SpotClient
	spotClient := NewSpotClient("your-api-key", "your-secret-key")
	resp, err = spotClient.Ping()
	if err != nil {
		fmt.Printf("Spot Ping Error: %v\n", err)
	} else {
		fmt.Printf("Spot Ping Response: %s\n", resp)
	}
}
