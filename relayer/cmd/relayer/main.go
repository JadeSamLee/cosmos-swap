package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/manus-ai/cronos-eth-bridge/pkg/config"
	"github.com/manus-ai/cronos-eth-bridge/pkg/cronos_client"
	"github.com/manus-ai/cronos-eth-bridge/pkg/ethereum_client"
	"github.com/manus-ai/cronos-eth-bridge/pkg/order_manager"
)

var (
	configPath string
	logger     *zap.Logger
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "relayer",
	Short: "Cronos-Ethereum Cross-Chain Bridge Relayer",
	Long: `A relayer service for facilitating cross-chain swaps between Cronos and Ethereum.
Supports Dutch auctions, partial fills, and 1inch Limit Order Protocol integration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initLogger()
	},
	RunE: runRelayer,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the relayer service",
	Long:  "Start the relayer service to monitor and facilitate cross-chain swaps",
	RunE:  runRelayer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Cronos-Ethereum Bridge Relayer v1.0.0")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to configuration file")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
}

func initLogger() error {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	return nil
}

func runRelayer(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger.Info("Starting Cronos-Ethereum Bridge Relayer",
		zap.String("cronos_chain_id", cfg.Cronos.ChainID),
		zap.String("ethereum_chain_id", cfg.Ethereum.ChainID))

	// Initialize blockchain clients
	cronosClient, err := cronos_client.NewClient(&cfg.Cronos, logger.Named("cronos"))
	if err != nil {
		return fmt.Errorf("failed to initialize Cronos client: %w", err)
	}

	ethereumClient, err := ethereum_client.NewClient(&cfg.Ethereum, &cfg.Contracts.Ethereum, logger.Named("ethereum"))
	if err != nil {
		return fmt.Errorf("failed to initialize Ethereum client: %w", err)
	}

	// Initialize order manager
	orderManager := order_manager.NewOrderManager(cfg, cronosClient, ethereumClient, logger.Named("order_manager"))

	// Start the relayer service
	relayerService := &RelayerService{
		config:         cfg,
		cronosClient:   cronosClient,
		ethereumClient: ethereumClient,
		orderManager:   orderManager,
		logger:         logger,
	}

	if err := relayerService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start relayer service: %w", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	// Graceful shutdown
	logger.Info("Shutting down relayer service...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := relayerService.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", zap.Error(err))
		return err
	}

	logger.Info("Relayer service stopped successfully")
	return nil
}

// RelayerService represents the main relayer service
type RelayerService struct {
	config         *config.Config
	cronosClient   *cronos_client.Client
	ethereumClient *ethereum_client.Client
	orderManager   *order_manager.OrderManager
	logger         *zap.Logger

	// Monitoring
	lastCronosBlock   int64
	lastEthereumBlock uint64

	// Stop channel
	stopChan chan struct{}
}

// Start starts the relayer service
func (rs *RelayerService) Start(ctx context.Context) error {
	rs.stopChan = make(chan struct{})

	// Start order manager
	if err := rs.orderManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start order manager: %w", err)
	}

	// Start monitoring goroutines
	go rs.monitorCronosOrders(ctx)
	go rs.monitorEthereumOrders(ctx)
	go rs.processOrderMatching(ctx)
	go rs.healthCheck(ctx)

	rs.logger.Info("Relayer service started successfully")
	return nil
}

// Stop stops the relayer service
func (rs *RelayerService) Stop(ctx context.Context) error {
	close(rs.stopChan)

	// Stop order manager
	if err := rs.orderManager.Stop(); err != nil {
		rs.logger.Error("Failed to stop order manager", zap.Error(err))
	}

	rs.logger.Info("Relayer service stopped")
	return nil
}

// monitorCronosOrders monitors for new orders on Cronos
func (rs *RelayerService) monitorCronosOrders(ctx context.Context) {
	ticker := time.NewTicker(rs.config.Relayer.BlockPollInterval)
	defer ticker.Stop()

	rs.logger.Info("Starting Cronos order monitoring")

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			if err := rs.scanCronosOrders(ctx); err != nil {
				rs.logger.Error("Failed to scan Cronos orders", zap.Error(err))
			}
		}
	}
}

// monitorEthereumOrders monitors for new orders on Ethereum
func (rs *RelayerService) monitorEthereumOrders(ctx context.Context) {
	ticker := time.NewTicker(rs.config.Relayer.BlockPollInterval)
	defer ticker.Stop()

	rs.logger.Info("Starting Ethereum order monitoring")

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			if err := rs.scanEthereumOrders(ctx); err != nil {
				rs.logger.Error("Failed to scan Ethereum orders", zap.Error(err))
			}
		}
	}
}

// scanCronosOrders scans for new orders on Cronos
func (rs *RelayerService) scanCronosOrders(ctx context.Context) error {
	// Get latest block
	latestBlock, err := rs.cronosClient.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest Cronos block: %w", err)
	}

	if latestBlock <= rs.lastCronosBlock {
		return nil // No new blocks
	}

	// Get new orders from the factory
	orders, err := rs.cronosClient.GetEscrowOrders(
		ctx,
		rs.config.Contracts.Cronos.EscrowFactory,
		"", // start_after
		50, // limit
	)
	if err != nil {
		return fmt.Errorf("failed to get Cronos orders: %w", err)
	}

	// Process new orders
	for _, cronosOrder := range orders {
		order := rs.convertCronosOrderToOrder(&cronosOrder)
		rs.orderManager.AddOrder(order)
	}

	rs.lastCronosBlock = latestBlock
	rs.logger.Debug("Scanned Cronos orders",
		zap.Int64("latest_block", latestBlock),
		zap.Int("new_orders", len(orders)))

	return nil
}

// scanEthereumOrders scans for new orders on Ethereum
func (rs *RelayerService) scanEthereumOrders(ctx context.Context) error {
	// Get latest block
	latestBlock, err := rs.ethereumClient.GetLatestBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest Ethereum block: %w", err)
	}

	if latestBlock <= rs.lastEthereumBlock {
		return nil // No new blocks
	}

	// Get new orders from the factory
	orders, err := rs.ethereumClient.GetEscrowOrders(
		ctx,
		rs.config.Contracts.Ethereum.EscrowFactory,
		rs.lastEthereumBlock,
	)
	if err != nil {
		return fmt.Errorf("failed to get Ethereum orders: %w", err)
	}

	// Process new orders
	for _, ethOrder := range orders {
		order := rs.convertEthereumOrderToOrder(&ethOrder)
		rs.orderManager.AddOrder(order)
	}

	rs.lastEthereumBlock = latestBlock
	rs.logger.Debug("Scanned Ethereum orders",
		zap.Uint64("latest_block", latestBlock),
		zap.Int("new_orders", len(orders)))

	return nil
}

// processOrderMatching processes order matching logic
func (rs *RelayerService) processOrderMatching(ctx context.Context) {
	ticker := time.NewTicker(rs.config.Relayer.OrderUpdateInterval)
	defer ticker.Stop()

	rs.logger.Info("Starting order matching processor")

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.matchOrders(ctx)
		}
	}
}

// matchOrders attempts to match orders
func (rs *RelayerService) matchOrders(ctx context.Context) {
	activeOrders := rs.orderManager.GetActiveOrders()
	
	// Simple matching logic - in practice, this would be more sophisticated
	for _, order := range activeOrders {
		if order.Status == order_manager.OrderStatusActive {
			// Check if order conditions are met for execution
			if rs.canExecuteOrder(order) {
				order.Status = order_manager.OrderStatusMatched
				rs.logger.Info("Order matched for execution", zap.String("order_id", order.ID))
			}
		}
	}
}

// canExecuteOrder checks if an order can be executed
func (rs *RelayerService) canExecuteOrder(order *order_manager.Order) bool {
	// Simplified logic - check if both escrows are funded
	// In practice, you would query both chains to verify the state
	return order.SourceEscrowAddr != "" && order.DestEscrowAddr != ""
}

// healthCheck performs periodic health checks
func (rs *RelayerService) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs a health check
func (rs *RelayerService) performHealthCheck(ctx context.Context) {
	// Check Cronos connection
	_, err := rs.cronosClient.GetLatestBlock(ctx)
	if err != nil {
		rs.logger.Error("Cronos health check failed", zap.Error(err))
	}

	// Check Ethereum connection
	_, err = rs.ethereumClient.GetLatestBlock(ctx)
	if err != nil {
		rs.logger.Error("Ethereum health check failed", zap.Error(err))
	}

	// Log order statistics
	stats := rs.orderManager.GetOrderStats()
	rs.logger.Info("Order manager statistics", zap.Any("stats", stats))
}

// convertCronosOrderToOrder converts a Cronos order to the internal Order format
func (rs *RelayerService) convertCronosOrderToOrder(cronosOrder *cronos_client.EscrowOrder) *order_manager.Order {
	order := &order_manager.Order{
		ID:               cronosOrder.ID,
		Type:             order_manager.OrderTypeCronosToEthereum,
		Status:           order_manager.OrderStatus(cronosOrder.Status),
		SourceChain:      "cronos",
		DestinationChain: cronosOrder.DstChainID,
		Maker:            cronosOrder.Maker,
		Taker:            cronosOrder.Taker,
		SecretHash:       cronosOrder.SecretHash,
		Timelock:         cronosOrder.Timelock,
		CreatedAt:        time.Unix(int64(cronosOrder.CreatedAt), 0),
		UpdatedAt:        time.Now(),
		ExpiresAt:        time.Unix(int64(cronosOrder.Timelock), 0),
	}

	// Set source asset info
	if amount, ok := new(big.Int).SetString(cronosOrder.DepositedAmount, 10); ok {
		order.SourceAsset = order_manager.AssetInfo{
			Symbol:  cronosOrder.DepositedDenom,
			Amount:  amount,
			Decimals: 18, // Default to 18 decimals
		}
	}

	// Set destination asset info
	if amount, ok := new(big.Int).SetString(cronosOrder.DstAmount, 10); ok {
		order.DestinationAsset = order_manager.AssetInfo{
			Symbol:  cronosOrder.DstAsset,
			Amount:  amount,
			Decimals: 18, // Default to 18 decimals
		}
	}

	// Set Dutch auction parameters if present
	if cronosOrder.InitialPrice != "" {
		if initialPrice, ok := new(big.Int).SetString(cronosOrder.InitialPrice, 10); ok {
			order.DutchAuction = &order_manager.DutchAuctionParams{
				InitialPrice: initialPrice,
				StartTime:    time.Unix(int64(cronosOrder.CreatedAt), 0),
				Duration:     rs.config.DutchAuction.MaxAuctionDuration,
			}

			if minPrice, ok := new(big.Int).SetString(cronosOrder.MinimumPrice, 10); ok {
				order.DutchAuction.MinimumPrice = minPrice
			}

			if decayRate, ok := new(big.Int).SetString(cronosOrder.PriceDecayRate, 10); ok {
				order.DutchAuction.DecayRate = decayRate
			}

			order.CurrentPrice = initialPrice
		}
	}

	// Set partial fill parameters if present
	if cronosOrder.AllowPartialFill {
		order.PartialFill = &order_manager.PartialFillParams{
			AllowPartialFill: true,
		}

		if filledAmount, ok := new(big.Int).SetString(cronosOrder.FilledAmount, 10); ok {
			order.PartialFill.FilledAmount = filledAmount
		}

		if remainingAmount, ok := new(big.Int).SetString(cronosOrder.RemainingAmount, 10); ok {
			order.PartialFill.RemainingAmount = remainingAmount
		}

		if minFillAmount, ok := new(big.Int).SetString(cronosOrder.MinimumFillAmount, 10); ok {
			order.PartialFill.MinimumFillAmount = minFillAmount
		}
	}

	return order
}

// convertEthereumOrderToOrder converts an Ethereum order to the internal Order format
func (rs *RelayerService) convertEthereumOrderToOrder(ethOrder *ethereum_client.EscrowOrder) *order_manager.Order {
	order := &order_manager.Order{
		ID:               ethOrder.ID,
		Type:             order_manager.OrderTypeEthereumToCronos,
		Status:           order_manager.OrderStatus(ethOrder.Status),
		SourceChain:      "ethereum",
		DestinationChain: ethOrder.SrcChainID,
		Maker:            ethOrder.Maker,
		Taker:            ethOrder.Taker,
		SecretHash:       ethOrder.SecretHash,
		Timelock:         ethOrder.Timelock,
		SourceEscrowAddr: ethOrder.EscrowAddress,
		CreatedAt:        time.Unix(int64(ethOrder.CreatedAt), 0),
		UpdatedAt:        time.Now(),
		ExpiresAt:        time.Unix(int64(ethOrder.Timelock), 0),
	}

	// Set source asset info
	order.SourceAsset = order_manager.AssetInfo{
		Symbol:   "ETH", // Default to ETH, could be ERC20 token
		Address:  ethOrder.TokenAddress,
		Amount:   ethOrder.DepositedAmount,
		Decimals: 18,
	}

	// Set destination asset info
	order.DestinationAsset = order_manager.AssetInfo{
		Symbol:   ethOrder.SrcAsset,
		Amount:   ethOrder.SrcAmount,
		Decimals: 18,
	}

	return order
}

