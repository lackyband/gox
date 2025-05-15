# gox

[![Go Reference](https://pkg.go.dev/badge/github.com/lackyband/gox.svg)](https://pkg.go.dev/github.com/lackyband/gox)

A modular Go SDK for Bitrue Spot, USDT-M, and COIN-M Futures APIs.

## Repository

This is the canonical repository for `gox`: https://github.com/lackyband/gox

## Modules

- [`bitrueSpot`](./bitrueSpot/README.md): Bitrue Spot API client
- [`bitrueFutures`](./bitrueFutures/README.md): Bitrue USDT-M Futures API client
- [`bitrueMFutures`](./bitrueMFutures/README.md): Bitrue COIN-M Futures API client

## Installation

Install the SDK or any submodule using Go modules:

```bash
go get github.com/lackyband/gox/bitrueSpot
go get github.com/lackyband/gox/bitrueFutures
go get github.com/lackyband/gox/bitrueMFutures
```

## Usage

Each module provides its own README with usage examples. See the respective subdirectories for documentation and code samples.

## Contributing

Contributions are welcome! Please open issues or submit pull requests on [GitHub](https://github.com/lackyband/gox).

## License

MIT License. See [LICENSE](./LICENSE) for details.
