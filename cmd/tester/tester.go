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
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
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

	// Test LeverageBracket (new endpoint)
	leverageBracketResp, err := client.LeverageBracket("E-BTC-USDT")
	if err != nil {
		fmt.Printf("LeverageBracket failed: %v\n", err)
	} else {
		fmt.Printf("LeverageBracket response: %s\n", leverageBracketResp)
	}

	// Test ModifyPositionMargin (new endpoint)
	// WARNING: This may fail if you do not have an open position on the contract.
	modMarginResp, err := client.ModifyPositionMargin("E-BTC-USDT", 10)
	if err != nil {
		fmt.Printf("ModifyPositionMargin failed (expected if no open position): %v\n", err)
	} else {
		fmt.Printf("ModifyPositionMargin response: %s\n", modMarginResp)
	}


	// Test WebSocket Endpoints
	fmt.Println("\n=== Testing WebSocket Endpoints ===")

	// Start Market WebSocket
	if err := client.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer client.StopWebSocketMarket()

	// Subscribe to depth updates using new typed function
	if err := client.SubscribeDepth("e_btcusdt", func(msg []byte) {
		var wsMsg bitrueFutures.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("[Depth] Failed to parse message: %v\n", err)
			return
		}
		fmt.Printf("[Depth] Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
	}); err != nil {
		fmt.Printf("Failed to subscribe to depth: %v\n", err)
		os.Exit(1)
	}

	// Subscribe to 1-minute Kline data using new typed function
	if err := client.SubscribeKline("e_btcusdt", "1min", func(msg []byte) {
		var wsMsg bitrueFutures.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("[Kline] Failed to parse message: %v\n", err)
			return
		}
		fmt.Printf("[Kline] Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
	}); err != nil {
		fmt.Printf("Failed to subscribe to kline: %v\n", err)
		os.Exit(1)
	}

	// Subscribe to trade ticker using new typed function
	if err := client.SubscribeTrade("e_btcusdt", func(msg []byte) {
		var wsMsg bitrueFutures.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("[Trade] Failed to parse message: %v\n", err)
			return
		}
		fmt.Printf("[Trade] Channel=%s, Status=%s, Tick=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Tick)
	}); err != nil {
		fmt.Printf("Failed to subscribe to trade: %v\n", err)
		os.Exit(1)
	}

	// Keep the program running for 30 seconds to collect market WebSocket messages
	fmt.Println("Running for 30 seconds to collect market WebSocket messages...")
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
