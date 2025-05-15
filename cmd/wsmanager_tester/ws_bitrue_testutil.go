package tester

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const (
	bitrueFuturesRestAPI = "https://fapi.bitrue.com/fapi/v1/contracts"
	bitrueFuturesWSURL   = "wss://fmarket-ws.bitrue.com/kline-api/ws"
	maxSubsPerConn       = 50
)

// fetchFuturesContracts fetches all active contracts from Bitrue REST API
func fetchFuturesContracts(t *testing.T) []string {
	req, err := http.NewRequest("GET", bitrueFuturesRestAPI, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to fetch contracts: %v", err)
	}
	defer resp.Body.Close()

	var contracts []struct {
		Symbol string `json:"symbol"`
		Status int    `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contracts); err != nil {
		t.Fatalf("Failed to decode contracts: %v", err)
	}

	symbols := make([]string, 0, len(contracts))
	for _, c := range contracts {
		if c.Status == 1 {
			symbols = append(symbols, c.Symbol)
		}
	}
	if len(symbols) == 0 {
		t.Fatalf("No active contracts found from REST API")
	}
	return symbols
}

// decompressGzip decompresses gzip-compressed data
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
