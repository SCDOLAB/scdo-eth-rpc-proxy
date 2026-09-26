// Package ethapi implements the eth_* JSON-RPC methods by translating
// them to scdo_* calls against the sharded backend.
package ethapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/addr"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/config"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/rlp"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/rpcmux"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/scdo"
)

type Backend struct {
	Cfg *config.Config
	Cli *scdo.Client
}

func (b *Backend) RegisterAll(mux *rpcmux.Mux) {
	mux.Register("web3_clientVersion", b.clientVersion)
	mux.Register("net_version", b.netVersion)
	mux.Register("eth_chainId", b.chainID)
	mux.Register("eth_syncing", b.syncing)
	mux.Register("eth_accounts", b.accounts)
	mux.Register("eth_blockNumber", b.blockNumber)
	mux.Register("eth_getBalance", b.getBalance)
	mux.Register("eth_getTransactionCount", b.getTxCount)
	mux.Register("eth_sendRawTransaction", b.sendRawTx)
	mux.Register("eth_gasPrice", b.gasPrice)
	mux.Register("eth_estimateGas", b.estimateGas)
	mux.Register("eth_getCode", b.getCode)
	mux.Register("eth_call", b.call)
}

// ── Static responses ─────────────────────────────────────

func (b *Backend) clientVersion(params []json.RawMessage) (interface{}, error) {
	return "scdo-eth-rpc-proxy/v0.1.0", nil
}

func (b *Backend) netVersion(params []json.RawMessage) (interface{}, error) {
	return fmt.Sprintf("%d", b.Cfg.ChainID), nil
}

func (b *Backend) chainID(params []json.RawMessage) (interface{}, error) {
	return toHex(b.Cfg.ChainID), nil
}

func (b *Backend) syncing(params []json.RawMessage) (interface{}, error) {
	return false, nil
}

func (b *Backend) accounts(params []json.RawMessage) (interface{}, error) {
	return []string{}, nil
}

// ── Block number ──────────────────────────────────────────

func (b *Backend) blockNumber(params []json.RawMessage) (interface{}, error) {
	result, err := b.Cli.Call(b.Cfg.DefaultShard, "scdo_getBlockCount", []interface{}{})
	if err != nil {
		return nil, err
	}
	var h json.Number
	if err := json.Unmarshal(result, &h); err != nil {
		var s string
		json.Unmarshal(result, &s)
		n, _ := strconv.ParseUint(s, 10, 64)
		return toHex(n), nil
	}
	n, _ := h.Int64()
	return toHex(uint64(n)), nil
}

// ── Balance / nonce ──────────────────────────────────────

func (b *Backend) getBalance(params []json.RawMessage) (interface{}, error) {
	var account string
	if err := json.Unmarshal(params[0], &account); err != nil {
		return nil, fmt.Errorf("invalid account param")
	}
	shard := b.routeShard(account)
	result, err := b.Cli.Call(shard, "scdo_getBalance", []interface{}{account, "", -1})
	if err != nil {
		return nil, err
	}
	var bal json.Number
	if err := json.Unmarshal(result, &bal); err != nil {
		var s string
		json.Unmarshal(result, &s)
		n, _ := strconv.ParseUint(s, 10, 64)
		return toHex(n), nil
	}
	n, _ := bal.Int64()
	return toHex(uint64(n)), nil
}

func (b *Backend) getTxCount(params []json.RawMessage) (interface{}, error) {
	var account string
	if err := json.Unmarshal(params[0], &account); err != nil {
		return nil, fmt.Errorf("invalid account param")
	}
	shard := b.routeShard(account)
	result, err := b.Cli.Call(shard, "scdo_getAccountNonce", []interface{}{account, "", -1})
	if err != nil {
		return nil, err
	}
	var nonce json.Number
	if err := json.Unmarshal(result, &nonce); err != nil {
		var s string
		json.Unmarshal(result, &s)
		n, _ := strconv.ParseUint(s, 10, 64)
		return toHex(n), nil
	}
	n, _ := nonce.Int64()
	return toHex(uint64(n)), nil
}

// ── Send raw transaction ──────────────────────────────────

func (b *Backend) sendRawTx(params []json.RawMessage) (interface{}, error) {
	var dataHex string
	if err := json.Unmarshal(params[0], &dataHex); err != nil {
		return nil, fmt.Errorf("invalid raw tx param")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("bad hex: %v", err)
	}

	// Parse "to" from RLP to route to correct shard
	// (bytes are forwarded as-is, no re-signing)
	to, err := rlp.ParseToAddress(raw)
	shard := b.Cfg.DefaultShard
	if err == nil && to != nil {
		shard = b.Cfg.ResolveShard(addr.DeriveShard(to))
	}

	result, err := b.Cli.Call(shard, "scdo_sendRawTransaction", []interface{}{dataHex})
	if err != nil {
		return nil, err
	}
	return string(result), nil
}

// ── Stub methods ─────────────────────────────────────────

func (b *Backend) gasPrice(params []json.RawMessage) (interface{}, error) {
	return "0x3b9aca00", nil // 1 Gwei
}

func (b *Backend) estimateGas(params []json.RawMessage) (interface{}, error) {
	return "0x5208", nil // 21000
}

func (b *Backend) getCode(params []json.RawMessage) (interface{}, error) {
	return "0x", nil
}

func (b *Backend) call(params []json.RawMessage) (interface{}, error) {
	return nil, fmt.Errorf("eth_call not implemented yet")
}

// ── Helpers ──────────────────────────────────────────────

func (b *Backend) routeShard(ethAddr string) uint {
	a, err := hex.DecodeString(strings.TrimPrefix(ethAddr, "0x"))
	if err != nil {
		return b.Cfg.DefaultShard
	}
	return b.Cfg.ResolveShard(addr.DeriveShard(a))
}

func toHex(n uint64) string {
	if n == 0 {
		return "0x0"
	}
	return "0x" + big.NewInt(0).SetUint64(n).Text(16)
}
