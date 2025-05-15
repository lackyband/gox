package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lackyband/gox/bitrueFutures"
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		fmt.Fprintf(os.Stderr, "Missing required environment variable: %s\n", key)
		os.Exit(1)
	}
	return val
}

func prettyPrint(label string, data []byte) {
	fmt.Printf("\n=== %s ===\n", label)
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err == nil {
		pretty, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Println(string(data))
	}
}

func main() {
	_ = godotenv.Load()

	apiKey := mustEnv("BITRUE_API_KEY")
	secretKey := mustEnv("BITRUE_SECRET_KEY")
	client := bitrueFutures.NewBitrueClient(apiKey, secretKey)

	fmt.Println("=== Testing REST Endpoints ===")

	if data, err := client.Ping(); err != nil {
		fmt.Printf("Ping failed: %v\n", err)
	} else {
		prettyPrint("Ping", data)
	}

	if data, err := client.ServerTime(); err != nil {
		fmt.Printf("ServerTime failed: %v\n", err)
	} else {
		prettyPrint("ServerTime", data)
	}

	contractsData, err := client.Contracts()
	if err != nil {
		fmt.Printf("Contracts failed: %v\n", err)
	} else {
		prettyPrint("Contracts", contractsData)
	}

	depthResp, err := client.Depth("e_btcusdt", 10)
	if err != nil {
		fmt.Printf("Depth failed: %v\n", err)
	} else {
		prettyPrint("Depth", depthResp)
	}

	accountResp, err := client.Account()
	if err != nil {
		fmt.Printf("Account failed: %v\n", err)
	} else {
		prettyPrint("Account", accountResp)
	}

	leverageBracketResp, err := client.LeverageBracket("e_btcusdt")
	if err != nil {
		fmt.Printf("LeverageBracket failed: %v\n", err)
	} else {
		prettyPrint("LeverageBracket", leverageBracketResp)
	}

	modMarginResp, err := client.ModifyPositionMargin(struct {
		ContractName   string
		PositionMargin float64
		Type           int
	}{ContractName: "e_btcusdt", PositionMargin: 10, Type: 1})
	if err != nil {
		fmt.Printf("ModifyPositionMargin failed (expected if no open position): %v\n", err)
	} else {
		prettyPrint("ModifyPositionMargin", modMarginResp)
	}

	fmt.Println("\n=== Testing WebSocket Endpoints ===")
	if err := client.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start market WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer client.StopWebSocketMarket()

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

	fmt.Println("Running for 30 seconds to collect market WebSocket messages...")
	time.Sleep(30 * time.Second)

	if err := client.UnsubscribeWebSocket("market_$symbol_depth_step0", "e_btcusdt", "", true); err != nil {
		fmt.Printf("Failed to unsubscribe from depth: %v\n", err)
	}
	if err := client.UnsubscribeWebSocket("market_$symbol_kline_$interval", "e_btcusdt", "1min", true); err != nil {
		fmt.Printf("Failed to unsubscribe from kline: %v\n", err)
	}

	fmt.Println("Tester completed.")
}
