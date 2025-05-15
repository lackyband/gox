module github.com/lackyband/gox/cmd/tester

replace github.com/lackyband/gox/bitrueFutures => ../../bitrueFutures

replace github.com/lackyband/gox/bitrueSpot => ../../bitrueSpot

go 1.22.2

require (
	github.com/lackyband/gox/bitrueFutures v0.0.0-00010101000000-000000000000
	github.com/lackyband/gox/bitrueSpot v0.0.0-00010101000000-000000000000
)

require github.com/gorilla/websocket v1.5.3 // indirect
