# wsmanager: WebSocket Pool Manager for gox

This package provides a reusable, thread-safe WebSocket connection pool and subscription manager for the gox SDK. It is designed to be used by all exchange modules (Futures, Spot, CopyTrade, etc.) to efficiently handle large numbers of subscriptions, manage callbacks, and automatically handle reconnections.

## Features
- Pooling of WebSocket connections, each with a configurable maximum number of subscriptions
- Automatic allocation of new connections when subscription limits are reached
- Per-subscription callback routing
- Automatic cleanup of idle connections
- Thread-safe

## Usage Example

```
go
import "github.com/lackyband/gox/wsutil/wsmanager"

// Define your WebSocket client factory (exchange-specific)
clientFactory := func() (*wsmanager.WebSocketClient, error) {
    // ... create and connect a new WebSocketClient ...
}

// Create the pool manager
manager := wsmanager.NewWebSocketPoolManager(50, clientFactory)

// Subscribe with a callback
subID, err := manager.Subscribe("kline", "BTC_USDT", "1m", func(msg []byte) {
    // handle kline message
})

// Unsubscribe later
err = manager.Unsubscribe("kline", subID)

// Cleanup idle connections
manager.Cleanup()
```

## Extending for an Exchange
- Use the pool manager in your exchange client
- Implement exchange-specific message parsing in the callback
- Use the subscription ID for precise unsubscription

## License
MIT
