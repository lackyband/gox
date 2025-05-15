package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lackyband/gox/bitrueFutures"
)

func main() {
	// Load API key and secret from environment variables
	apiKey := "5f1310fd1cbd2cf9736e1feef5f014f4569b37ff4110c26bf5c39026ddca71ad"
	secretKey := "074fade678ae56177ceb40504c3faa9abd8d07250dda5c4e115f9b7f1389df56"
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		os.Exit(1)
	}

	client := bitrueFutures.NewBitrueClient(apiKey, secretKey)

	// Test REST Endpoints
	fmt.Println("=== Testing REST Endpoints ===")

	// Test Ping
	pingResp, err := client.Ping()
	if err != nil {
		fmt.Printf("Ping failed: %v\n", err)
	} else {
		fmt.Printf("Ping response: %s\n", pingResp)
	}

	// Test ServerTime
	timeResp, err := client.ServerTime()
	if err != nil {
		fmt.Printf("ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("ServerTime response: %s\n", timeResp)
	}

	// Test Contracts
	contractsResp, err := client.Contracts()
	if err != nil {
		fmt.Printf("Contracts failed: %v\n", err)
	} else {
		fmt.Printf("Contracts response: %s\n", contractsResp)
	}

	// Test Depth
	depthResp, err := client.Depth("E-BTC-USDT", 10)
	if err != nil {
		fmt.Printf("Depth failed: %v\n", err)
	} else {
		fmt.Printf("Depth response: %s\n", depthResp)
	}

	// Test Account (authenticated)
	accountResp, err := client.Account()
	if err != nil {
		fmt.Printf("Account failed: %v\n", err)
	} else {
		fmt.Printf("Account response: %s\n", accountResp)
	}

	// Test WebSocket Endpoints
	fmt.Println("\n=== Testing WebSocket Endpoints ===")

	// Start Market WebSocket
	if err := client.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer client.StopWebSocketMarket()

	// Subscribe to depth updates
	if err := client.SubscribeWebSocket("market_$symbol_depth_step0", "e_btcusdt", "", true); err != nil {
		fmt.Printf("Failed to subscribe to depth: %v\n", err)
		os.Exit(1)
	}

	// Subscribe to 1-minute Kline data
	if err := client.SubscribeWebSocket("market_$symbol_kline_$interval", "e_btcusdt", "1min", true); err != nil {
		fmt.Printf("Failed to subscribe to kline: %v\n", err)
		os.Exit(1)
	}

	// Request historical trade data
	if err := client.RequestWebSocketData("market_$symbol_trade_ticker", "e_btcusdt", "", 10, "", true); err != nil {
		fmt.Printf("Failed to request trade history: %v\n", err)
		os.Exit(1)
	}

	// Handle market WebSocket messages
	go func() {
		for msg := range client.GetWebSocketMessages(true) {
			var wsMsg bitrueFutures.WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				fmt.Printf("Failed to parse market WebSocket message: %v\n", err)
				continue
			}
			fmt.Printf("Market WebSocket message: Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
		}
	}()

	// Test User WebSocket
	listenKeyResp, err := client.CreateListenKey()
	if err != nil {
		fmt.Printf("Failed to create listenKey: %v\n", err)
		os.Exit(1)
	}
	var listenKeyData struct {
		Data struct {
			ListenKey string `json:"listenKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listenKeyResp, &listenKeyData); err != nil {
		fmt.Printf("Failed to parse listenKey: %v\n", err)
		os.Exit(1)
	}
	listenKey := listenKeyData.Data.ListenKey

	if err := client.StartWebSocketUser(listenKey); err != nil {
		fmt.Printf("Failed to start user WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer client.StopWebSocketUser()

	if err := client.SubscribeWebSocket("user_account_update", "", "", false); err != nil {
		fmt.Printf("Failed to subscribe to user account updates: %v\n", err)
		os.Exit(1)
	}

	// Handle user WebSocket messages
	go func() {
		for msg := range client.GetWebSocketMessages(false) {
			var wsMsg bitrueFutures.WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				fmt.Printf("Failed to parse user WebSocket message: %v\n", err)
				continue
			}
			fmt.Printf("User WebSocket message: Event=%s, Data=%v\n", wsMsg.Event, wsMsg.Data)
		}
	}()

	// Keep the program running for 30 seconds to collect messages
	fmt.Println("Running for 30 seconds to collect WebSocket messages...")
	time.Sleep(30 * time.Second)

	// Unsubscribe from market WebSocket channels
	if err := client.UnsubscribeWebSocket("market_$symbol_depth_step0", "e_btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from depth: %v\n", err)
	}
	if err := client.UnsubscribeWebSocket("market_$symbol_kline_$interval", "e_btcusdt", "1min", true); err != nil {
		fmt.Printf("Failed to unsubscribe from kline: %v\n", err)
	}

	fmt.Println("Tester completed.")
}
