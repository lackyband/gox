# BitrueCopyTrade Module

The `bitrueCopyTrade` module is a Go library for interacting with the Bitrue Copy Trading API, part of the USDT-M Futures platform. It supports REST API endpoints for managing copy trading orders, account information, positions, and configuration, as well as WebSocket streams for real-time user data updates. This module is designed for developers building applications to automate or monitor copy trading activities on Bitrue.

## Installation

To use the `bitrueCopyTrade` module, ensure you have Go installed. Add the module to your project and install the required dependency:

```bash
go get github.com/gorilla/websocket
```

Clone or include the `bitrueCopyTrade` package in your Go module:

```bash
# If the module is hosted on GitHub, replace <repository> with the actual path
go get github.com/<repository>/bitrueCopyTrade
```

Alternatively, copy the `bitrueCopyTrade` package into your project directory.

## Setup

### Prerequisites
- **API Credentials**: Obtain an API key and secret key from your Bitrue account. Ensure the API key has permissions for Copy Trading (USDT-M Futures) and user data access.
- **Go Environment**: Go 1.16 or later is recommended.

### Project Structure
Place the `bitrueCopyTrade` package in your project directory or import it as a module. A typical setup might look like:

```
my_project/
├── bitrueCopyTrade/
│   └── bitrueCopyTrade.go
├── main.go
└── go.mod
```

Initialize your Go module:

```bash
go mod init my_project
```

### Dependencies
The module requires the `gorilla/websocket` package for WebSocket functionality:

```bash
go get github.com/gorilla/websocket
```

## Usage

### Initializing the Client
Create a `BitrueClient` instance with your API credentials:

```go
package main

import (
	"fmt"
	"os"
	"my_project/bitrueCopyTrade"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		return
	}

	client := bitrueCopyTrade.NewBitrueClient(apiKey, secretKey)
}
```

Set environment variables for security:

```bash
export BITRUE_API_KEY="your_api_key"
export BITRUE_SECRET_KEY="your_secret_key"
```

### REST API Examples

#### Retrieve Open Orders
Get all open copy trading orders for a contract:

```go
resp, err := client.OpenOrders("E-SAND-USDT")
if err != nil {
	fmt.Printf("OpenOrders failed: %v\n", err)
} else {
	fmt.Printf("OpenOrders response: %s\n", string(resp))
}
```

#### Place a New Copy Trading Order
Place a limit buy order for the `E-SAND-USDT` contract:

```go
orderParams := struct {
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
	TriggerPrice:   100.0, // Required for trigger orders
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
resp, err := client.NewOrder(orderParams)
if err != nil {
	fmt.Printf("NewOrder failed: %v\n", err)
} else {
	fmt.Printf("NewOrder response: %s\n", string(resp))
}
```

#### Cancel a Copy Trading Order
Cancel an order by order ID:

```go
cancelParams := struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}{
	ContractName:   "E-SAND-USDT",
	OrderID:        1690615847831143159, // Replace with a valid order ID
	ConditionOrder: true,
}
resp, err := client.CancelOrder(cancelParams)
if err != nil {
	fmt.Printf("CancelOrder failed: %v\n", err)
} else {
	fmt.Printf("CancelOrder response: %s\n", string(resp))
}
```

#### Query a Copy Trading Order
Query an order by order ID:

```go
queryParams := struct {
	ContractName   string
	ClientOrderID  string
	OrderID        int64
	ConditionOrder bool
}{
	ContractName:   "E-SAND-USDT",
	OrderID:        1690615710392189353, // Replace with a valid order ID
	ConditionOrder: true,
}
resp, err := client.QueryOrder(queryParams)
if err != nil {
	fmt.Printf("QueryOrder failed: %v\n", err)
} else {
	fmt.Printf("QueryOrder response: %s\n", string(resp))
}
```

#### Retrieve Account Information
Get copy trading account details:

```go
resp, err := client.Account()
if err != nil {
	fmt.Printf("Account failed: %v\n", err)
} else {
	fmt.Printf("Account response: %s\n", string(resp))
}
```

#### Retrieve Copy Trading Positions
Get current copy trading positions:

```go
resp, err := client.Positions()
if err != nil {
	fmt.Printf("Positions failed: %v\n", err)
} else {
	fmt.Printf("Positions response: %s\n", string(resp))
}
```

#### Retrieve Symbol Configuration
Get configuration for copy trading symbols:

```go
resp, err := client.ConfigQuerySymbols()
if err != nil {
	fmt.Printf("ConfigQuerySymbols failed: %v\n", err)
} else {
	fmt.Printf("ConfigQuerySymbols response: %s\n", string(resp))
}
```

#### Retrieve Copy Trading Status
Check the status of copy trading:

```go
resp, err := client.Status()
if err != nil {
	fmt.Printf("Status failed: %v\n", err)
} else {
	fmt.Printf("Status response: %s\n", string(resp))
}
```

### WebSocket Examples

#### Subscribe to User Data Stream
Subscribe to real-time updates for copy trading orders and positions:

```go
listenKeyResp, err := client.CreateListenKey()
if err != nil {
	fmt.Printf("Failed to create listenKey: %v\n", err)
	return
}
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
fmt.Printf("Created listenKey: %s\n", listenKey)

if err := client.StartWebSocketUser(listenKey); err != nil {
	fmt.Printf("Failed to start WebSocket user: %v\n", err)
	return
}
defer client.StopWebSocketUser()

userDataCallback := func(message []byte) {
	fmt.Printf("User data message: %s\n", string(message))
}
if err := client.SubscribeUserData(listenKey, userDataCallback); err != nil {
	fmt.Printf("Failed to subscribe to user data: %v\n", err)
}
```

#### Keep ListenKey Alive
Periodically ping the listenKey to prevent expiration:

```go
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
```

## Important Notes

- **Signature Requirements**: All endpoints require HMAC SHA256 signatures passed via `X-CH-SIGN` and `X-CH-TS` headers. Ensure your system clock is synchronized to avoid timestamp errors.
- **Rate Limits**: The API may return HTTP 429 if rate limits are exceeded. Implement exponential backoff or rate limiting in your application to avoid bans.
- **WebSocket Limits**: Do not maintain more than 100 WebSocket connections per IP to avoid disconnection or IP bans.
- **Error Handling**: Check response codes and error messages (e.g., `{"code": -1121, "msg": "Invalid symbol."}`) for debugging. The module retries failed requests up to three times.
- **Contract Naming**: Use contract names as specified in the API (e.g., `E-SAND-USDT`, `E-BTC-USDT`), ensuring correct casing.
- **User Data Streams**: The WebSocket user data stream provides real-time updates for orders and positions. A valid `listenKey` is required, which expires after 60 minutes unless kept alive.

## Example Project

A complete example tester can be found below. Save it as `main.go` and run it after setting up your environment:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"my_project/bitrueCopyTrade"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY")
		return
	}

	client := bitrueCopyTrade.NewBitrueClient(apiKey, secretKey)

	// Test REST API
	resp, err := client.Account()
	if err != nil {
		fmt.Printf("Account failed: %v\n", err)
	} else {
		fmt.Printf("Account response: %s\n", string(resp))
	}

	// Test WebSocket
	listenKeyResp, err := client.CreateListenKey()
	if err != nil {
		fmt.Printf("Failed to create listenKey: %v\n", err)
		return
	}
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

	if err := client.StartWebSocketUser(listenKey); err != nil {
		fmt.Printf("Failed to start WebSocket: %v\n", err)
		return
	}
	defer client.StopWebSocketUser()

	userDataCallback := func(message []byte) {
		fmt.Printf("User data message: %s\n", string(message))
	}
	if err := client.SubscribeUserData(listenKey, userDataCallback); err != nil {
		fmt.Printf("Failed to subscribe: %v\n", err)
	}

	// Run for 30 seconds
	time.Sleep(30 * time.Second)
}
```

Run the example:

```bash
go run main.go
```

## Contributing

Contributions are welcome! Please submit pull requests or open issues on the GitHub repository for bug fixes, enhancements, or additional endpoint support.

## Disclaimer

This module is not officially supported by Bitrue. Use it at your own risk, and verify all API interactions for security and accuracy. Bitrue is not liable for any losses resulting from the use of this module.