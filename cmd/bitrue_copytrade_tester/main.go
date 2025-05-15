package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lackyband/gox/bitrueCopyTrade"
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
	client := bitrueCopyTrade.NewBitrueClient(apiKey, secretKey)

	fmt.Println("=== Testing REST Endpoints ===")

	// Test OpenOrders
	if data, err := client.OpenOrders("E-SAND-USDT"); err != nil {
		fmt.Printf("OpenOrders failed: %v\n", err)
	} else {
		prettyPrint("OpenOrders", data)
	}

	// Test Account
	if data, err := client.Account(); err != nil {
		fmt.Printf("Account failed: %v\n", err)
	} else {
		prettyPrint("Account", data)
	}

	// Test Positions
	if data, err := client.Positions(); err != nil {
		fmt.Printf("Positions failed: %v\n", err)
	} else {
		prettyPrint("Positions", data)
	}

	// Test ConfigQuerySymbols
	if data, err := client.ConfigQuerySymbols(); err != nil {
		fmt.Printf("ConfigQuerySymbols failed: %v\n", err)
	} else {
		prettyPrint("ConfigQuerySymbols", data)
	}

	// Test Status
	if data, err := client.Status(); err != nil {
		fmt.Printf("Status failed: %v\n", err)
	} else {
		prettyPrint("Status", data)
	}

	// Test QueryOrder (may fail without valid order ID)
	queryParams := struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}{
	ContractName:   "E-SAND-USDT",
	OrderID:        1690615710392189353, // Replace with valid order ID
	ConditionOrder: true,
}
	if data, err := client.QueryOrder(queryParams); err != nil {
		fmt.Printf("QueryOrder failed (expected if invalid order ID): %v\n", err)
	} else {
		prettyPrint("QueryOrder", data)
	}

	// Test NewOrder (may fail without sufficient balance or valid parameters)
	newOrderParams := struct {
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
}{
	ContractName:   "E-SAND-USDT",
	ClientOrderID:  "test123", // Replace with valid client order ID
	Side:           "BUY",
	Type:           "LIMIT",
	PositionType:   1,
	Open:           "OPEN",
	Volume:         100.0,
	Amount:         1.0,
	Price:          2.0,
	Leverage:       5,
	ConditionOrder: true,
	TriggerOrderParams: []map[string]interface{}{
		{
			"clientOrderId":    "1",
			"triggerType":      0,
			"triggerPriceType": 0,
			"triggerPrice":     100.0,
			"type":             "LIMIT",
			"price":            100.0,
		},
	},
}
	if data, err := client.NewOrder(newOrderParams); err != nil {
		fmt.Printf("NewOrder failed (expected if insufficient balance or invalid parameters): %v\n", err)
	} else {
		prettyPrint("NewOrder", data)
	}

	// Test CancelOrder (may fail without valid order ID)
	cancelParams := struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}{
	ContractName:   "E-SAND-USDT",
	OrderID:        1690615847831143159, // Replace with valid order ID
	ConditionOrder: true,
}
	if data, err := client.CancelOrder(cancelParams); err != nil {
		fmt.Printf("CancelOrder failed (expected if invalid order ID): %v\n", err)
	} else {
		prettyPrint("CancelOrder", data)
	}

	fmt.Println("\n=== Testing WebSocket Endpoints ===")

	// Test WebSocket User Data Stream
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
	fmt.Printf("Created listenKey: %s\n", listenKey)

	if err := client.StartWebSocketUser(listenKey); err != nil {
		fmt.Printf("Failed to start user WebSocket: %v\n", err)
		os.Exit(1)
	}
	defer client.StopWebSocketUser()

	if err := client.SubscribeUserData(listenKey, func(msg []byte) {
		var wsMsg bitrueCopyTrade.WebSocketMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			fmt.Printf("[UserData] Failed to parse message: %v\n", err)
			return
		}
		fmt.Printf("[UserData] Channel=%s, Status=%s, Data=%v\n", wsMsg.Channel, wsMsg.Status, wsMsg.Data)
	}); err != nil {
		fmt.Printf("Failed to subscribe to user data: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running for 30 seconds to collect user WebSocket messages...")
	time.Sleep(30 * time.Second)

	fmt.Println("Tester completed.")
}
