package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lackyband/gox/bitrueMFutures"
)

func main() {
	// Load API credentials from environment variables
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		return
	}

	// Initialize client
	client := bitrueMFutures.NewBitrueMClient(apiKey, secretKey)

	// Test REST API endpoints
	fmt.Println("Testing REST API Endpoints...")

	// Ping
	if resp, err := client.Ping(); err != nil {
		fmt.Printf("Ping failed: %v\n", err)
	} else {
		fmt.Printf("Ping response: %s\n", string(resp))
	}

	// ServerTime
	if resp, err := client.ServerTime(); err != nil {
		fmt.Printf("ServerTime failed: %v\n", err)
	} else {
		fmt.Printf("ServerTime response: %s\n", string(resp))
	}

	// Contracts
	var contractSymbol string
	if resp, err := client.Contracts(); err != nil {
		fmt.Printf("Contracts failed: %v\n", err)
	} else {
		fmt.Printf("Contracts response: %s\n", string(resp))
		// DEBUG: Try to parse the first contract symbol
		var contractsData struct {
			Data []struct {
				ContractName string `json:"contractName"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &contractsData); err == nil && len(contractsData.Data) > 0 {
			contractSymbol = contractsData.Data[0].ContractName
			fmt.Printf("DEBUG: Using contract symbol: %s\n", contractSymbol)
		} else {
			contractSymbol = "H-HT-USD" // fallback
		}
	}

	// Depth
	if resp, err := client.Depth(contractSymbol, 10); err != nil {
		fmt.Printf("Depth failed: %v\n", err)
	} else {
		fmt.Printf("Depth response: %s\n", string(resp))
	}

	// Test WebSocket market data
	fmt.Println("\nTesting WebSocket Market Data...")

	// Test user stream
	fmt.Println("\nTesting User Stream...")
	// Create listenKey
	listenKeyResp, err := client.CreateListenKey()
	if err != nil {
		fmt.Printf("CreateListenKey failed: %v\n", err)
		return
	}
	fmt.Printf("Raw listenKey response: %s\n", string(listenKeyResp))
	var listenKeyData struct {
		Data struct {
			ListenKey string `json:"listenKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listenKeyResp, &listenKeyData); err != nil {
		fmt.Printf("Failed to parse listenKey: %v\n", err)
		return
	}
	listenKey := listenKeyData.Data.ListenKey
	fmt.Printf("Parsed listenKey: %s\n", listenKey)
	fmt.Printf("Created listenKey: %s\n", listenKey)

	// Immediately try to keep alive the listenKey and print the response
	if resp, err := client.KeepAliveListenKey(listenKey); err != nil {
		fmt.Printf("KeepAliveListenKey (immediate) failed: %v\n", err)
	} else {
		fmt.Printf("KeepAliveListenKey (immediate) response: %s\n", string(resp))
	}

	// Start user WebSocket
	if err := client.StartWebSocketUser(listenKey); err != nil {
		fmt.Printf("Failed to start WebSocket user: %v\n", err)
		return
	}
	fmt.Printf("DEBUG: Called StartWebSocketUser for listenKey: %s\n", listenKey)

	defer client.StopWebSocketUser()

	userDataCallback := func(message []byte) {
		fmt.Printf("User data message: %s\n", string(message))
	}
	if err := client.SubscribeUserData(listenKey, userDataCallback); err != nil {
		fmt.Printf("Failed to subscribe to user data: %v\n", err)
	}

	go func() {
		ticker := time.NewTicker(25 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := client.KeepAliveListenKey(listenKey); err != nil {
				fmt.Printf("Failed to keep alive listenKey: %v\n", err)
			} else {
				fmt.Println("ListenKey kept alive")
			}
		}
	}()

	// Wait for user data messages
	fmt.Println("Running for 30 seconds to observe WebSocket messages...")
	time.Sleep(30 * time.Second)
}
