package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the relayer
type Config struct {
	// Chain configurations
	Cronos   ChainConfig `mapstructure:"cronos"`
	Ethereum ChainConfig `mapstructure:"ethereum"`

	// Contract addresses
	Contracts ContractConfig `mapstructure:"contracts"`

	// Relayer configuration
	Relayer RelayerConfig `mapstructure:"relayer"`

	// IBC configuration
	IBC IBCConfig `mapstructure:"ibc"`

	// Dutch auction configuration
	DutchAuction DutchAuctionConfig `mapstructure:"dutch_auction"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging"`
}

// ChainConfig holds configuration for a blockchain
type ChainConfig struct {
	ChainID     string `mapstructure:"chain_id"`
	RPCEndpoint string `mapstructure:"rpc_endpoint"`
	WSEndpoint  string `mapstructure:"ws_endpoint"`
	GasPrice    string `mapstructure:"gas_price"`
	GasLimit    uint64 `mapstructure:"gas_limit"`
	// Private key for the relayer account
	PrivateKey string `mapstructure:"private_key"`
	// Mnemonic as alternative to private key
	Mnemonic string `mapstructure:"mnemonic"`
	// HD derivation path
	HDPath string `mapstructure:"hd_path"`
}

// ContractConfig holds contract addresses for both chains
type ContractConfig struct {
	Cronos   CronosContracts   `mapstructure:"cronos"`
	Ethereum EthereumContracts `mapstructure:"ethereum"`
}

// CronosContracts holds CosmWasm contract addresses on Cronos
type CronosContracts struct {
	EscrowFactory    string `mapstructure:"escrow_factory"`
	EscrowResolver   string `mapstructure:"escrow_resolver"`
	DutchAuction     string `mapstructure:"dutch_auction"`
	PartialFill      string `mapstructure:"partial_fill"`
	IBCBridgeAdapter string `mapstructure:"ibc_bridge_adapter"`
	// Code IDs for contract instantiation
	SourceEscrowCodeID      uint64 `mapstructure:"source_escrow_code_id"`
	DestinationEscrowCodeID uint64 `mapstructure:"destination_escrow_code_id"`
}

// EthereumContracts holds Solidity contract addresses on Ethereum
type EthereumContracts struct {
	EscrowFactory string `mapstructure:"escrow_factory"`
	Resolver      string `mapstructure:"resolver"`
	IBCHandler    string `mapstructure:"ibc_handler"`
	// 1inch Limit Order Protocol contract
	LimitOrderProtocol string `mapstructure:"limit_order_protocol"`
}

// RelayerConfig holds relayer-specific configuration
type RelayerConfig struct {
	// Polling intervals
	BlockPollInterval    time.Duration `mapstructure:"block_poll_interval"`
	EventPollInterval    time.Duration `mapstructure:"event_poll_interval"`
	OrderUpdateInterval  time.Duration `mapstructure:"order_update_interval"`
	
	// Retry configuration
	MaxRetries    int           `mapstructure:"max_retries"`
	RetryInterval time.Duration `mapstructure:"retry_interval"`
	
	// Timeouts
	TransactionTimeout time.Duration `mapstructure:"transaction_timeout"`
	
	// Batch processing
	BatchSize int `mapstructure:"batch_size"`
	
	// Fee configuration
	RelayerFeePercentage float64 `mapstructure:"relayer_fee_percentage"`
}

// IBCConfig holds IBC-related configuration
type IBCConfig struct {
	// Channel information
	CronosToEthChannel string `mapstructure:"cronos_to_eth_channel"`
	EthToCronosChannel string `mapstructure:"eth_to_cronos_channel"`
	
	// Port information
	TransferPort string `mapstructure:"transfer_port"`
	
	// Timeout configuration
	PacketTimeout time.Duration `mapstructure:"packet_timeout"`
	
	// IBC relayer endpoint (e.g., Hermes)
	RelayerEndpoint string `mapstructure:"relayer_endpoint"`
}

// DutchAuctionConfig holds Dutch auction parameters
type DutchAuctionConfig struct {
	// Default parameters for new orders
	DefaultDecayRate    string        `mapstructure:"default_decay_rate"`
	DefaultMinimumPrice string        `mapstructure:"default_minimum_price"`
	MaxAuctionDuration  time.Duration `mapstructure:"max_auction_duration"`
	
	// Price update frequency
	PriceUpdateInterval time.Duration `mapstructure:"price_update_interval"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Set default values
	setDefaults()

	// Set config file path
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME/.cronos-eth-bridge")
	}

	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("BRIDGE")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Cronos defaults
	viper.SetDefault("cronos.chain_id", "cronos_777-1")
	viper.SetDefault("cronos.gas_price", "5000000000000basecro")
	viper.SetDefault("cronos.gas_limit", 300000)
	viper.SetDefault("cronos.hd_path", "m/44'/60'/0'/0/0")

	// Ethereum defaults
	viper.SetDefault("ethereum.chain_id", "1")
	viper.SetDefault("ethereum.gas_price", "20000000000")
	viper.SetDefault("ethereum.gas_limit", 300000)

	// Relayer defaults
	viper.SetDefault("relayer.block_poll_interval", "5s")
	viper.SetDefault("relayer.event_poll_interval", "10s")
	viper.SetDefault("relayer.order_update_interval", "30s")
	viper.SetDefault("relayer.max_retries", 3)
	viper.SetDefault("relayer.retry_interval", "10s")
	viper.SetDefault("relayer.transaction_timeout", "60s")
	viper.SetDefault("relayer.batch_size", 10)
	viper.SetDefault("relayer.relayer_fee_percentage", 0.1)

	// IBC defaults
	viper.SetDefault("ibc.transfer_port", "transfer")
	viper.SetDefault("ibc.packet_timeout", "600s")

	// Dutch auction defaults
	viper.SetDefault("dutch_auction.default_decay_rate", "1000000000000000000")
	viper.SetDefault("dutch_auction.default_minimum_price", "1000000000000000000")
	viper.SetDefault("dutch_auction.max_auction_duration", "24h")
	viper.SetDefault("dutch_auction.price_update_interval", "60s")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output_path", "stdout")
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	// Validate chain configurations
	if config.Cronos.ChainID == "" {
		return fmt.Errorf("cronos.chain_id is required")
	}
	if config.Cronos.RPCEndpoint == "" {
		return fmt.Errorf("cronos.rpc_endpoint is required")
	}
	if config.Ethereum.ChainID == "" {
		return fmt.Errorf("ethereum.chain_id is required")
	}
	if config.Ethereum.RPCEndpoint == "" {
		return fmt.Errorf("ethereum.rpc_endpoint is required")
	}

	// Validate private keys or mnemonics
	if config.Cronos.PrivateKey == "" && config.Cronos.Mnemonic == "" {
		return fmt.Errorf("cronos private_key or mnemonic is required")
	}
	if config.Ethereum.PrivateKey == "" && config.Ethereum.Mnemonic == "" {
		return fmt.Errorf("ethereum private_key or mnemonic is required")
	}

	// Validate contract addresses
	if config.Contracts.Cronos.EscrowFactory == "" {
		return fmt.Errorf("contracts.cronos.escrow_factory is required")
	}
	if config.Contracts.Ethereum.EscrowFactory == "" {
		return fmt.Errorf("contracts.ethereum.escrow_factory is required")
	}

	return nil
}

// GetConfigFromEnv loads configuration from environment variables only
func GetConfigFromEnv() (*Config, error) {
	config := &Config{
		Cronos: ChainConfig{
			ChainID:     getEnvOrDefault("BRIDGE_CRONOS_CHAIN_ID", "cronos_777-1"),
			RPCEndpoint: getEnvOrDefault("BRIDGE_CRONOS_RPC_ENDPOINT", ""),
			WSEndpoint:  getEnvOrDefault("BRIDGE_CRONOS_WS_ENDPOINT", ""),
			GasPrice:    getEnvOrDefault("BRIDGE_CRONOS_GAS_PRICE", "5000000000000basecro"),
			GasLimit:    300000,
			PrivateKey:  getEnvOrDefault("BRIDGE_CRONOS_PRIVATE_KEY", ""),
			Mnemonic:    getEnvOrDefault("BRIDGE_CRONOS_MNEMONIC", ""),
			HDPath:      getEnvOrDefault("BRIDGE_CRONOS_HD_PATH", "m/44'/60'/0'/0/0"),
		},
		Ethereum: ChainConfig{
			ChainID:     getEnvOrDefault("BRIDGE_ETHEREUM_CHAIN_ID", "1"),
			RPCEndpoint: getEnvOrDefault("BRIDGE_ETHEREUM_RPC_ENDPOINT", ""),
			WSEndpoint:  getEnvOrDefault("BRIDGE_ETHEREUM_WS_ENDPOINT", ""),
			GasPrice:    getEnvOrDefault("BRIDGE_ETHEREUM_GAS_PRICE", "20000000000"),
			GasLimit:    300000,
			PrivateKey:  getEnvOrDefault("BRIDGE_ETHEREUM_PRIVATE_KEY", ""),
			Mnemonic:    getEnvOrDefault("BRIDGE_ETHEREUM_MNEMONIC", ""),
		},
		Contracts: ContractConfig{
			Cronos: CronosContracts{
				EscrowFactory:  getEnvOrDefault("BRIDGE_CRONOS_ESCROW_FACTORY", ""),
				EscrowResolver: getEnvOrDefault("BRIDGE_CRONOS_ESCROW_RESOLVER", ""),
			},
			Ethereum: EthereumContracts{
				EscrowFactory:      getEnvOrDefault("BRIDGE_ETHEREUM_ESCROW_FACTORY", ""),
				Resolver:           getEnvOrDefault("BRIDGE_ETHEREUM_RESOLVER", ""),
				LimitOrderProtocol: getEnvOrDefault("BRIDGE_ETHEREUM_LOP", ""),
			},
		},
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

