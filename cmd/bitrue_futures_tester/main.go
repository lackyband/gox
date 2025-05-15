package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lackyband/gox/bitrueFutures"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		os.Exit(1)
	}

	futuresClient := bitrueFutures.NewBitrueClient(apiKey, secretKey)

	fmt.Println("=== Testing Futures Endpoints ===")

	pingResp, err := futuresClient.Ping()
	if err != nil {
		fmt.Printf("Futures Ping failed: %v\n", err)
	} else {
		fmt.Printf("Futures Ping response: %s\n", pingResp)
	}

	timeResp, err := futuresClient.ServerTime()
	if err != nil {
		fmt.Printf("Futures ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("Futures ServerTime response: %s\n", timeResp)
	}

	depthResp, err := futuresClient.Depth("E-BTC-USDT", 10)
	if err != nil {
		fmt.Printf("Futures Depth failed: %v\n", err)
	} else {
		fmt.Printf("Futures Depth response: %s\n", depthResp)
	}

	accountResp, err := futuresClient.Account()
	if err != nil {
		fmt.Printf("Futures Account failed: %v\n", err)
	} else {
		fmt.Printf("Futures Account response: %s\n", accountResp)
	}

	// WebSocket Endpoints (optional, can comment out if not needed)
	fmt.Println("\n=== Testing Futures WebSocket Endpoints ===")
	if err := futuresClient.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start futures market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer futuresClient.StopWebSocketMarket()
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

	fmt.Println("Running for 30 seconds to collect WebSocket messages...")
	time.Sleep(30 * time.Second)

	if err := futuresClient.UnsubscribeWebSocket("market_$symbol_depth_step0", "e_btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from futures depth: %v\n", err)
	}
	fmt.Println("Futures tester completed.")
}
