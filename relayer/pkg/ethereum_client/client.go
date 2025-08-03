package ethereum_client

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/manus-ai/cronos-eth-bridge/pkg/config"
	"go.uber.org/zap"
)

// Client represents an Ethereum blockchain client
type Client struct {
	config     *config.ChainConfig
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    *big.Int
	logger     *zap.Logger
	
	// Contract ABIs
	escrowFactoryABI abi.ABI
	resolverABI      abi.ABI
	escrowABI        abi.ABI
	ibcHandlerABI    abi.ABI
	lopABI           abi.ABI
}

// EscrowOrder represents an escrow order from Ethereum
type EscrowOrder struct {
	ID              string    `json:"id"`
	Maker           string    `json:"maker"`
	Taker           string    `json:"taker,omitempty"`
	SecretHash      string    `json:"secret_hash"`
	Timelock        uint64    `json:"timelock"`
	SrcChainID      string    `json:"src_chain_id"`
	SrcAsset        string    `json:"src_asset"`
	SrcAmount       *big.Int  `json:"src_amount"`
	DepositedAmount *big.Int  `json:"deposited_amount"`
	TokenAddress    string    `json:"token_address,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       uint64    `json:"created_at"`
	EscrowAddress   string    `json:"escrow_address"`
}

// ContractAddresses holds the addresses of deployed contracts
type ContractAddresses struct {
	EscrowFactory      common.Address
	Resolver           common.Address
	IBCHandler         common.Address
	LimitOrderProtocol common.Address
}

// NewClient creates a new Ethereum client
func NewClient(cfg *config.ChainConfig, contracts *config.EthereumContracts, logger *zap.Logger) (*Client, error) {
	// Connect to Ethereum node
	client, err := ethclient.Dial(cfg.RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	// Load private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Get address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Load contract ABIs
	escrowFactoryABI, err := abi.JSON(strings.NewReader(EscrowFactoryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse EscrowFactory ABI: %w", err)
	}

	resolverABI, err := abi.JSON(strings.NewReader(ResolverABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Resolver ABI: %w", err)
	}

	escrowABI, err := abi.JSON(strings.NewReader(EscrowABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse Escrow ABI: %w", err)
	}

	ibcHandlerABI, err := abi.JSON(strings.NewReader(IBCHandlerABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse IBCHandler ABI: %w", err)
	}

	lopABI, err := abi.JSON(strings.NewReader(LimitOrderProtocolABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse LOP ABI: %w", err)
	}

	ethClient := &Client{
		config:           cfg,
		client:           client,
		privateKey:       privateKey,
		address:          address,
		chainID:          chainID,
		logger:           logger,
		escrowFactoryABI: escrowFactoryABI,
		resolverABI:      resolverABI,
		escrowABI:        escrowABI,
		ibcHandlerABI:    ibcHandlerABI,
		lopABI:           lopABI,
	}

	logger.Info("Ethereum client initialized",
		zap.String("address", address.Hex()),
		zap.String("chain_id", chainID.String()))

	return ethClient, nil
}

// GetLatestBlock returns the latest block number
func (c *Client) GetLatestBlock(ctx context.Context) (uint64, error) {
	header, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	return header.Number.Uint64(), nil
}

// GetEscrowOrders retrieves escrow orders from the factory contract
func (c *Client) GetEscrowOrders(ctx context.Context, factoryAddr string, fromBlock uint64) ([]EscrowOrder, error) {
	contractAddr := common.HexToAddress(factoryAddr)
	
	// Query for EscrowCreated events
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   nil,
		Addresses: []common.Address{contractAddr},
		Topics:    [][]common.Hash{{crypto.Keccak256Hash([]byte("EscrowCreated(address,address,address,bytes32,uint256)"))}},
	}

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs: %w", err)
	}

	var orders []EscrowOrder
	for _, log := range logs {
		order, err := c.parseEscrowCreatedEvent(ctx, log)
		if err != nil {
			c.logger.Warn("Failed to parse escrow created event",
				zap.String("tx_hash", log.TxHash.Hex()),
				zap.Error(err))
			continue
		}
		orders = append(orders, *order)
	}

	return orders, nil
}

// parseEscrowCreatedEvent parses an EscrowCreated event log
func (c *Client) parseEscrowCreatedEvent(ctx context.Context, log types.Log) (*EscrowOrder, error) {
	// Parse the event data
	event := struct {
		Escrow     common.Address
		Maker      common.Address
		Taker      common.Address
		SecretHash [32]byte
		Timelock   *big.Int
	}{}

	err := c.escrowFactoryABI.UnpackIntoInterface(&event, "EscrowCreated", log.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack event data: %w", err)
	}

	// Get additional escrow details
	escrowDetails, err := c.getEscrowDetails(ctx, event.Escrow.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to get escrow details: %w", err)
	}

	order := &EscrowOrder{
		ID:              log.TxHash.Hex(),
		Maker:           event.Maker.Hex(),
		Taker:           event.Taker.Hex(),
		SecretHash:      fmt.Sprintf("0x%x", event.SecretHash),
		Timelock:        event.Timelock.Uint64(),
		EscrowAddress:   event.Escrow.Hex(),
		DepositedAmount: escrowDetails.DepositedAmount,
		TokenAddress:    escrowDetails.TokenAddress,
		Status:          escrowDetails.Status,
		CreatedAt:       escrowDetails.CreatedAt,
	}

	return order, nil
}

// getEscrowDetails retrieves detailed information about a specific escrow
func (c *Client) getEscrowDetails(ctx context.Context, escrowAddr string) (*EscrowOrder, error) {
	contractAddr := common.HexToAddress(escrowAddr)
	
	// Call the escrow contract to get details
	callOpts := &bind.CallOpts{Context: ctx}
	
	// Pack the call data for getting escrow info
	data, err := c.escrowABI.Pack("getEscrowInfo")
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	// Make the call
	result, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	var escrowInfo struct {
		DepositedAmount *big.Int
		TokenAddress    common.Address
		Status          uint8
		CreatedAt       *big.Int
	}

	err = c.escrowABI.UnpackIntoInterface(&escrowInfo, "getEscrowInfo", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	statusMap := map[uint8]string{
		0: "Active",
		1: "Withdrawn",
		2: "Cancelled",
	}

	return &EscrowOrder{
		DepositedAmount: escrowInfo.DepositedAmount,
		TokenAddress:    escrowInfo.TokenAddress.Hex(),
		Status:          statusMap[escrowInfo.Status],
		CreatedAt:       escrowInfo.CreatedAt.Uint64(),
	}, nil
}

// CreateDestinationEscrow creates a new destination escrow through the resolver
func (c *Client) CreateDestinationEscrow(ctx context.Context, resolverAddr string, params CreateDestEscrowParams) (string, error) {
	contractAddr := common.HexToAddress(resolverAddr)
	
	// Create transaction options
	auth, err := c.createTransactOpts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction options: %w", err)
	}

	// Pack the function call
	data, err := c.resolverABI.Pack("deployDst",
		params.DstImmutables,
		params.SrcCancellationTimestamp,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create transaction
	tx := types.NewTransaction(
		auth.Nonce.Uint64(),
		contractAddr,
		params.Value,
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("Destination escrow creation transaction sent",
		zap.String("tx_hash", signedTx.Hash().Hex()))

	return signedTx.Hash().Hex(), nil
}

// WithdrawFromEscrow withdraws funds from an escrow using the resolver
func (c *Client) WithdrawFromEscrow(ctx context.Context, resolverAddr string, escrowAddr string, secret string, immutables interface{}) (string, error) {
	contractAddr := common.HexToAddress(resolverAddr)
	
	// Create transaction options
	auth, err := c.createTransactOpts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction options: %w", err)
	}

	// Convert secret to bytes32
	secretBytes := crypto.Keccak256([]byte(secret))
	var secretHash [32]byte
	copy(secretHash[:], secretBytes)

	// Pack the function call
	data, err := c.resolverABI.Pack("withdraw",
		common.HexToAddress(escrowAddr),
		secretHash,
		immutables,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create and send transaction
	tx := types.NewTransaction(
		auth.Nonce.Uint64(),
		contractAddr,
		big.NewInt(0),
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("Withdraw transaction sent",
		zap.String("tx_hash", signedTx.Hash().Hex()),
		zap.String("escrow", escrowAddr))

	return signedTx.Hash().Hex(), nil
}

// CancelEscrow cancels an escrow through the resolver
func (c *Client) CancelEscrow(ctx context.Context, resolverAddr string, escrowAddr string, immutables interface{}) (string, error) {
	contractAddr := common.HexToAddress(resolverAddr)
	
	// Create transaction options
	auth, err := c.createTransactOpts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction options: %w", err)
	}

	// Pack the function call
	data, err := c.resolverABI.Pack("cancel",
		common.HexToAddress(escrowAddr),
		immutables,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create and send transaction
	tx := types.NewTransaction(
		auth.Nonce.Uint64(),
		contractAddr,
		big.NewInt(0),
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("Cancel transaction sent",
		zap.String("tx_hash", signedTx.Hash().Hex()),
		zap.String("escrow", escrowAddr))

	return signedTx.Hash().Hex(), nil
}

// FillLimitOrder fills a 1inch limit order
func (c *Client) FillLimitOrder(ctx context.Context, lopAddr string, order interface{}, signature []byte, amount *big.Int, takerTraits *big.Int, args []byte) (string, error) {
	contractAddr := common.HexToAddress(lopAddr)
	
	// Create transaction options
	auth, err := c.createTransactOpts(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction options: %w", err)
	}

	// Pack the function call
	data, err := c.lopABI.Pack("fillOrderArgs",
		order,
		signature,
		amount,
		takerTraits,
		args,
	)
	if err != nil {
		return "", fmt.Errorf("failed to pack function call: %w", err)
	}

	// Create and send transaction
	tx := types.NewTransaction(
		auth.Nonce.Uint64(),
		contractAddr,
		big.NewInt(0),
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(c.chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.Info("Limit order fill transaction sent",
		zap.String("tx_hash", signedTx.Hash().Hex()))

	return signedTx.Hash().Hex(), nil
}

// WaitForTransaction waits for a transaction to be mined
func (c *Client) WaitForTransaction(ctx context.Context, txHash string, timeout time.Duration) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash)
		case <-ticker.C:
			receipt, err := c.client.TransactionReceipt(ctx, hash)
			if err == nil {
				return receipt, nil
			}
			if err != ethereum.NotFound {
				return nil, fmt.Errorf("error getting transaction receipt: %w", err)
			}
		}
	}
}

// GetBalance returns the balance of the relayer account
func (c *Client) GetBalance(ctx context.Context) (*big.Int, error) {
	return c.client.BalanceAt(ctx, c.address, nil)
}

// GetTokenBalance returns the balance of a specific ERC20 token
func (c *Client) GetTokenBalance(ctx context.Context, tokenAddr string) (*big.Int, error) {
	// This would require the ERC20 ABI to make the balanceOf call
	// Simplified implementation
	return big.NewInt(0), nil
}

// createTransactOpts creates transaction options for sending transactions
func (c *Client) createTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	nonce, err := c.client.PendingNonceAt(ctx, c.address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = c.config.GasLimit
	auth.GasPrice = gasPrice
	auth.Context = ctx

	return auth, nil
}

// Helper types for method parameters
type CreateDestEscrowParams struct {
	DstImmutables             interface{}
	SrcCancellationTimestamp  *big.Int
	Value                     *big.Int
}

// Contract ABI constants (simplified versions)
const EscrowFactoryABI = `[
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "escrow", "type": "address"},
			{"indexed": true, "name": "maker", "type": "address"},
			{"indexed": true, "name": "taker", "type": "address"},
			{"indexed": false, "name": "secretHash", "type": "bytes32"},
			{"indexed": false, "name": "timelock", "type": "uint256"}
		],
		"name": "EscrowCreated",
		"type": "event"
	}
]`

const ResolverABI = `[
	{
		"inputs": [
			{"name": "dstImmutables", "type": "tuple"},
			{"name": "srcCancellationTimestamp", "type": "uint256"}
		],
		"name": "deployDst",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "escrow", "type": "address"},
			{"name": "secret", "type": "bytes32"},
			{"name": "immutables", "type": "tuple"}
		],
		"name": "withdraw",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"name": "escrow", "type": "address"},
			{"name": "immutables", "type": "tuple"}
		],
		"name": "cancel",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

const EscrowABI = `[
	{
		"inputs": [],
		"name": "getEscrowInfo",
		"outputs": [
			{"name": "depositedAmount", "type": "uint256"},
			{"name": "tokenAddress", "type": "address"},
			{"name": "status", "type": "uint8"},
			{"name": "createdAt", "type": "uint256"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

const IBCHandlerABI = `[
	{
		"inputs": [
			{"name": "packet", "type": "tuple"},
			{"name": "proof", "type": "bytes"},
			{"name": "proofHeight", "type": "tuple"}
		],
		"name": "recvPacket",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

const LimitOrderProtocolABI = `[
	{
		"inputs": [
			{"name": "order", "type": "tuple"},
			{"name": "signature", "type": "bytes"},
			{"name": "amount", "type": "uint256"},
			{"name": "takerTraits", "type": "uint256"},
			{"name": "args", "type": "bytes"}
		],
		"name": "fillOrderArgs",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

