package main

import (
	"encoding/json"
	"fmt"
	"log"

	bitrue "github.com/lackyband/gox/bitrue"
	wsmanager "github.com/lackyband/gox/wsutil/wsmanager"
)

// API Credentials (Replace with actual keys)
const (
	apiKey    = "your-api-key"
	secretKey = "your-secret-key"
)

// Callback functions for WebSocket messages
func depthCallback(message []byte) {
	fmt.Println("Depth Update:", string(message))
}

func klineCallback(message []byte) {
	fmt.Println("Kline Update:", string(message))
}

func userDataCallback(message []byte) {
	fmt.Println("User Data Update:", string(message))
}

func main() {
	// Initialize clients for each trading type
	futuresClient := bitrue.NewFuturesClient(apiKey, secretKey)
	copyTradeClient := bitrue.NewCopyTradeClient(apiKey, secretKey)
	mFuturesClient := bitrue.NewMFuturesClient(apiKey, secretKey)
	spotClient := bitrue.NewSpotClient(apiKey, secretKey)

	// Test REST Endpoints
	testFuturesREST(futuresClient)
	testCopyTradeREST(copyTradeClient)
	testMFuturesREST(mFuturesClient)
	testSpotREST(spotClient)

	// Test WebSocket Functionalities with Connection Pooling
	testWebSocket(futuresClient, spotClient)
}

// Test Futures REST Endpoints
func testFuturesREST(client *bitrue.FuturesClient) {
	fmt.Println("Testing Futures REST Endpoints...")

	// Ping
	_, err := client.Ping()
	if err != nil {
		log.Printf("Futures Ping failed: %v", err)
	} else {
		fmt.Println("Futures Ping: Success")
	}

	// ServerTime
	_, err = client.ServerTime()
	if err != nil {
		log.Printf("Futures ServerTime failed: %v", err)
	} else {
		fmt.Println("Futures ServerTime: Success")
	}

	// Contracts
	_, err = client.Contracts()
	if err != nil {
		log.Printf("Futures Contracts failed: %v", err)
	} else {
		fmt.Println("Futures Contracts: Success")
	}

	// Ticker
	_, err = client.Ticker("BTCUSDT")
	if err != nil {
		log.Printf("Futures Ticker failed: %v", err)
	} else {
		fmt.Println("Futures Ticker: Success")
	}

	// Depth
	_, err = client.Depth("BTCUSDT", 10)
	if err != nil {
		log.Printf("Futures Depth failed: %v", err)
	} else {
		fmt.Println("Futures Depth: Success")
	}

	// Klines
	_, err = client.Klines("BTCUSDT", "1min", 10)
	if err != nil {
		log.Printf("Futures Klines failed: %v", err)
	} else {
		fmt.Println("Futures Klines: Success")
	}

	// NewOrder (example parameters)
	orderParams := map[string]interface{}{
		"contractName": "BTCUSDT",
		"side":         "BUY",
		"type":         "LIMIT",
		"open":         "open",
		"volume":       1.0,
		"price":        50000.0,
	}
	_, err = client.NewOrder(orderParams)
	if err != nil {
		log.Printf("Futures NewOrder failed: %v", err)
	} else {
		fmt.Println("Futures NewOrder: Success")
	}

	// QueryOrder
	_, err = client.QueryOrder("123456", "BTCUSDT")
	if err != nil {
		log.Printf("Futures QueryOrder failed: %v", err)
	} else {
		fmt.Println("Futures QueryOrder: Success")
	}

	// CancelOrder
	cancelParams := map[string]interface{}{
		"orderId":      "123456",
		"contractName": "BTCUSDT",
	}
	_, err = client.CancelOrder(cancelParams)
	if err != nil {
		log.Printf("Futures CancelOrder failed: %v", err)
	} else {
		fmt.Println("Futures CancelOrder: Success")
	}

	// OpenOrders
	_, err = client.OpenOrders("BTCUSDT")
	if err != nil {
		log.Printf("Futures OpenOrders failed: %v", err)
	} else {
		fmt.Println("Futures OpenOrders: Success")
	}

	// Account
	_, err = client.Account()
	if err != nil {
		log.Printf("Futures Account failed: %v", err)
	} else {
		fmt.Println("Futures Account: Success")
	}

	// MyTrades
	_, err = client.MyTrades("BTCUSDT", 10)
	if err != nil {
		log.Printf("Futures MyTrades failed: %v", err)
	} else {
		fmt.Println("Futures MyTrades: Success")
	}

	// LeverageBracket
	_, err = client.LeverageBracket("BTCUSDT")
	if err != nil {
		log.Printf("Futures LeverageBracket failed: %v", err)
	} else {
		fmt.Println("Futures LeverageBracket: Success")
	}

	// CommissionRate
	_, err = client.CommissionRate("BTCUSDT")
	if err != nil {
		log.Printf("Futures CommissionRate failed: %v", err)
	} else {
		fmt.Println("Futures CommissionRate: Success")
	}

	// FuturesTransfer
	transferParams := map[string]interface{}{
		"asset":        "USDT",
		"amount":       100.0,
		"transferType": "1",
	}
	_, err = client.FuturesTransfer(transferParams)
	if err != nil {
		log.Printf("Futures FuturesTransfer failed: %v", err)
	} else {
		fmt.Println("Futures FuturesTransfer: Success")
	}

	// FuturesTransferHistory
	_, err = client.FuturesTransferHistory(0, 0, 10)
	if err != nil {
		log.Printf("Futures FuturesTransferHistory failed: %v", err)
	} else {
		fmt.Println("Futures FuturesTransferHistory: Success")
	}

	// ForceOrdersHistory
	_, err = client.ForceOrdersHistory("BTCUSDT", 10)
	if err != nil {
		log.Printf("Futures ForceOrdersHistory failed: %v", err)
	} else {
		fmt.Println("Futures ForceOrdersHistory: Success")
	}

	// CreateListenKey
	_, err = client.CreateListenKey()
	if err != nil {
		log.Printf("Futures CreateListenKey failed: %v", err)
	} else {
		fmt.Println("Futures CreateListenKey: Success")
	}
}

// Test Copy Trading REST Endpoints
func testCopyTradeREST(client *bitrue.CopyTradeClient) {
	fmt.Println("Testing Copy Trading REST Endpoints...")

	// OpenOrders
	_, err := client.OpenOrders("BTCUSDT")
	if err != nil {
		log.Printf("CopyTrade OpenOrders failed: %v", err)
	} else {
		fmt.Println("CopyTrade OpenOrders: Success")
	}

	// NewOrder (example parameters)
	orderParams := map[string]interface{}{
		"contractName": "BTCUSDT",
		"side":         "BUY",
		"type":         "LIMIT",
		"open":         "open",
		"volume":       1.0,
		"price":        50000.0,
	}
	_, err = client.NewOrder(orderParams)
	if err != nil {
		log.Printf("CopyTrade NewOrder failed: %v", err)
	} else {
		fmt.Println("CopyTrade NewOrder: Success")
	}

	// Account
	_, err = client.Account()
	if err != nil {
		log.Printf("CopyTrade Account failed: %v", err)
	} else {
		fmt.Println("CopyTrade Account: Success")
	}

	// Positions
	_, err = client.Positions()
	if err != nil {
		log.Printf("CopyTrade Positions failed: %v", err)
	} else {
		fmt.Println("CopyTrade Positions: Success")
	}

	// ConfigQuerySymbols
	_, err = client.ConfigQuerySymbols()
	if err != nil {
		log.Printf("CopyTrade ConfigQuerySymbols failed: %v", err)
	} else {
		fmt.Println("CopyTrade ConfigQuerySymbols: Success")
	}

	// Status
	statusParams := map[string]interface{}{
		"contractName": "BTCUSDT",
	}
	_, err = client.Status(statusParams)
	if err != nil {
		log.Printf("CopyTrade Status failed: %v", err)
	} else {
		fmt.Println("CopyTrade Status: Success")
	}
}

// Test COIN-M Futures REST Endpoints
func testMFuturesREST(client *bitrue.MFuturesClient) {
	fmt.Println("Testing COIN-M Futures REST Endpoints...")

	// Ping
	_, err := client.Ping()
	if err != nil {
		log.Printf("MFutures Ping failed: %v", err)
	} else {
		fmt.Println("MFutures Ping: Success")
	}

	// ServerTime
	_, err = client.ServerTime()
	if err != nil {
		log.Printf("MFutures ServerTime failed: %v", err)
	} else {
		fmt.Println("MFutures ServerTime: Success")
	}

	// Contracts
	_, err = client.Contracts()
	if err != nil {
		log.Printf("MFutures Contracts failed: %v", err)
	} else {
		fmt.Println("MFutures Contracts: Success")
	}

	// Ticker
	_, err = client.Ticker("BTCUSD")
	if err != nil {
		log.Printf("MFutures Ticker failed: %v", err)
	} else {
		fmt.Println("MFutures Ticker: Success")
	}

	// Depth
	_, err = client.Depth("BTCUSD", 10)
	if err != nil {
		log.Printf("MFutures Depth failed: %v", err)
	} else {
		fmt.Println("MFutures Depth: Success")
	}

	// Klines
	_, err = client.Klines("BTCUSD", "1min", 10)
	if err != nil {
		log.Printf("MFutures Klines failed: %v", err)
	} else {
		fmt.Println("MFutures Klines: Success")
	}

	// NewOrder (example parameters)
	orderParams := map[string]interface{}{
		"contractName": "BTCUSD",
		"side":         "BUY",
		"type":         "LIMIT",
		"open":         "open",
		"volume":       1.0,
		"price":        50000.0,
	}
	_, err = client.NewOrder(orderParams)
	if err != nil {
		log.Printf("MFutures NewOrder failed: %v", err)
	} else {
		fmt.Println("MFutures NewOrder: Success")
	}

	// Account
	_, err = client.Account()
	if err != nil {
		log.Printf("MFutures Account failed: %v", err)
	} else {
		fmt.Println("MFutures Account: Success")
	}

	// FuturesTransfer
	transferParams := map[string]interface{}{
		"asset":        "BTC",
		"amount":       0.01,
		"transferType": "1",
	}
	_, err = client.FuturesTransfer(transferParams)
	if err != nil {
		log.Printf("MFutures FuturesTransfer failed: %v", err)
	} else {
		fmt.Println("MFutures FuturesTransfer: Success")
	}
}

// Test Spot Trading REST Endpoints
func testSpotREST(client *bitrue.SpotClient) {
	fmt.Println("Testing Spot Trading REST Endpoints...")

	// Ping
	_, err := client.Ping()
	if err != nil {
		log.Printf("Spot Ping failed: %v", err)
	} else {
		fmt.Println("Spot Ping: Success")
	}

	// ServerTime
	_, err = client.ServerTime()
	if err != nil {
		log.Printf("Spot ServerTime failed: %v", err)
	} else {
		fmt.Println("Spot ServerTime: Success")
	}

	// ExchangeInfo
	_, err = client.ExchangeInfo()
	if err != nil {
		log.Printf("Spot ExchangeInfo failed: %v", err)
	} else {
		fmt.Println("Spot ExchangeInfo: Success")
	}

	// Trades
	_, err = client.Trades("BTCUSDT", 10)
	if err != nil {
		log.Printf("Spot Trades failed: %v", err)
	} else {
		fmt.Println("Spot Trades: Success")
	}

	// HistoricalTrades
	_, err = client.HistoricalTrades("BTCUSDT", 10, 0)
	if err != nil {
		log.Printf("Spot HistoricalTrades failed: %v", err)
	} else {
		fmt.Println("Spot HistoricalTrades: Success")
	}

	// AggTrades
	_, err = client.AggTrades("BTCUSDT", 10)
	if err != nil {
		log.Printf("Spot AggTrades failed: %v", err)
	} else {
		fmt.Println("Spot AggTrades: Success")
	}

	// Klines
	_, err = client.Klines("BTCUSDT", "1min", 10)
	if err != nil {
		log.Printf("Spot Klines failed: %v", err)
	} else {
		fmt.Println("Spot Klines: Success")
	}

	// Ticker24hr
	_, err = client.Ticker24hr("BTCUSDT")
	if err != nil {
		log.Printf("Spot Ticker24hr failed: %v", err)
	} else {
		fmt.Println("Spot Ticker24hr: Success")
	}

	// TickerPrice
	_, err = client.TickerPrice("BTCUSDT")
	if err != nil {
		log.Printf("Spot TickerPrice failed: %v", err)
	} else {
		fmt.Println("Spot TickerPrice: Success")
	}

	// TickerBook
	_, err = client.TickerBook("BTCUSDT")
	if err != nil {
		log.Printf("Spot TickerBook failed: %v", err)
	} else {
		fmt.Println("Spot TickerBook: Success")
	}

	// Depth
	_, err = client.Depth("BTCUSDT", 10)
	if err != nil {
		log.Printf("Spot Depth failed: %v", err)
	} else {
		fmt.Println("Spot Depth: Success")
	}

	// NewOrder (example parameters)
	orderParams := map[string]interface{}{
		"symbol":      "BTCUSDT",
		"side":        "BUY",
		"type":        "LIMIT",
		"quantity":    0.001,
		"price":       50000.0,
		"timeInForce": "GTC",
	}
	_, err = client.NewOrder(orderParams)
	if err != nil {
		log.Printf("Spot NewOrder failed: %v", err)
	} else {
		fmt.Println("Spot NewOrder: Success")
	}

	// QueryOrder
	queryParams := map[string]interface{}{
		"symbol":  "BTCUSDT",
		"orderId": "123456",
	}
	_, err = client.QueryOrder(queryParams)
	if err != nil {
		log.Printf("Spot QueryOrder failed: %v", err)
	} else {
		fmt.Println("Spot QueryOrder: Success")
	}

	// CancelOrder
	cancelParams := map[string]interface{}{
		"symbol":  "BTCUSDT",
		"orderId": "123456",
	}
	_, err = client.CancelOrder(cancelParams)
	if err != nil {
		log.Printf("Spot CancelOrder failed: %v", err)
	} else {
		fmt.Println("Spot CancelOrder: Success")
	}

	// OpenOrders
	_, err = client.OpenOrders("BTCUSDT")
	if err != nil {
		log.Printf("Spot OpenOrders failed: %v", err)
	} else {
		fmt.Println("Spot OpenOrders: Success")
	}

	// Account
	_, err = client.Account()
	if err != nil {
		log.Printf("Spot Account failed: %v", err)
	} else {
		fmt.Println("Spot Account: Success")
	}

	// MyTrades
	_, err = client.MyTrades("BTCUSDT", 10)
	if err != nil {
		log.Printf("Spot MyTrades failed: %v", err)
	} else {
		fmt.Println("Spot MyTrades: Success")
	}

	// WithdrawCommit (example parameters)
	withdrawParams := map[string]interface{}{
		"coin":    "BTC",
		"address": "your-wallet-address",
		"amount":  0.01,
		"network": "BTC",
	}
	_, err = client.WithdrawCommit(withdrawParams)
	if err != nil {
		log.Printf("Spot WithdrawCommit failed: %v", err)
	} else {
		fmt.Println("Spot WithdrawCommit: Success")
	}

	// WithdrawHistory
	withdrawHistoryParams := map[string]interface{}{
		"coin": "BTC",
	}
	_, err = client.WithdrawHistory(withdrawHistoryParams)
	if err != nil {
		log.Printf("Spot WithdrawHistory failed: %v", err)
	} else {
		fmt.Println("Spot WithdrawHistory: Success")
	}

	// DepositHistory
	depositHistoryParams := map[string]interface{}{
		"coin": "BTC",
	}
	_, err = client.DepositHistory(depositHistoryParams)
	if err != nil {
		log.Printf("Spot DepositHistory failed: %v", err)
	} else {
		fmt.Println("Spot DepositHistory: Success")
	}

	// CreateListenKey
	_, err = client.CreateListenKey()
	if err != nil {
		log.Printf("Spot CreateListenKey failed: %v", err)
	} else {
		fmt.Println("Spot CreateListenKey: Success")
	}
}

// Test WebSocket Functionalities with Connection Pooling
func testWebSocket(futuresClient *bitrue.FuturesClient, spotClient *bitrue.SpotClient) {
	fmt.Println("Testing WebSocket Functionalities with Connection Pooling...")

	// Initialize WebSocket Pool Manager for Futures
	futuresPoolManager := wsmanager.NewWebSocketPoolManager(5, func() (wsmanager.WebSocketClientIface, error) {
		return wsmanager.NewWebSocketClient(futuresClient.WsMarketURL()), nil
	})

	// Subscribe to Depth and Kline for Futures with separate callbacks
	_, err := futuresPoolManager.Subscribe("market_BTCUSDT_depth_step0", "depth", "BTCUSDT", "", depthCallback)
	if err != nil {
		log.Printf("Futures Depth Subscription failed: %v", err)
	} else {
		fmt.Println("Futures Depth Subscription: Success")
	}

	_, err = futuresPoolManager.Subscribe("market_BTCUSDT_kline_1min", "kline", "BTCUSDT", "1min", klineCallback)
	if err != nil {
		log.Printf("Futures Kline Subscription failed: %v", err)
	} else {
		fmt.Println("Futures Kline Subscription: Success")
	}

	// Initialize WebSocket Pool Manager for Spot
	spotPoolManager := wsmanager.NewWebSocketPoolManager(5, func() (wsmanager.WebSocketClientIface, error) {
		return wsmanager.NewWebSocketClient(spotClient.WsMarketURL()), nil
	})

	// Subscribe to Depth for Spot with a separate callback
	_, err = spotPoolManager.Subscribe("market_BTCUSDT_depth_step0", "depth", "BTCUSDT", "", depthCallback)
	if err != nil {
		log.Printf("Spot Depth Subscription failed: %v", err)
	} else {
		fmt.Println("Spot Depth Subscription: Success")
	}

	// Start User WebSocket for Futures
	listenKeyResp, err := futuresClient.CreateListenKey()
	if err != nil {
		log.Printf("Futures CreateListenKey failed: %v", err)
	} else {
		var listenKey struct {
			ListenKey string `json:"listenKey"`
		}
		json.Unmarshal(listenKeyResp, &listenKey)
		err = futuresClient.StartUserWebSocket(listenKey.ListenKey, userDataCallback)
		if err != nil {
			log.Printf("Futures StartUserWebSocket failed: %v", err)
		} else {
			fmt.Println("Futures StartUserWebSocket: Success")
		}
	}

	// Start User WebSocket for Spot
	listenKeyResp, err = spotClient.CreateListenKey()
	if err != nil {
		log.Printf("Spot CreateListenKey failed: %v", err)
	} else {
		var listenKey struct {
			ListenKey string `json:"listenKey"`
		}
		json.Unmarshal(listenKeyResp, &listenKey)
		err = spotClient.StartUserWebSocket(listenKey.ListenKey, userDataCallback)
		if err != nil {
			log.Printf("Spot StartUserWebSocket failed: %v", err)
		} else {
			fmt.Println("Spot StartUserWebSocket: Success")
		}
	}

	// Keep the script running to receive WebSocket messages
	fmt.Println("WebSocket subscriptions active. Press Ctrl+C to exit.")
	select {}
}
