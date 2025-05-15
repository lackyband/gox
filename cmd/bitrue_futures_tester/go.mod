module github.com/lackyband/gox/cmd/bitrue_futures_tester

go 1.22.2

replace github.com/lackyband/gox/bitrueFutures => ../../bitrueFutures

require github.com/lackyband/gox/bitrueFutures v0.0.0-00010101000000-000000000000

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
)
