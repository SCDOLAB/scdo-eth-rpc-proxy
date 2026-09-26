// scdo-eth-rpc-proxy: a light JSON-RPC 2.0 proxy that translates Ethereum
// eth_* calls into scdo_* calls against a sharded go-scdo backend.
package main

import (
	"log"
	"net/http"

	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/config"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/ethapi"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/rpcmux"
	"github.com/scdo-hub/scdo-eth-rpc-proxy/internal/scdo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	mux := rpcmux.New()
	be := &ethapi.Backend{
		Cfg: cfg,
		Cli: scdo.New(cfg),
	}
	be.RegisterAll(mux)

	log.Printf("scdo-eth-rpc-proxy listening on %s", cfg.ListenAddr)
	log.Printf("  chainID       = %#x", cfg.ChainID)
	log.Printf("  default shard = %d", cfg.DefaultShard)
	log.Printf("  finality depth= %d blocks", cfg.FinalityDepth)
	for shard, ep := range cfg.ShardRPC {
		log.Printf("  shard %d -> %s", shard, ep)
	}
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}
