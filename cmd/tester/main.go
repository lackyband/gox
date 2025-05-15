package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lackyband/gox/bitrueFutures"
	"github.com/lackyband/gox/bitrueSpot"
)

func main() {
	// Load API key and secret from environment variables
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		os.Exit(1)
	}

	// Initialize clients
	futuresClient := bitrueFutures.NewBitrueClient(apiKey, secretKey)
	spotClient := bitrueSpot.NewBitrueClient(apiKey, secretKey)

	// Test Futures Endpoints
	fmt.Println("=== Testing Futures Endpoints ===")

	// Futures: Ping
	pingResp, err := futuresClient.Ping()
	if err != nil {
		fmt.Printf("Futures Ping failed: %v\n", err)
	} else {
		fmt.Printf("Futures Ping response: %s\n", pingResp)
	}

	// Futures: ServerTime
	timeResp, err := futuresClient.ServerTime()
	if err != nil {
		fmt.Printf("Futures ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("Futures ServerTime response: %s\n", timeResp)
	}

	// Futures: Depth
	depthResp, err := futuresClient.Depth("E-BTC-USDT", 10)
	if err != nil {
		fmt.Printf("Futures Depth failed: %v\n", err)
	} else {
		fmt.Printf("Futures Depth response: %s\n", depthResp)
	}

	// Futures: Account
	accountResp, err := futuresClient.Account()
	if err != nil {
		fmt.Printf("Futures Account failed: %v\n", err)
	} else {
		fmt.Printf("Futures Account response: %s\n", accountResp)
	}

	// Test Spot Endpoints
	fmt.Println("\n=== Testing Spot Endpoints ===")

	// Spot: Ping
	pingResp, err = spotClient.Ping()
	if err != nil {
		fmt.Printf("Spot Ping failed: %v\n", err)
	} else {
		fmt.Printf("Spot Ping response: %s\n", pingResp)
	}

	// Spot: ServerTime
	timeResp, err = spotClient.ServerTime()
	if err != nil {
		fmt.Printf("Spot ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("Spot ServerTime response: %s\n", timeResp)
	}

	// Spot: ExchangeInfo
	exchangeInfoResp, err := spotClient.ExchangeInfo()
	if err != nil {
		fmt.Printf("Spot ExchangeInfo failed: %v\n", err)
	} else {
		fmt.Printf("Spot ExchangeInfo response: %s\n", exchangeInfoResp)
	}

	// Spot: Depth
	depthResp, err = spotClient.Depth("BTCUSDT", 10)
	if err != nil {
		fmt.Printf("Spot Depth failed: %v\n", err)
	} else {
		fmt.Printf("Spot Depth response: %s\n", depthResp)
	}

	// Spot: Klines
	klinesResp, err := spotClient.Klines("BTCUSDT", "15m", 0, 10)
	if err != nil {
		fmt.Printf("Spot Klines failed: %v\n", err)
	} else {
		fmt.Printf("Spot Klines response: %s\n", klinesResp)
	}

	// Spot: Account
	accountResp, err = spotClient.Account(5000)
	if err != nil {
		fmt.Printf("Spot Account failed: %v\n", err)
	} else {
		fmt.Printf("Spot Account response: %s\n", accountResp)
	}

	// Test WebSocket Endpoints
	fmt.Println("\n=== Testing WebSocket Endpoints ===")

	// Futures: Market WebSocket
	if err := futuresClient.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start futures market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer futuresClient.StopWebSocketMarket()

	// Futures: Subscribe to depth
	if err := futuresClient.SubscribeDepth("e_btcusdt", func(msg []byte) {
		var wsMsg bitrueFutures.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("Failed to parse futures depth message: %v\n", err)
			return
		}
		fmt.Printf("Futures Depth message: Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
	}); err != nil {
		fmt.Printf("Failed to subscribe to futures depth: %v\n", err)
		os.Exit(1)
	}

	// Spot: Market WebSocket
	if err := spotClient.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start spot market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer spotClient.StopWebSocketMarket()

	// Spot: Subscribe to depth
	if err := spotClient.SubscribeDepth("btcusdt", func(msg []byte) {
		var wsMsg bitrueSpot.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("Failed to parse spot depth message: %v\n", err)
			return
		}
		fmt.Printf("Spot Depth message: Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
	}); err != nil {
		fmt.Printf("Failed to subscribe to spot depth: %v\n", err)
		os.Exit(1)
	}

	// Futures: User WebSocket
	futuresListenKeyResp, err := futuresClient.CreateListenKey()
	if err != nil {
		fmt.Printf("Failed to create futures listenKey: %v\n", err)
		os.Exit(1)
	}
	var futuresListenKeyData struct {
		Data struct {
			ListenKey string `json:"listenKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(futuresListenKeyResp, &futuresListenKeyData); err != nil {
		fmt.Printf("Failed to parse futures listenKey: %v\n", err)
		os.Exit(1)
	}
	futuresListenKey := futuresListenKeyData.Data.ListenKey

	if err := futuresClient.StartWebSocketUser(futuresListenKey); err != nil {
		fmt.Printf("Failed to start futures user WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer futuresClient.StopWebSocketUser()

	if err := futuresClient.SubscribeUserData(futuresListenKey, func(msg []byte) {
		var wsMsg bitrueFutures.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("Failed to parse futures user message: %v\n", err)
			return
		}
		fmt.Printf("Futures User message: Event=%s, Data=%v\n", wsMsg.Event, wsMsg.Data)
	}); err != nil {
		fmt.Printf("Failed to subscribe to futures user data: %v\n", err)
		os.Exit(1)
	}

	// Spot: User WebSocket
	spotListenKeyResp, err := spotClient.CreateListenKey()
	if err != nil {
		fmt.Printf("Failed to create spot listenKey: %v\n", err)
		os.Exit(1)
	}
	var spotListenKeyData struct {
		Data struct {
			ListenKey string `json:"listenKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(spotListenKeyResp, &spotListenKeyData); err != nil {
		fmt.Printf("Failed to parse spot listenKey: %v\n", err)
		os.Exit(1)
	}
	spotListenKey := spotListenKeyData.Data.ListenKey

	if err := spotClient.StartWebSocketUser(spotListenKey); err != nil {
		fmt.Printf("Failed to start spot user WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer spotClient.StopWebSocketUser()

	if err := spotClient.SubscribeUserOrderUpdate(func(msg []byte) {
		var wsMsg bitrueSpot.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("Failed to parse spot order update message: %v\n", err)
			return
		}
		fmt.Printf("Spot Order Update message: Event=%s, Data=%v\n", wsMsg.Event, wsMsg.Data)
	}); err != nil {
		fmt.Printf("Failed to subscribe to spot order updates: %v\n", err)
		os.Exit(1)
	}

	if err := spotClient.SubscribeUserBalanceUpdate(func(msg []byte) {
		var wsMsg bitrueSpot.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("Failed to parse spot balance update message: %v\n", err)
			return
		}
		fmt.Printf("Spot Balance Update message: Event=%s, Data=%v\n", wsMsg.Event, wsMsg.Data)
	}); err != nil {
		fmt.Printf("Failed to subscribe to spot balance updates: %v\n", err)
		os.Exit(1)
	}

	// Keep the program running for 30 seconds to collect messages
	fmt.Println("Running for 30 seconds to collect WebSocket messages...")
	time.Sleep(30 * time.Second)

	// Unsubscribe from WebSocket channels
	if err := futuresClient.UnsubscribeWebSocket("market_$symbol_depth_step0", "e_btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from futures depth: %v\n", err)
	}
	if err := spotClient.UnsubscribeWebSocket("market_$symbol_simple_depth_step0", "btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from spot depth: %v\n", err)
	}
	if err := spotClient.UnsubscribeWebSocket("user_order_update", "", "", false); err != nil {
		fmt.Printf("Failed to unsubscribe from spot order updates: %v\n", err)
	}
	if err := spotClient.UnsubscribeWebSocket("user_balance_update", "", "", false); err != nil {
		fmt.Printf("Failed to unsubscribe from spot balance updates: %v\n", err)
	}

	fmt.Println("Tester completed.")
}
