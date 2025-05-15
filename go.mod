module github.com/lackyband/gox

go 1.22.2

replace github.com/lackyband/gox/bitrueFutures => ./bitrueFutures

replace github.com/lackyband/gox/bitrueSpot => ./bitrueSpot

require (
	github.com/gorilla/websocket v1.5.3
	github.com/lackyband/gox/bitrueFutures v0.0.0-00010101000000-000000000000
)
