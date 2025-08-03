# InterchainX: Cross-Chain Atomic Swap powered by 1inch

## Table of Contents

1.  [Introduction](#1-introduction)
2.  [Features](#2-features)
3.  [Architecture Overview](#3-architecture-overview)
4.  [Workflow](#4-workflow)
5.  [Getting Started](#5-getting-started)
    *   [Prerequisites](#prerequisites)
    *   [Environment Setup](#environment-setup)
    *   [Building Components](#building-components)
    *   [Deployment](#deployment)
6.  [Configuration](#6-configuration)
    *   [Environment Variables](#environment-variables)
    *   [Relayer Configuration](#relayer-configuration)
7.  [Testing](#7-testing)
8.  [License](#9-license)

## 1. Introduction

InterchainX is a robust and secure cross-chain atomic swap bridge designed to facilitate seamless and trustless asset exchanges between the Cronos blockchain (Cosmos SDK-based with CosmWasm support) and the Ethereum blockchain. Leveraging the principles of Hash Time-Locked Contracts (HTLCs) and enhanced with advanced trading features like Dutch auctions, partial fills, and Limit Order Protocol (LOP) integration, InterchainX aims to provide a highly efficient and flexible solution for interchain liquidity.

This project addresses the growing need for interoperability in the decentralized finance (DeFi) space, enabling users to swap assets between two distinct blockchain ecosystems without relying on centralized intermediaries. By utilizing the Inter-Blockchain Communication (IBC) protocol for secure asset transfers and an intelligent off-chain relayer, InterchainX ensures atomicity, security, and a superior user experience.

## 2. Features

*   **Bidirectional Cross-Chain Swaps**: Facilitates atomic swaps from Cronos to Ethereum and vice-versa.
*   **Hash Time-Locked Contracts (HTLCs)**: Ensures the atomicity of swaps, guaranteeing that either both sides of the swap complete or neither does.
*   **Dutch Auction Integration**: Allows Makers to initiate swaps with a dynamically decreasing price over time, encouraging faster fulfillment by Takers.
*   **Partial Fill Support**: Enables Takers to fulfill only a portion of a Maker's order, providing greater flexibility and liquidity.
*   **Limit Order Protocol (LOP)**: Supports the creation and matching of limit orders, allowing users to specify desired prices for their swaps.
*   **CosmWasm Smart Contracts**: Secure and efficient Rust-based smart contracts for the Cronos (non-EVM) side, leveraging the power of WebAssembly.
*   **Solidity Smart Contracts**: Robust and audited Solidity contracts for the Ethereum (EVM) side, compatible with the vast Ethereum ecosystem.
*   **Go Relayer Service**: A high-performance off-chain relayer written in Go, responsible for monitoring blockchain events, relaying swap information, managing secrets, and orchestrating Dutch auctions and partial fills.
*   **Inter-Blockchain Communication (IBC) Protocol**: Utilizes IBC for secure, trustless, and permissionless asset transfers between Cronos and Ethereum, ensuring true interoperability.
*   **Modular and Extensible Architecture**: Designed with clear separation of concerns, making it easy to extend, maintain, and integrate new features or blockchain networks.
*   **Comprehensive Testing**: Includes unit, integration, and end-to-end tests to ensure the reliability and security of the system.
*   **Dockerized Deployment**: Provides Docker configurations for easy and consistent deployment of the relayer service.

## 3. Architecture Overview

InterchainX operates on a two-sided escrow model, where assets are locked on both the source and destination chains until the swap conditions are met. An off-chain relayer acts as the orchestrator, facilitating communication and ensuring the atomic execution of swaps.

### Core Components:

*   **Cronos Chain (CosmWasm Side)**:
    *   **`SourceEscrow`**: Manages assets locked by the Maker on Cronos when it's the source chain.
    *   **`DestinationEscrow`**: Manages assets locked by the Taker on Cronos when it's the destination chain.
    *   **`EscrowFactory`**: Deploys new instances of escrow contracts deterministically.
    *   **`EscrowResolver`**: The primary interface for the relayer and users to interact with the CosmWasm escrow system, integrating Dutch auction and partial fill logic.
    *   **`DutchAuction`**: CosmWasm contract implementing the dynamic pricing logic for auctions.
    *   **`PartialFill`**: Logic integrated within escrow contracts to handle partial fulfillment of orders.
    *   **`IBCHandler` (Conceptual)**: A CosmWasm contract to manage IBC packet handling related to asset transfers.

*   **Ethereum Chain (EVM Side)**:
    *   **`Escrow.sol`**: Manages assets locked by the Maker/Taker on Ethereum.
    *   **`EscrowFactory.sol`**: Deploys new instances of `Escrow.sol`.
    *   **`Resolver.sol`**: The primary interface for the relayer and users to interact with the EVM escrow system, similar to the 1inch resolver.
    *   **`IBCHandler.sol`**: A Solidity contract acting as an interface to an IBC client on Ethereum, handling incoming and outgoing IBC transfers.

*   **Go Relayer Service (Off-chain)**:
    *   Monitors `SwapCreated` events on both chains.
    *   Relays order details and secrets between chains.
    *   Manages the Dutch auction pricing and partial fill states.
    *   Interacts with IBC relayers for cross-chain asset movements.
    *   Provides a RESTful API for status monitoring and management.

*   **IBC Relayer (External)**: A dedicated, off-chain IBC relayer (e.g., Hermes, `ibc-go`'s `relayer`) is required to relay IBC packets between the Cronos and Ethereum IBC clients. This component is external to this project's codebase but essential for the bridge's operation.

### Workflow Overview:

1.  **Order Creation**: Maker creates a swap order on the source chain (Cronos or Ethereum), locking assets in an escrow contract and providing a `secretHash`.
2.  **Order Discovery & Relay**: The Go Relayer detects the new order, calculates the dynamic price (if Dutch auction), and relays the order details to the destination chain.
3.  **Order Fulfillment**: Taker discovers the order, deposits equivalent assets on the destination chain, and reveals the `secret` (preimage) to claim the Maker's assets.
4.  **Secret Revelation & Withdrawal**: The Relayer observes the secret revelation, relays it to the Maker's chain, allowing the Maker to withdraw the Taker's assets. The Taker then uses the known secret to withdraw the Maker's original assets.
5.  **Recovery (Optional)**: If the swap is not completed within a `timelock`, either party can cancel and reclaim their locked assets.


## 4. Workflow

<img width="1131" height="779" alt="image" src="https://github.com/user-attachments/assets/bde7e7a8-785e-4463-822a-5a2aaf90947a" />


## 5. Getting Started

Follow these steps to set up, build, and deploy the InterchainX project.

### Prerequisites

Before you begin, ensure you have the following installed:

*   **Git**: For cloning the repository.
*   **Docker & Docker Compose**: For building and running containerized services.
*   **Rust & Cargo**: For CosmWasm contract development.
    *   Install `rustup` from [rustup.rs](https://rustup.rs/).
    *   Add the `wasm32-unknown-unknown` target: `rustup target add wasm32-unknown-unknown`.
*   **Go**: For the relayer service.
    *   Install Go from [go.dev/doc/install](https://go.dev/doc/install).
*   **Node.js & npm/yarn**: For TypeScript-based frontend integration and testing.
    *   Install Node.js from [nodejs.org](https://nodejs.org/).
*   **Cosmos SDK CLI tools**: For interacting with Cronos (e.g., `cronosd` or `wasmd`).
*   **Anvil/Ganache/Hardhat Network**: For local EVM development and testing.

### Environment Setup

Run the setup script to install necessary dependencies and tools:

```bash
./scripts/setup-environment.sh
```

This script will:
*   Install Go dependencies.
*   Install Node.js dependencies for TypeScript projects.
*   Ensure Rust toolchains are correctly configured.
*   (Potentially) Install `cronosd` or other Cosmos SDK CLIs.

### Building Components

Build all project components (CosmWasm contracts, EVM contracts, Go relayer):

```bash
./scripts/build-all.sh
```

This script will:
*   Build optimized CosmWasm `.wasm` binaries using Docker (`Dockerfile.cosmwasm`).
*   Compile Solidity contracts using Foundry.
*   Build the Go relayer executable.

### Deployment

Deployment involves deploying smart contracts to both Cronos and Ethereum, and then running the off-chain relayer.

**1. Deploy CosmWasm Contracts (Cronos)**:

```bash
./scripts/deploy-cosmwasm.sh
```

This script will:
*   Upload the optimized `.wasm` binaries to the Cronos chain.
*   Instantiate the `EscrowFactory`, `EscrowResolver`, `DutchAuction`, and other necessary CosmWasm contracts.
*   Record the deployed contract addresses and code IDs.

**2. Deploy EVM Contracts (Ethereum)**:

```bash
./scripts/deploy-evm.sh
```

This script will:
*   Deploy `Escrow.sol`, `EscrowFactory.sol`, `Resolver.sol`, and `IBCHandler.sol` to the Ethereum chain.
*   Record the deployed contract addresses.

**3. Configure and Run Relayer**:

After deploying contracts, you must configure the Go relayer with the deployed contract addresses and chain IDs. Refer to the [Configuration](#6-configuration) section.

```bash
# Build and run relayer using Docker Compose
docker-compose -f docker/docker-compose.yml up --build relayer

# Or run directly (after building)
./go-relayer/cmd/relayer/main
```

**4. Setup IBC Relayer (External)**:

If you are running a full cross-chain setup, you will need to set up and run an external IBC relayer (e.g., Hermes) to facilitate packet transfer between Cronos and Ethereum. Consult the documentation for your chosen IBC relayer.

## 6. Configuration

### Environment Variables

Key environment variables are used to configure blockchain RPC endpoints, private keys (for development/testing), and other sensitive information. These should ideally be managed through a `.env` file (which is excluded from Git) or your deployment environment's secrets management system.

Example `.env` file (for development/testing):

```dotenv
# EVM Chain Configuration
EVM_RPC_URL=http://localhost:8545
EVM_CHAIN_ID=1337 # Example for local development
EVM_PRIVATE_KEY=your_evm_private_key
EVM_CONTRACT_ADDRESS_ESCROW_FACTORY=
EVM_CONTRACT_ADDRESS_RESOLVER=
EVM_CONTRACT_ADDRESS_IBCHANDLER=

# CosmWasm Chain Configuration
COSMWASM_RPC_URL=http://localhost:26657
COSMWASM_CHAIN_ID=testnet-1 # Example for local development
COSMWASM_MNEMONIC="...your_cosmwasm_mnemonic_phrase..."
COSMWASM_CONTRACT_ADDRESS_ESCROW_FACTORY=
COSMWASM_CONTRACT_ADDRESS_ESCROW_RESOLVER=
COSMWASM_CONTRACT_ADDRESS_DUTCH_AUCTION=
COSMWASM_CONTRACT_ADDRESS_SOURCE_ESCROW_CODE_ID=1 # Code ID after uploading WASM
COSMWASM_CONTRACT_ADDRESS_DESTINATION_ESCROW_CODE_ID=2 # Code ID after uploading WASM

# Relayer Configuration
RELAYER_API_PORT=8080
RELAYER_LOG_LEVEL=info
RELAYER_DB_PATH=./data/relayer.db

# IBC Configuration (if managed by relayer or for external relayer config)
IBC_CHANNEL_ID_CRONOS_TO_ETH=channel-0
IBC_CHANNEL_ID_ETH_TO_CRONOS=channel-1
```

**Important**: Never commit sensitive information like private keys or mnemonic phrases directly to version control (e.g., GitHub). Use `.gitignore` to exclude `.env` files.

### Relayer Configuration (`config/relayer-config.template.yaml`)

The Go relayer service uses a YAML configuration file. A template is provided, which you will need to copy and populate with your deployed contract addresses and chain-specific details.

```yaml

evm:
  rpc_url: "${EVM_RPC_URL}"
  chain_id: ${EVM_CHAIN_ID}
  private_key: "${EVM_PRIVATE_KEY}"
  contract_addresses:
    escrow_factory: "${EVM_CONTRACT_ADDRESS_ESCROW_FACTORY}"
    resolver: "${EVM_CONTRACT_ADDRESS_RESOLVER}"
    ibc_handler: "${EVM_CONTRACT_ADDRESS_IBCHANDLER}"

cosmwasm:
  rpc_url: "${COSMWASM_RPC_URL}"
  chain_id: "${COSMWASM_CHAIN_ID}"
  mnemonic: "${COSMWASM_MNEMONIC}"
  contract_addresses:
    escrow_factory: "${COSMWASM_CONTRACT_ADDRESS_ESCROW_FACTORY}"
    escrow_resolver: "${COSMWASM_CONTRACT_ADDRESS_ESCROW_RESOLVER}"
    dutch_auction: "${COSMWASM_CONTRACT_ADDRESS_DUTCH_AUCTION}"
    source_escrow_code_id: ${COSMWASM_CONTRACT_ADDRESS_SOURCE_ESCROW_CODE_ID}
    destination_escrow_code_id: ${COSMWASM_CONTRACT_ADDRESS_DESTINATION_ESCROW_CODE_ID}

relayer:
  api_port: ${RELAYER_API_PORT}
  log_level: "${RELAYER_LOG_LEVEL}"
  db_path: "${RELAYER_DB_PATH}"

ibc:
  channel_id_cronos_to_eth: "${IBC_CHANNEL_ID_CRONOS_TO_ETH}"
  channel_id_eth_to_cronos: "${IBC_CHANNEL_ID_ETH_TO_CRONOS}"
```

## 7. Testing

The project includes a comprehensive test suite to ensure the correctness and security of all components. 

## 8. License

This project is licensed under the MIT License. See the `LICENSE` file for details.


