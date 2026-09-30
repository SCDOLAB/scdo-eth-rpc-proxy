// Package config holds all runtime configuration for the eth-rpc proxy.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ListenAddr    string
	ChainID       uint64
	DefaultShard  uint
	PinShard      uint
	FinalityDepth uint64
	ShardRPC      map[uint]string
}

func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:    getEnv("SCDO_ETH_LISTEN", ":8038"),
		ChainID:       getEnvUint("SCDO_CHAIN_ID", 0x238), // 568
		DefaultShard:  uint(getEnvUint("SCDO_DEFAULT_SHARD", 1)),
		PinShard:      uint(getEnvUint("SCDO_PIN_SHARD", 1)),
		FinalityDepth: getEnvUint("SCDO_FINALITY_DEPTH", 12),
		ShardRPC: map[uint]string{
			1: getEnv("SCDO_SHARD1_RPC", "http://127.0.0.1:8037"),
			2: getEnv("SCDO_SHARD2_RPC", "http://127.0.0.1:8037"),
			3: getEnv("SCDO_SHARD3_RPC", "http://127.0.0.1:8037"),
			4: getEnv("SCDO_SHARD4_RPC", "http://127.0.0.1:8037"),
		},
	}
	if cfg.DefaultShard < 1 || cfg.DefaultShard > 4 {
		return nil, fmt.Errorf("invalid SCDO_DEFAULT_SHARD=%d, must be 1..4", cfg.DefaultShard)
	}
	return cfg, nil
}

func (c *Config) ShardEndpoint(shard uint) (string, error) {
	ep, ok := c.ShardRPC[shard]
	if !ok {
		return "", fmt.Errorf("no rpc endpoint configured for shard %d", shard)
	}
	return ep, nil
}

func (c *Config) ResolveShard(derived uint) uint {
	if c.PinShard > 0 {
		return c.PinShard
	}
	return derived
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvUint(key string, def uint64) uint64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseUint(v, 0, 64); err == nil {
			return n
		}
	}
	return def
}
