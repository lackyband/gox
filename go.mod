module github.com/lackyband/gox

go 1.22.2

require (
	github.com/gorilla/websocket v1.5.3
)

replace github.com/lackyband/gox/bitrue => ./bitrue
replace github.com/lackyband/gox/wsutil/wsmanager => ./wsutil/wsmanager
