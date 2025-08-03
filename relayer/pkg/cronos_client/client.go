package cronos_client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/manus-ai/cronos-eth-bridge/pkg/config"
	"go.uber.org/zap"
)

// Client represents a Cronos blockchain client
type Client struct {
	config     *config.ChainConfig
	clientCtx  client.Context
	txConfig   client.TxConfig
	logger     *zap.Logger
	chainID    string
	account    sdk.AccAddress
	accountNum uint64
	sequence   uint64
}

// EscrowOrder represents an escrow order from the blockchain
type EscrowOrder struct {
	ID              string    `json:"id"`
	Maker           string    `json:"maker"`
	Taker           string    `json:"taker,omitempty"`
	SecretHash      string    `json:"secret_hash"`
	Timelock        uint64    `json:"timelock"`
	DstChainID      string    `json:"dst_chain_id"`
	DstAsset        string    `json:"dst_asset"`
	DstAmount       string    `json:"dst_amount"`
	DepositedAmount string    `json:"deposited_amount"`
	DepositedDenom  string    `json:"deposited_denom,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       uint64    `json:"created_at"`
	// Dutch auction fields
	InitialPrice    string `json:"initial_price,omitempty"`
	PriceDecayRate  string `json:"price_decay_rate,omitempty"`
	MinimumPrice    string `json:"minimum_price,omitempty"`
	// Partial fill fields
	AllowPartialFill   bool   `json:"allow_partial_fill"`
	FilledAmount       string `json:"filled_amount"`
	RemainingAmount    string `json:"remaining_amount"`
	MinimumFillAmount  string `json:"minimum_fill_amount,omitempty"`
}

// ContractExecuteMsg represents a CosmWasm contract execute message
type ContractExecuteMsg struct {
	Contract string      `json:"contract"`
	Msg      interface{} `json:"msg"`
	Funds    []sdk.Coin  `json:"funds,omitempty"`
}

// NewClient creates a new Cronos client
func NewClient(cfg *config.ChainConfig, logger *zap.Logger) (*Client, error) {
	// Initialize codec
	encodingConfig := makeEncodingConfig()
	
	// Create client context
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(nil).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode("sync").
		WithHomeDir("").
		WithViper("").
		WithNodeURI(cfg.RPCEndpoint).
		WithChainID(cfg.ChainID)

	// Initialize account from private key or mnemonic
	var account sdk.AccAddress
	if cfg.PrivateKey != "" {
		// TODO: Implement private key loading
		logger.Info("Loading account from private key")
	} else if cfg.Mnemonic != "" {
		// Create keyring from mnemonic
		kb := keyring.NewInMemory(encodingConfig.Marshaler)
		
		// Derive key from mnemonic
		hdPath := hd.CreateHDPath(44, 60, 0, 0, 0) // Ethereum-compatible path for Cronos
		if cfg.HDPath != "" {
			// Parse custom HD path if provided
			// TODO: Implement HD path parsing
		}
		
		keyInfo, err := kb.NewAccount("relayer", cfg.Mnemonic, "", hdPath, hd.Secp256k1)
		if err != nil {
			return nil, fmt.Errorf("failed to create account from mnemonic: %w", err)
		}
		
		account = keyInfo.GetAddress()
		clientCtx = clientCtx.WithKeyring(kb).WithFromAddress(account).WithFromName("relayer")
	} else {
		return nil, fmt.Errorf("either private_key or mnemonic must be provided")
	}

	client := &Client{
		config:    cfg,
		clientCtx: clientCtx,
		txConfig:  encodingConfig.TxConfig,
		logger:    logger,
		chainID:   cfg.ChainID,
		account:   account,
	}

	// Initialize account number and sequence
	if err := client.updateAccountInfo(); err != nil {
		return nil, fmt.Errorf("failed to update account info: %w", err)
	}

	return client, nil
}

// GetLatestBlock returns the latest block height
func (c *Client) GetLatestBlock(ctx context.Context) (int64, error) {
	node, err := c.clientCtx.GetNode()
	if err != nil {
		return 0, fmt.Errorf("failed to get node: %w", err)
	}

	status, err := node.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get node status: %w", err)
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// QueryContract queries a CosmWasm contract
func (c *Client) QueryContract(ctx context.Context, contractAddr string, queryMsg interface{}) ([]byte, error) {
	queryBytes, err := json.Marshal(queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query message: %w", err)
	}

	// Use the Cosmos SDK query client to query the contract
	// This is a simplified implementation - in practice, you'd use the wasmd query client
	node, err := c.clientCtx.GetNode()
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Query the contract state
	queryPath := fmt.Sprintf("store/wasm/key")
	result, err := node.ABCIQuery(ctx, queryPath, queryBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to query contract: %w", err)
	}

	return result.Response.Value, nil
}

// ExecuteContract executes a CosmWasm contract
func (c *Client) ExecuteContract(ctx context.Context, contractAddr string, executeMsg interface{}, funds []sdk.Coin) (string, error) {
	msgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal execute message: %w", err)
	}

	// Create execute message
	msg := &wasmtypes.MsgExecuteContract{
		Sender:   c.account.String(),
		Contract: contractAddr,
		Msg:      msgBytes,
		Funds:    funds,
	}

	// Build and broadcast transaction
	txHash, err := c.broadcastTx(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	c.logger.Info("Contract executed successfully",
		zap.String("contract", contractAddr),
		zap.String("tx_hash", txHash))

	return txHash, nil
}

// GetEscrowOrders retrieves escrow orders from the factory contract
func (c *Client) GetEscrowOrders(ctx context.Context, factoryAddr string, startAfter string, limit uint32) ([]EscrowOrder, error) {
	queryMsg := map[string]interface{}{
		"escrow_list": map[string]interface{}{
			"start_after": startAfter,
			"limit":       limit,
		},
	}

	result, err := c.QueryContract(ctx, factoryAddr, queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to query escrow orders: %w", err)
	}

	var response struct {
		Escrows []struct {
			Address     string `json:"address"`
			EscrowType  string `json:"escrow_type"`
			Creator     string `json:"creator"`
			CreatedAt   uint64 `json:"created_at"`
			Salt        string `json:"salt"`
		} `json:"escrows"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal escrow list response: %w", err)
	}

	// Query each escrow for detailed information
	var orders []EscrowOrder
	for _, escrowInfo := range response.Escrows {
		if escrowInfo.EscrowType == "Source" {
			order, err := c.getEscrowDetails(ctx, escrowInfo.Address)
			if err != nil {
				c.logger.Warn("Failed to get escrow details",
					zap.String("address", escrowInfo.Address),
					zap.Error(err))
				continue
			}
			order.ID = escrowInfo.Salt
			orders = append(orders, *order)
		}
	}

	return orders, nil
}

// getEscrowDetails retrieves detailed information about a specific escrow
func (c *Client) getEscrowDetails(ctx context.Context, escrowAddr string) (*EscrowOrder, error) {
	queryMsg := map[string]interface{}{
		"escrow": map[string]interface{}{},
	}

	result, err := c.QueryContract(ctx, escrowAddr, queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to query escrow details: %w", err)
	}

	var order EscrowOrder
	if err := json.Unmarshal(result, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal escrow details: %w", err)
	}

	return &order, nil
}

// GetCurrentPrice retrieves the current price for a Dutch auction order
func (c *Client) GetCurrentPrice(ctx context.Context, escrowAddr string) (string, error) {
	queryMsg := map[string]interface{}{
		"current_price": map[string]interface{}{},
	}

	result, err := c.QueryContract(ctx, escrowAddr, queryMsg)
	if err != nil {
		return "", fmt.Errorf("failed to query current price: %w", err)
	}

	var response struct {
		CurrentPrice string `json:"current_price"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal price response: %w", err)
	}

	return response.CurrentPrice, nil
}

// CreateSourceEscrow creates a new source escrow through the factory
func (c *Client) CreateSourceEscrow(ctx context.Context, factoryAddr string, params CreateEscrowParams) (string, error) {
	executeMsg := map[string]interface{}{
		"create_source_escrow": map[string]interface{}{
			"maker":                 params.Maker,
			"taker":                 params.Taker,
			"secret_hash":           params.SecretHash,
			"timelock":              params.Timelock,
			"dst_chain_id":          params.DstChainID,
			"dst_asset":             params.DstAsset,
			"dst_amount":            params.DstAmount,
			"initial_price":         params.InitialPrice,
			"price_decay_rate":      params.PriceDecayRate,
			"minimum_price":         params.MinimumPrice,
			"allow_partial_fill":    params.AllowPartialFill,
			"minimum_fill_amount":   params.MinimumFillAmount,
			"label":                 params.Label,
		},
	}

	return c.ExecuteContract(ctx, factoryAddr, executeMsg, nil)
}

// CreateDestinationEscrow creates a new destination escrow through the factory
func (c *Client) CreateDestinationEscrow(ctx context.Context, factoryAddr string, params CreateDestEscrowParams) (string, error) {
	executeMsg := map[string]interface{}{
		"create_destination_escrow": map[string]interface{}{
			"taker":               params.Taker,
			"maker":               params.Maker,
			"secret_hash":         params.SecretHash,
			"timelock":            params.Timelock,
			"src_chain_id":        params.SrcChainID,
			"src_escrow_address":  params.SrcEscrowAddress,
			"expected_amount":     params.ExpectedAmount,
			"label":               params.Label,
		},
	}

	return c.ExecuteContract(ctx, factoryAddr, executeMsg, nil)
}

// WithdrawFromEscrow withdraws funds from an escrow using the secret
func (c *Client) WithdrawFromEscrow(ctx context.Context, escrowAddr string, secret string) (string, error) {
	executeMsg := map[string]interface{}{
		"withdraw": map[string]interface{}{
			"secret": secret,
		},
	}

	return c.ExecuteContract(ctx, escrowAddr, executeMsg, nil)
}

// PartialWithdrawFromEscrow performs a partial withdrawal from an escrow
func (c *Client) PartialWithdrawFromEscrow(ctx context.Context, escrowAddr string, secret string, amount string) (string, error) {
	executeMsg := map[string]interface{}{
		"partial_withdraw": map[string]interface{}{
			"secret": secret,
			"amount": amount,
		},
	}

	return c.ExecuteContract(ctx, escrowAddr, executeMsg, nil)
}

// CancelEscrow cancels an escrow after the timelock expires
func (c *Client) CancelEscrow(ctx context.Context, escrowAddr string) (string, error) {
	executeMsg := map[string]interface{}{
		"cancel": map[string]interface{}{},
	}

	return c.ExecuteContract(ctx, escrowAddr, executeMsg, nil)
}

// broadcastTx builds and broadcasts a transaction
func (c *Client) broadcastTx(ctx context.Context, msgs ...sdk.Msg) (string, error) {
	// Update sequence number
	if err := c.updateAccountInfo(); err != nil {
		return "", fmt.Errorf("failed to update account info: %w", err)
	}

	// Build transaction
	txBuilder := c.txConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return "", fmt.Errorf("failed to set messages: %w", err)
	}

	// Set gas and fees
	gasLimit, err := strconv.ParseUint(fmt.Sprintf("%d", c.config.GasLimit), 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse gas limit: %w", err)
	}
	txBuilder.SetGasLimit(gasLimit)

	// Parse gas price and set fees
	gasPrice, err := sdk.ParseDecCoin(c.config.GasPrice)
	if err != nil {
		return "", fmt.Errorf("failed to parse gas price: %w", err)
	}
	
	feeAmount := gasPrice.Amount.MulInt64(int64(gasLimit))
	fees := sdk.NewCoins(sdk.NewCoin(gasPrice.Denom, feeAmount.TruncateInt()))
	txBuilder.SetFeeAmount(fees)

	// Sign transaction
	sigV2 := signing.SignatureV2{
		PubKey: nil, // Will be set by the signing process
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: c.sequence,
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return "", fmt.Errorf("failed to set signatures: %w", err)
	}

	// Create signing data
	signerData := authsigning.SignerData{
		ChainID:       c.chainID,
		AccountNumber: c.accountNum,
		Sequence:      c.sequence,
	}

	// Sign the transaction
	sigV2, err = tx.SignWithPrivKey(
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		nil, // Use private key from keyring
		c.txConfig,
		c.sequence,
	)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return "", fmt.Errorf("failed to set final signatures: %w", err)
	}

	// Broadcast transaction
	txBytes, err := c.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	node, err := c.clientCtx.GetNode()
	if err != nil {
		return "", fmt.Errorf("failed to get node: %w", err)
	}

	result, err := node.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s", result.Code, result.Log)
	}

	// Increment sequence for next transaction
	c.sequence++

	return fmt.Sprintf("%X", result.Hash), nil
}

// updateAccountInfo updates the account number and sequence
func (c *Client) updateAccountInfo() error {
	accountRetriever := authtypes.AccountRetriever{}
	account, err := accountRetriever.GetAccount(c.clientCtx, c.account)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	c.accountNum = account.GetAccountNumber()
	c.sequence = account.GetSequence()

	return nil
}

// Helper types for method parameters
type CreateEscrowParams struct {
	Maker               string
	Taker               string
	SecretHash          string
	Timelock            uint64
	DstChainID          string
	DstAsset            string
	DstAmount           string
	InitialPrice        string
	PriceDecayRate      string
	MinimumPrice        string
	AllowPartialFill    bool
	MinimumFillAmount   string
	Label               string
}

type CreateDestEscrowParams struct {
	Taker             string
	Maker             string
	SecretHash        string
	Timelock          uint64
	SrcChainID        string
	SrcEscrowAddress  string
	ExpectedAmount    string
	Label             string
}

// makeEncodingConfig creates the encoding configuration
func makeEncodingConfig() EncodingConfig {
	// This is a simplified version - in practice, you'd import the actual Cronos app encoding config
	return EncodingConfig{
		InterfaceRegistry: nil, // TODO: Initialize with proper registry
		Marshaler:         nil, // TODO: Initialize with proper codec
		TxConfig:          nil, // TODO: Initialize with proper tx config
		Amino:             nil, // TODO: Initialize with proper amino codec
	}
}

type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Marshaler         codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

