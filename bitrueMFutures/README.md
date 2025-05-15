# BitrueMFutures Module

The `bitrueMFutures` module is a Go library for interacting with the Bitrue COIN-M Futures API. It supports REST API endpoints for market data, trading, account management, and fund transfers, as well as WebSocket streams for real-time market and user data. This module is designed for developers building trading bots or applications that require programmatic access to Bitrue's COIN-M Futures platform.

## Installation

To use the `bitrueMFutures` module, ensure you have Go installed. Add the module to your project and install the required dependency:

```bash
go get github.com/gorilla/websocket
```

Clone or include the `bitrueMFutures` package in your Go module:

```bash
# If the module is hosted on GitHub, replace <repository> with the actual path
go get github.com/lackyband/gox/bitrueMFutures
```

Alternatively, copy the `bitrueMFutures` package into your project directory.

## Setup

### Prerequisites
- **API Credentials**: Obtain an API key and secret key from your Bitrue account. Ensure the API key has permissions for COIN-M Futures trading and account queries.
- **Go Environment**: Go 1.16 or later is recommended.

### Project Structure
Place the `bitrueMFutures` package in your project directory or import it as a module. A typical setup might look like:

```
gox/
├── bitrueMFutures/
│   └── bitrueMFutures.go
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
Create a `BitrueMClient` instance with your API credentials:

```go
package main

import (
	"fmt"
	"os"
	"github.com/lackyband/gox/bitrueMFutures"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY environment variables")
		return
	}

	client := bitrueMFutures.NewBitrueMClient(apiKey, secretKey)
}
```

Set environment variables for security (these must be set before running examples):

```bash
export BITRUE_API_KEY="your_api_key"
export BITRUE_SECRET_KEY="your_secret_key"
```

### REST API Examples

#### Ping the API
Test connectivity to the Bitrue COIN-M Futures API:

```go
resp, err := client.Ping()
if err != nil {
	fmt.Printf("Ping failed: %v\n", err)
} else {
	fmt.Printf("Ping response: %s\n", string(resp))
}
```

#### Retrieve Contracts
Get information about available COIN-M Futures contracts:

```go
resp, err := client.Contracts()
if err != nil {
	fmt.Printf("Contracts failed: %v\n", err)
} else {
	fmt.Printf("Contracts response: %s\n", string(resp))
}
```

#### Place a New Order
Place a limit order for the `H-HT-USD` contract:

```go
orderParams := bitrueMFutures.OrderParams{
	ContractName: "H-HT-USD",
	Side:         "BUY",
	Type:         "LIMIT",
	PositionType: 1, // Crossed position
	Open:         "OPEN",
	Volume:       1.0,
	Price:        0.1,
	Leverage:     10,
	ConditionOrder: false,
}
resp, err := client.NewOrder(orderParams)
if err != nil {
	fmt.Printf("NewOrder failed: %v\n", err)
} else {
	fmt.Printf("NewOrder response: %s\n", string(resp))
}
```

#### Modify Position Margin
Adjust the isolated position margin for a contract:

```go
marginParams := bitrueMFutures.ModifyPositionMarginParams{
	ContractName: "H-HT-USD",
	Amount:       10.0,
}
resp, err := client.ModifyPositionMargin(marginParams)
if err != nil {
	fmt.Printf("ModifyPositionMargin failed: %v\n", err)
} else {
	fmt.Printf("ModifyPositionMargin response: %s\n", string(resp))
}
```

#### Transfer Funds
Transfer funds between the wallet and COIN-M Futures account:

```go
resp, err := client.FuturesTransfer("BTC", 0.01, "TRANSFER_IN", "")
if err != nil {
	fmt.Printf("FuturesTransfer failed: %v\n", err)
} else {
	fmt.Printf("FuturesTransfer response: %s\n", string(resp))
}
```

### WebSocket Examples

#### Subscribe to Kline Data
Subscribe to 1-minute kline data for `H-HT-USD`:

```go
if err := client.StartWebSocketMarket(); err != nil {
	fmt.Printf("Failed to start WebSocket market: %v\n", err)
	return
}
defer client.StopWebSocketMarket()

klineCallback := func(message []byte) {
	var msg bitrueMFutures.WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		fmt.Printf("Failed to unmarshal kline message: %v\n", err)
		return
	}
	fmt.Printf("Kline message: %+v\n", msg)
	// Note: Multiply msg.Tick.(map[string]interface{})["amount"] and ["vol"]
	// by the contract size from /dapi/v1/contracts
}
if err := client.SubscribeKline("H-HT-USD", "1min", klineCallback); err != nil {
	fmt.Printf("Failed to subscribe to kline: %v\n", err)
}
```

#### Request Historical Kline Data
Request historical kline data for `H-HT-USD`:

```go
if err := client.RequestWebSocketData("market_$symbol_kline_$interval", "H-HT-USD", "1min", 0, "1506602880", true); err != nil {
	fmt.Printf("Failed to request historical kline: %v\n", err)
}
```

#### Subscribe to User Data Stream
Subscribe to user data updates:

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

- **Contract Size Multiplication**: For WebSocket market data (e.g., `depth`, `trade`, `kline`, `ticker`), the `amount` and `vol` fields must be multiplied by the contract size (`multiplier`) from the `/dapi/v1/contracts` endpoint to get the correct contract values. This is a manual step required by the Bitrue API.
- **Signature Requirements**: Signed endpoints (e.g., `NewOrder`, `Account`, `FuturesTransfer`) require HMAC SHA256 signatures passed via `X-CH-SIGN` and `X-CH-TS` headers. Ensure your system clock is synchronized to avoid timestamp errors.
- **Rate Limits**: The API may return HTTP 429 if rate limits are exceeded. Implement exponential backoff or rate limiting in your application to avoid bans.
- **WebSocket Limits**: Do not subscribe to more than 100 streams per connection, and maintain no more than 100 connections per IP to avoid disconnection or IP bans.
- **Error Handling**: Check response codes and error messages (e.g., `{"code": -1121, "msg": "Invalid symbol."}`) for debugging. The module retries failed requests up to three times.
- **Contract Naming**: Use contract names as returned by `/dapi/v1/contracts` (e.g., `H-HT-USD`, `E-BTC-USD`), ensuring correct casing.

## Example Project

A complete example tester can be found below. Save it as `main.go` and run it after setting up your environment:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"my_project/bitrueMFutures"
)

func main() {
	apiKey := os.Getenv("BITRUE_API_KEY")
	secretKey := os.Getenv("BITRUE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		fmt.Println("Please set BITRUE_API_KEY and BITRUE_SECRET_KEY")
		return
	}

	client := bitrueMFutures.NewBitrueMClient(apiKey, secretKey)

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

	klineCallback := func(message []byte) {
		var msg bitrueMFutures.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("Failed to unmarshal: %v\n", err)
			return
		}
		fmt.Printf("Kline message: %+v\n", msg)
	}
	if err := client.SubscribeKline("H-HT-USD", "1min", klineCallback); err != nil {
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