package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
)
const erc20ABIJSON = `[
  {
    "type": "function",
    "name": "increment",
    "inputs": [],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "number",
    "inputs": [],
    "outputs": [
      {
        "name": "",
        "type": "uint256",
        "internalType": "uint256"
      }
    ],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "setNumber",
    "inputs": [
      {
        "name": "newNumber",
        "type": "uint256",
        "internalType": "uint256"
      }
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  }
]
`
// go run main.go --blockNum 123456
//go run main.go --send --to  0x74c56945C0E54D264Cbaa595eD16f971D8471707 --amount 0.01
//go run main.go --mode setNumber --contract 0x74c56945C0E54D264Cbaa595eD16f971D8471707 --num 123456
 //go run main.go --mode number --contract 0x1b9bDfB2AB104Dd77e9a3bc6F62ebf1c248E246c
 //go run main.go --mode increment --contract 0x1b9bDfB2AB104Dd77e9a3bc6F62ebf1c248E246c
func main() {
	// 查询区块信息,获取区块号
	blockNum := flag.Uint64("blockNum", 0, "block number to query")
	sendMode := flag.Bool("send", false, "enable send transaction mode")
	toAddrHex := flag.String("to", "", "recipient address (required for send mode)")
	amountEth := flag.Float64("amount", 0, "amount in ETH (required for send mode)")
	
	mode := flag.String("mode", "number", "operation mode: number, setNumber, increment")
	contractHex := flag.String("contract", "", "Counter contract address")
	newNum := flag.Uint64("num", 0, "block number to query")
	flag.Parse()
		// 连接到以太坊节点
	
	// 指定区块
	if *blockNum > 0 {
		num := big.NewInt(0).SetUint64(*blockNum)
		block, err := fetchBlockWithRetry( num, 3)
		if err != nil {
			log.Fatalf("failed to get block %d: %v", *blockNum, err)
		}
		printBlockInfo(fmt.Sprintf("Block %d", *blockNum), block)
	}
	if *sendMode {
		if *toAddrHex == "" || *amountEth <= 0 {
			log.Fatal("send mode requires --to and --amount flags")
		}
		sendTransaction(*toAddrHex, *amountEth)
	}



	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		log.Fatalf("failed to parse ABI: %v", err)
	}
	switch *mode {
	case "number":
		getNumber(ctx, client, parsedABI, *contractHex)
	case "setNumber":
		setNumber(ctx, client, parsedABI, *contractHex, *newNum)
	case "increment":
		inc(ctx, client, parsedABI, *contractHex)
	default:
		log.Fatalf("unknown mode: %s (use: balance, transfer, or parse-event)", *mode)
	}

	}
	// 查询区块信息
	func fetchBlockWithRetry(blockNum *big.Int,maxRetries int)(*types.Block, error){
		rpcUrl := os.Getenv("ETH_RPC_URL")
		if rpcUrl == "" {
			fmt.Println("ETH_RPC_URL is not set")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client, err := ethclient.DialContext(ctx, rpcUrl)
		if err != nil {
			log.Fatalf("Failed to connect to the Ethereum node: %v", err)
		}
		defer client.Close()
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			retryCtx,cancel :=	context.WithTimeout(ctx, 10*time.Second)
			block, err := client.BlockByNumber(retryCtx, blockNum)
			cancel()

			if err != nil {
				lastErr = err
			} else {
				return block, nil
			}
			if i < maxRetries-1 {
			backoff := time.Duration(i+1) * 500 * time.Millisecond
			log.Printf("[WARN] failed to fetch block %s, retry %d/%d after %v: %v",
				blockNum.String(), i+1, maxRetries, backoff, err)
			time.Sleep(backoff)
			}

		}
		return nil,  fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)


	} 


	func printBlockInfo(title string, block *types.Block) {
	fmt.Println("======================================")
	fmt.Println(title)
	fmt.Println("======================================")
	fmt.Printf("Block: %+v\n", block)

	// 基本信息
	fmt.Printf("Number       : %d\n", block.Number().Uint64())
	fmt.Printf("Hash         : %s\n", block.Hash().Hex())
	fmt.Printf("Parent Hash  : %s\n", block.ParentHash().Hex())

	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	fmt.Printf("Time         : %s\n", blockTime.Format(time.RFC3339))
	fmt.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))

	// Gas 信息
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()
	gasUsagePercent := float64(gasUsed) / float64(gasLimit) * 100
	fmt.Printf("Gas Used     : %d (%.2f%%)\n", gasUsed, gasUsagePercent)
	fmt.Printf("Gas Limit    : %d\n", gasLimit)

	// 交易信息
	txCount := len(block.Transactions())
	fmt.Printf("Tx Count     : %d\n", txCount)

	// 区块根信息（Merkle 树根）
	fmt.Printf("State Root   : %s\n", block.Root().Hex())
	fmt.Printf("Tx Root      : %s\n", block.TxHash().Hex())
	fmt.Printf("Receipt Root : %s\n", block.ReceiptHash().Hex())

	// 区块大小估算（简化版，实际大小还包括其他字段）
	if txCount > 0 {
		fmt.Printf("\nFirst Tx Hash: %s\n", block.Transactions()[0].Hash().Hex())
		if txCount > 1 {
			fmt.Printf("Last Tx Hash : %s\n", block.Transactions()[txCount-1].Hash().Hex())
		}
	}

	// 难度信息（PoW 相关，PoS 后基本固定）
	fmt.Printf("Difficulty   : %s\n", block.Difficulty().String())

	// 区块奖励相关信息
	coinbase := block.Coinbase()
	if coinbase != (common.Address{}) {
		fmt.Printf("Coinbase     : %s\n", coinbase.Hex())
	}

	fmt.Println("======================================")
	fmt.Println()
}


func trim0x(hex string) string {
	return strings.TrimPrefix(hex, "0x")
}

func sendTransaction( toAddrHex string, amountEth float64) {
	rpcUrl := os.Getenv("ETH_RPC_URL")
	if rpcUrl == "" {
		fmt.Println("ETH_RPC_URL is not set")
	}
	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
			log.Fatal("SENDER_PRIVATE_KEY is not set (required for send mode)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	// 解析私钥
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	// 获取发送方地址
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddrHex)

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}

	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get header: %v", err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	// 估算 Gas Limit（普通转账固定为 21000）
	gasLimit := uint64(21000)

	// 转换 ETH 金额为 Wei
	// amountEth * 1e18
	amountWei := new(big.Float).Mul(
		big.NewFloat(amountEth),
		big.NewFloat(1e18),
	)
	valueWei, _ := amountWei.Int(nil)

	// 检查余额是否足够
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}

	// 计算总费用：value + gasFeeCap * gasLimit
	totalCost := new(big.Int).Add(
		valueWei,
		new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit))),
	)

	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("insufficient balance: have %s wei, need %s wei", balance.String(), totalCost.String())
	}

	// 构造交易（EIP-1559 动态费用交易）
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      nil,
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	// 发送交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
//	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
//	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())
	}


	
// handleBalanceOf 查询 ERC-20 代币余额
func getNumber(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contractHex string) {
	if contractHex == ""  {
		log.Fatal("missing --contract or --address flag for balance mode")
	}

	contractAddr := common.HexToAddress(contractHex)
	

	// 编码 balanceOf 调用数据
	data, err := parsedABI.Pack("number")
	if err != nil {
		log.Fatalf("failed to pack data: %v", err)
	}

	callMsg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	// 执行只读调用
	output, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		log.Fatalf("CallContract error: %v", err)
	}

	// 解码返回值
	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "number", output)
	if err != nil {
		log.Fatalf("failed to unpack output: %v", err)
	}

	fmt.Printf("Contract : %s\n", contractAddr.Hex())
	fmt.Printf("getNumber  : %s (raw uint256)\n", balance.String())
}


func setNumber(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contractHex string, number uint64) {
	if contractHex == ""  {
		log.Fatal("missing --contract, --to, or --amount flag for transfer mode")
	}

	// 检查私钥环境变量
	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for transfer mode)")
	}

	// 解析私钥
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	// 获取发送方地址
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

	contractAddr := common.HexToAddress(contractHex)

	

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	// 编码 transfer 调用数据
	// transfer(address to, uint256 value)
	callData, err := parsedABI.Pack("setNumber", big.NewInt(0).SetUint64(number))
	if err != nil {
		log.Fatalf("failed to pack transfer data: %v", err)
	}

	// 估算 Gas Limit（合约调用需要更多 Gas）
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddr,
		To:   &contractAddr,
		Data: callData,
	})
	if err != nil {
		log.Fatalf("failed to estimate gas: %v", err)
	}
	// 增加 20% 的缓冲，避免 Gas 不足
	gasLimit = gasLimit * 120 / 100

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}

	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get header: %v", err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	// 检查 ETH 余额是否足够支付 Gas 费用
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}

	// 计算总费用：gasFeeCap * gasLimit（ERC-20 转账不需要发送 ETH，只需要支付 Gas）
	totalGasCost := new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit)))

	if balance.Cmp(totalGasCost) < 0 {
		log.Fatalf("insufficient ETH balance for gas: have %s wei, need %s wei", balance.String(), totalGasCost.String())
	}

	// 构造交易（EIP-1559 动态费用交易）
	// 注意：ERC-20 transfer 的 value 为 0，调用数据在 Data 字段中
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &contractAddr, // 合约地址
		Value:     big.NewInt(0), // ERC-20 转账不需要发送 ETH
		Data:      callData,      // transfer 调用数据
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	// 发送交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("ERC-20 Transfer Transaction Sent\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("From          : %s\n", fromAddr.Hex())
	fmt.Printf("Contract      : %s\n", contractAddr.Hex())
	// 显示代币数量（根据 decimals 转换）
	fmt.Printf("Gas Limit     : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap   : %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap   : %s Wei\n", gasFeeCap.String())
	fmt.Printf("Estimated Cost: %s Wei\n", totalGasCost.String())
	fmt.Printf("Nonce         : %d\n", nonce)
	fmt.Printf("Tx Hash       : %s\n", signedTx.Hash().Hex())
	fmt.Printf("\n")
	fmt.Printf("Transaction is pending. Waiting for confirmation...\n")
	fmt.Printf("\n")

	// 等待交易确认
	waitForTransaction(ctx, client, signedTx.Hash())
}


func inc(ctx context.Context, client *ethclient.Client, parsedABI abi.ABI, contractHex string) {
	if contractHex == ""  {
		log.Fatal("missing --contract, --to, or --amount flag for transfer mode")
	}

	// 检查私钥环境变量
	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for transfer mode)")
	}

	// 解析私钥
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	// 获取发送方地址
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

	contractAddr := common.HexToAddress(contractHex)

	

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	// 编码 transfer 调用数据
	// transfer(address to, uint256 value)
	callData, err := parsedABI.Pack("increment")
	if err != nil {
		log.Fatalf("failed to pack transfer data: %v", err)
	}

	// 估算 Gas Limit（合约调用需要更多 Gas）
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddr,
		To:   &contractAddr,
		Data: callData,
	})
	if err != nil {
		log.Fatalf("failed to estimate gas: %v", err)
	}
	// 增加 20% 的缓冲，避免 Gas 不足
	gasLimit = gasLimit * 120 / 100

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}

	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get header: %v", err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	// 检查 ETH 余额是否足够支付 Gas 费用
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}

	// 计算总费用：gasFeeCap * gasLimit（ERC-20 转账不需要发送 ETH，只需要支付 Gas）
	totalGasCost := new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit)))

	if balance.Cmp(totalGasCost) < 0 {
		log.Fatalf("insufficient ETH balance for gas: have %s wei, need %s wei", balance.String(), totalGasCost.String())
	}

	// 构造交易（EIP-1559 动态费用交易）
	// 注意：ERC-20 transfer 的 value 为 0，调用数据在 Data 字段中
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &contractAddr, // 合约地址
		Value:     big.NewInt(0), // ERC-20 转账不需要发送 ETH
		Data:      callData,      // transfer 调用数据
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	// 发送交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("ERC-20 Transfer Transaction Sent\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("From          : %s\n", fromAddr.Hex())
	fmt.Printf("Contract      : %s\n", contractAddr.Hex())
	// 显示代币数量（根据 decimals 转换）
	fmt.Printf("Gas Limit     : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap   : %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap   : %s Wei\n", gasFeeCap.String())
	fmt.Printf("Estimated Cost: %s Wei\n", totalGasCost.String())
	fmt.Printf("Nonce         : %d\n", nonce)
	fmt.Printf("Tx Hash       : %s\n", signedTx.Hash().Hex())
	fmt.Printf("\n")
	fmt.Printf("Transaction is pending. Waiting for confirmation...\n")
	fmt.Printf("\n")

	// 等待交易确认
	waitForTransaction(ctx, client, signedTx.Hash())
}

// waitForTransaction 等待交易确认并显示回执信息
func waitForTransaction(ctx context.Context, client *ethclient.Client, txHash common.Hash) {
	// 设置超时上下文（最多等待 2 分钟）
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Polling for transaction receipt...\n")
	for {
		select {
		case <-waitCtx.Done():
			fmt.Printf("\nTimeout waiting for transaction confirmation.\n")
			fmt.Printf("You can check the transaction status later:\n")
			fmt.Printf("  go run main.go --mode parse-event --tx %s\n", txHash.Hex())
			return

		case <-ticker.C:
			receipt, err := client.TransactionReceipt(waitCtx, txHash)
			if err != nil {
				// 交易可能还在 pending
				continue
			}

			// 交易已确认
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("Transaction Confirmed!\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("Status       : %d (1=success, 0=failed)\n", receipt.Status)
			fmt.Printf("Block Number : %d\n", receipt.BlockNumber.Uint64())
			fmt.Printf("Block Hash   : %s\n", receipt.BlockHash.Hex())
			fmt.Printf("Gas Used     : %d / %d\n", receipt.GasUsed, receipt.GasUsed)
			fmt.Printf("Logs Count   : %d\n", len(receipt.Logs))

			if receipt.Status == 0 {
				fmt.Printf("\n⚠️  Transaction failed! Check the transaction on block explorer.\n")
			} else {
				fmt.Printf("\n✅ Transaction successful!\n")
				if len(receipt.Logs) > 0 {
					fmt.Printf("\nTo parse Transfer event from this transaction:\n")
					fmt.Printf("  go run main.go --mode parse-event --tx %s\n", txHash.Hex())
				}
			}
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			return
		}
	}
}