# Ethereum 交互工具

一个用于与以太坊区块链交互的 Go 命令行工具，支持查询区块信息、发送 ETH 交易以及与智能合约交互。

## 功能特性

- **区块查询**: 根据区块号查询区块详细信息
- **ETH 转账**: 向指定地址发送 ETH (支持 EIP-1559 动态费用)
- **合约交互**: 与 Counter 合约进行交互
  - `number`: 查询当前数值
  - `setNumber`: 设置数值
  - `increment`: 数值 +1

## 环境变量

使用前需要设置以下环境变量：

```bash
export ETH_RPC_URL="https://your-ethereum-rpc-url"  # 以太坊 RPC 节点 URL
export SENDER_PRIVATE_KEY="0x..."                     # 发送方私钥 (发送交易时需要)
```

## 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--blockNum` | uint64 | 0 | 要查询的区块号 |
| `--send` | bool | false | 启用发送交易模式 |
| `--to` | string | "" | 接收方地址 (send 模式必需) |
| `--amount` | float64 | 0 | 发送的 ETH 数量 (send 模式必需) |
| `--mode` | string | "number" | 操作模式: number, setNumber, increment |
| `--contract` | string | "" | Counter 合约地址 |
| `--num` | uint64 | 0 | setNumber 模式要设置的数值 |

## 使用示例

### 1. 查询区块信息

```bash
go run main.go --blockNum 123456
```

输出包括：
- 区块号、哈希、父哈希
- 时间戳
- Gas 使用情况
- 交易数量
- 区块根哈希
- 验证者地址 (Coinbase)

### 2. 发送 ETH 交易

```bash
go run main.go --send --to 0x74c56945C0E54D264Cbaa595eD16f971D8471707 --amount 0.01
```

### 3. 查询合约数值

```bash
go run main.go --mode number --contract 0x1b9bDfB2AB104Dd77e9a3bc6F62ebf1c248E246c
```

### 4. 设置合约数值

```bash
go run main.go --mode setNumber --contract 0x1b9bDfB2AB104Dd77e9a3bc6F62ebf1c248E246c --num 123456
```

### 5. 增加合约数值

```bash
go run main.go --mode increment --contract 0x1b9bDfB2AB104Dd77e9a3bc6F62ebf1c248E246c
```

## 技术细节

### ABI 接口

程序内置了 Counter 合约的 ABI：

```json
[
  {"name": "increment", "type": "function", "stateMutability": "nonpayable"},
  {"name": "number", "type": "function", "stateMutability": "view", "outputs": [{"type": "uint256"}]},
  {"name": "setNumber", "type": "function", "stateMutability": "nonpayable", "inputs": [{"name": "newNumber", "type": "uint256"}]}
]
```

### 交易费用

- 使用 EIP-1559 动态费用机制
- Gas Fee Cap = Base Fee * 2 + Gas Tip Cap
- 普通 ETH 转账: Gas Limit = 21000
- 合约调用: 自动估算 Gas + 20% 缓冲

### 重试机制

- 区块查询失败时自动重试最多 3 次
- 每次重试间隔递增 (500ms, 1000ms, 1500ms)
- 交易发送后等待确认，最长 2 分钟

## 依赖

- Go 1.16+
- [go-ethereum](https://github.com/ethereum/go-ethereum)

安装依赖：
```bash
go mod tidy
```