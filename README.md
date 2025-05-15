# gox

[![Go Reference](https://pkg.go.dev/badge/github.com/lackyband/gox.svg)](https://pkg.go.dev/github.com/lackyband/gox)

A modular and extensible Go SDK for cryptocurrency exchange APIs.

Currently, gox provides full support for Bitrue Spot, USDT-M, and COIN-M Futures APIs, as well as Bitrue Copy Trading. The SDK is designed to be easily extended to support additional exchanges in the future.

## Repository

This is the canonical repository for `gox`: https://github.com/lackyband/gox

## Modules

- [`bitrueSpot`](./bitrueSpot/README.md): Bitrue Spot API client
- [`bitrueFutures`](./bitrueFutures/README.md): Bitrue USDT-M Futures API client
- [`bitrueMFutures`](./bitrueMFutures/README.md): Bitrue COIN-M Futures API client
- [`bitrueCopyTrade`](./bitrueCopyTrade/README.md): Bitrue Copy Trading API client

## Installation

Install the SDK or any submodule using Go modules:

```bash
go get github.com/lackyband/gox/bitrueSpot
go get github.com/lackyband/gox/bitrueFutures
go get github.com/lackyband/gox/bitrueMFutures
```

## Usage

Each module provides its own README with usage examples. See the respective subdirectories for documentation and code samples.

## Extensibility & Multi-Exchange Support

gox is designed to be modular and extensible. While the initial focus is on Bitrue APIs (including Spot, Futures, and Copy Trading), the SDK architecture allows for easy integration of additional exchanges in the future. Support for other major exchanges is planned—stay tuned for updates!

## Contributing

Contributions are welcome! Please open issues or submit pull requests on [GitHub](https://github.com/lackyband/gox).

## License

MIT License. See [LICENSE](./LICENSE) for details.
