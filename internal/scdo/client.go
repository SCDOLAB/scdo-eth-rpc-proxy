// Package scdo is a minimal JSON-RPC client for go-scdo.
package scdo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/config"
)

type Client struct {
	cfg *config.Config
	mu  sync.Mutex
	id  uint64
}

func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Call(shard uint, method string, params []interface{}) (json.RawMessage, error) {
	ep, err := c.cfg.ShardEndpoint(shard)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.id++
	id := c.id
	c.mu.Unlock()

	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      id,
	})

	resp, err := http.Post(ep, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("bad json-rpc response: %s", respBody)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("scdo error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}
