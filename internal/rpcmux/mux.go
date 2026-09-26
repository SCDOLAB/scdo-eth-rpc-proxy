// Package rpcmux is a tiny JSON-RPC 2.0 server router.
package rpcmux

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handler func(params []json.RawMessage) (interface{}, error)

type Mux struct {
	handlers map[string]Handler
}

func New() *Mux {
	return &Mux{handlers: map[string]Handler{}}
}

func (m *Mux) Register(method string, h Handler) {
	m.handlers[method] = h
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32600, Message: "only POST is supported"},
		})
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: "parse error: " + err.Error()},
		})
		return
	}

	h, ok := m.handlers[req.Method]
	if !ok {
		writeJSON(w, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32601, Message: "method not found: " + req.Method},
			ID:      req.ID,
		})
		return
	}

	result, err := h(req.Params)
	if err != nil {
		writeJSON(w, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32000, Message: err.Error()},
			ID:      req.ID,
		})
		return
	}
	writeJSON(w, rpcResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	})
}

func writeJSON(w http.ResponseWriter, v rpcResponse) {
	_ = json.NewEncoder(w).Encode(v)
}

func TrimmedMethod(m string) string {
	return strings.TrimPrefix(strings.TrimPrefix(m, "eth_"), "scdo_")
}
