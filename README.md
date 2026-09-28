# scdo-eth-rpc-proxy

> **Archived proof of concept, not used in production.** It was never deployed. `eth_call` is not implemented, `eth_gasPrice` / `eth_estimateGas` / `eth_getCode` return fixed values, and it needs a patched go-scdo node that does not exist. Chain ID 5680 belongs to SCDO **shard 0**, which is a native EVM chain and needs no proxy: use `https://scdoscan.io/rpc/0` (see [SCDOLAB/scdo-shard0](https://github.com/SCDOLAB/scdo-shard0)). Do not add this proxy to MetaMask with chain ID 5680.

> **已归档的概念验证，生产未使用。** 功能不完整。chainId 5680 是 shard0 的，shard0 直接用 https://scdoscan.io/rpc/0 即可。

把以太坊 `eth_*` JSON-RPC 调用翻译成 SCDO 原生 `scdo_*` 调用的轻量代理。
对外呈现单一逻辑链，对内按分片路由到 4 个 SCDO 节点。
目标：让 MetaMask / Hardhat / Foundry 无需改造即可接入 SCDO。

## 架构

```
MetaMask / Hardhat / Blockscout
    │  eth_* (JSON-RPC 2.0, EVM format)
    ▼
┌─────────────────────────────────┐
│  scdo-eth-rpc-proxy (:8038)     │
│  rpc mux → addr shard router    │
└──────┬──────┬──────┬──────┬────┘
       ▼      ▼      ▼      ▼
    shard1 shard2 shard3 shard4
    scdo_* scdo_* scdo_* scdo_*
```

## 部署

```bash
export SCDO_CHAIN_ID=0x1630                 # 5680
export SCDO_ETH_LISTEN=:8038
export SCDO_DEFAULT_SHARD=1
export SCDO_PIN_SHARD=1                     # PoC: pin all traffic to shard1
export SCDO_SHARD1_RPC=http://node1:8037
export SCDO_SHARD2_RPC=http://node2:8037
export SCDO_SHARD3_RPC=http://node3:8037
export SCDO_SHARD4_RPC=http://node4:8037

go build -o scdo-eth-rpc-proxy .
./scdo-eth-rpc-proxy
```

MetaMask → Add Network:
- Network Name: `SCDO Mainnet`
- RPC URL: `http://<host>:8038`
- Chain ID: `5680`
- Currency Symbol: `SCDO`
- **Important**: Settings → Advanced → disable "EIP-1559" (use legacy gas)

## 接口映射

| eth_* 方法 | 后端 scdo_* | 状态 |
|---|---|---|
| web3_clientVersion | — | ✅ fixed string |
| net_version | — | ✅ chainID decimal |
| eth_chainId | — | ✅ 0x1630 |
| eth_syncing | — | ✅ false |
| eth_accounts | — | ✅ [] |
| eth_blockNumber | scdo_getBlockCount | ✅ |
| eth_getBalance | scdo_getBalance | ✅ |
| eth_getTransactionCount | scdo_getAccountNonce | ✅ |
| eth_sendRawTransaction | scdo_sendRawTransaction | ✅ byte-passthrough |
| eth_gasPrice | — | ⚠️ stub 1 Gwei |
| eth_estimateGas | — | ⚠️ stub 21000 |
| eth_getCode | — | ⚠️ stub 0x |
| eth_call | — | ❌ not implemented |

## 关键设计

- **不重签交易**：收到 MetaMask 签好的 raw tx 后，解 RLP 找 `to` 地址路由分片，原始字节原样转发。
- **地址透传**：MetaMask 的 keccak 地址 = 链上地址，节点 patch 后接受任意 0x+40hex。
- **零外部依赖**：纯 Go 标准库，`go build` 直接出二进制。
