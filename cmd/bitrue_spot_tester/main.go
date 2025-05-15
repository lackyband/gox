package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lackyband/gox/bitrueSpot"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		os.Exit(1)
	}

	spotClient := bitrueSpot.NewBitrueClient(apiKey, secretKey)

	fmt.Println("=== Testing Spot Endpoints ===")

	pingResp, err := spotClient.Ping()
	if err != nil {
		fmt.Printf("Spot Ping failed: %v\n", err)
	} else {
		fmt.Printf("Spot Ping response: %s\n", pingResp)
	}

	timeResp, err := spotClient.ServerTime()
	if err != nil {
		fmt.Printf("Spot ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("Spot ServerTime response: %s\n", timeResp)
	}

	exchangeInfoResp, err := spotClient.ExchangeInfo()
	if err != nil {
		fmt.Printf("Spot ExchangeInfo failed: %v\n", err)
	} else {
		fmt.Printf("Spot ExchangeInfo response: %s\n", exchangeInfoResp)
	}

	depthResp, err := spotClient.Depth("BTCUSDT", 10)
	if err != nil {
		fmt.Printf("Spot Depth failed: %v\n", err)
	} else {
		fmt.Printf("Spot Depth response: %s\n", depthResp)
	}

	klinesResp, err := spotClient.Klines("BTCUSDT", "15m", 0, 10)
	if err != nil {
		fmt.Printf("Spot Klines failed: %v\n", err)
	} else {
		fmt.Printf("Spot Klines response: %s\n", klinesResp)
	}

	accountResp, err := spotClient.Account(5000)
	if err != nil {
		fmt.Printf("Spot Account failed: %v\n", err)
	} else {
		fmt.Printf("Spot Account response: %s\n", accountResp)
	}

	// WebSocket Endpoints (optional, can comment out if not needed)
	fmt.Println("\n=== Testing Spot WebSocket Endpoints ===")
	if err := spotClient.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start spot market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer spotClient.StopWebSocketMarket()
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

	fmt.Println("Running for 30 seconds to collect WebSocket messages...")
	time.Sleep(30 * time.Second)

	if err := spotClient.UnsubscribeWebSocket("market_$symbol_simple_depth_step0", "btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from spot depth: %v\n", err)
	}
	if err := spotClient.UnsubscribeWebSocket("user_order_update", "", "", false); err != nil {
		fmt.Printf("Failed to unsubscribe from spot order updates: %v\n", err)
	}
	if err := spotClient.UnsubscribeWebSocket("user_balance_update", "", "", false); err != nil {
		fmt.Printf("Failed to unsubscribe from spot balance updates: %v\n", err)
	}
	fmt.Println("Spot tester completed.")
}
