# BitrueSpot Module

The `bitrueSpot` module is a Go library for interacting with the Bitrue Spot API. It supports REST API endpoints for market data, trading, account management, and withdrawals/deposits, as well as WebSocket streams for real-time market and user data. This module is designed for developers building trading bots or applications that require programmatic access to Bitrue's Spot trading platform.

## Installation

To use the `bitrueSpot` module, ensure you have Go installed. Add the module to your project and install the required dependency:

```bash
go get github.com/gorilla/websocket
```

Clone or include the `bitrueSpot` package in your Go module:

```bash
# If the module is hosted on GitHub, replace <repository> with the actual path
go get github.com/lackyband/gox/bitrueSpot
```

Alternatively, copy the `bitrueSpot` package into your project directory.

## Setup

### Prerequisites
- **API Credentials**: Obtain an API key and secret key from your Bitrue account. Ensure the API key has permissions for Spot trading, account queries, and withdrawals (if needed).
- **Go Environment**: Go 1.16 or later is recommended.

### Project Structure
Place the `bitrueSpot` package in your project directory or import it as a module. A typical setup might look like:

```
gox/
├── bitrueSpot/
│   └── bitrueSpot.go
├── main.go
└── go.mod
```

Initialize your Go module:

```bash
go mod init github.com/lackyband/gox
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
	"github.com/lackyband/gox/bitrueSpot"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		return
	}

	client := bitrueSpot.NewBitrueClient(apiKey, secretKey)
}
```

Set environment variables for security (these must be set before running examples):

```bash
export BITRUE_API_KEY="your_api_key"
export BITRUE_SECRET_KEY="your_secret_key"
```

### REST API Examples

#### Ping the API
Test connectivity to the Bitrue Spot API:

```go
resp, err := client.Ping()
if err != nil {
	fmt.Printf("Ping failed: %v\n", err)
} else {
	fmt.Printf("Ping response: %s\n", string(resp))
}
```

#### Retrieve Exchange Information
Get trading pairs and rules:

```go
resp, err := client.ExchangeInfo()
if err != nil {
	fmt.Printf("ExchangeInfo failed: %v\n", err)
} else {
	fmt.Printf("ExchangeInfo response: %s\n", string(resp))
}
```

#### Place a New Order
Place a limit buy order for the `BTCUSDT` trading pair:

```go
orderParams := bitrueSpot.OrderParams{
	Symbol:      "BTCUSDT",
	Side:        "BUY",
	Type:        "LIMIT",
	TimeInForce: "GTC",
	Quantity:    0.001,
	Price:       30000.0,
}
resp, err := client.NewOrder(orderParams)
if err != nil {
	fmt.Printf("NewOrder failed: %v\n", err)
} else {
	fmt.Printf("NewOrder response: %s\n", string(resp))
}
```

#### Query Account Information
Retrieve account balances and details:

```go
resp, err := client.Account(5000) // recvWindow of 5000ms
if err != nil {
	fmt.Printf("Account failed: %v\n", err)
} else {
	fmt.Printf("Account response: %s\n", string(resp))
}
```

#### Initiate a Withdrawal
Commit a withdrawal request for USDT:

```go
withdrawParams := bitrueSpot.WithdrawParams{
	Coin:      "USDT",
	Amount:    100.0,
	AddressTo: "0xYourWalletAddress",
	ChainName: "TRC20",
}
resp, err := client.WithdrawCommit(withdrawParams)
if err != nil {
	fmt.Printf("WithdrawCommit failed: %v\n", err)
} else {
	fmt.Printf("WithdrawCommit response: %s\n", string(resp))
}
```

### WebSocket Examples

#### Subscribe to Depth Data
Subscribe to order book updates for `BTCUSDT`:

```go
if err := client.StartWebSocketMarket(); err != nil {
	fmt.Printf("Failed to start WebSocket market: %v\n", err)
	return
}
defer client.StopWebSocketMarket()

depthCallback := func(message []byte) {
	var msg bitrueSpot.WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		fmt.Printf("Failed to unmarshal depth message: %v\n", err)
		return
	}
	fmt.Printf("Depth message: %+v\n", msg)
}
if err := client.SubscribeDepth("BTCUSDT", depthCallback); err != nil {
	fmt.Printf("Failed to subscribe to depth: %v\n", err)
}
```

#### Subscribe to User Order Updates
Subscribe to real-time order updates via the user data stream:

```go
listenKeyResp, err := client.CreateListenKey()
if err != nil {
	fmt.Printf("Failed to create listenKey: %v\n", err)
	return
}
var listenKeyData struct {
	ListenKey string `json:"listenKey"`
}
if err := json.Unmarshal(listenKeyResp, &listenKeyData); err != nil {
	fmt.Printf("Failed to parse listenKey: %v\n", err)
	return
}
listenKey := listenKeyData.ListenKey
fmt.Printf("Created listenKey: %s\n", listenKey)

if err := client.StartWebSocketUser(listenKey); err != nil {
	fmt.Printf("Failed to start WebSocket user: %v\n", err)
	return
}
defer client.StopWebSocketUser()

orderCallback := func(message []byte) {
	fmt.Printf("Order update message: %s\n", string(message))
}
if err := client.SubscribeUserOrderUpdate(orderCallback); err != nil {
	fmt.Printf("Failed to subscribe to order updates: %v\n", err)
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

- **Signature Requirements**: Signed endpoints (e.g., `NewOrder`, `Account`, `WithdrawCommit`) require HMAC SHA256 signatures included in the query string (`signature` parameter). The module handles this automatically, but ensure your system clock is synchronized to avoid timestamp errors.
- **WebSocket Streams**: Market data streams (e.g., `depth`) are compressed with Gzip and require decompression, which the module handles. User data streams (e.g., `user_order_update`, `user_balance_update`) require a valid `listenKey`.
- **Rate Limits**: The Spot API may return HTTP 429 if rate limits are exceeded. Implement exponential backoff or rate limiting in your application to avoid bans.
- **WebSocket Limits**: Do not subscribe to more than 100 streams per connection, and maintain no more than 100 connections per IP to avoid disconnection or IP bans.
- **Error Handling**: Check response codes and error messages (e.g., `{"code": -1121, "msg": "Invalid symbol."}`) for debugging. The module retries failed requests up to three times.
- **Symbol Format**: Use uppercase symbols (e.g., `BTCUSDT`) as required by the Bitrue Spot API.

## Example Project

A complete example tester can be found below. Save it as `main.go` and run it after setting up your environment:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"my_project/bitrueSpot"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY")
		return
	}

	client := bitrueSpot.NewBitrueClient(apiKey, secretKey)

	// Test REST API
	resp, err := client.Ping()
	if err != nil {
		fmt.Printf("Ping failed: %v\n", err)
	} else {
		fmt.Printf("Ping response: %s\n", string(resp))
	}

	// Test WebSocket
	if err := client.StartWebSocketMarket(); err != nil {
		fmt.Printf("Failed to start WebSocket: %v\n", err)
		return
	}
	defer client.StopWebSocketMarket()

	depthCallback := func(message []byte) {
		var msg bitrueSpot.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("Failed to unmarshal: %v\n", err)
			return
		}
		fmt.Printf("Depth message: %+v\n", msg)
	}
	if err := client.SubscribeDepth("BTCUSDT", depthCallback); err != nil {
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