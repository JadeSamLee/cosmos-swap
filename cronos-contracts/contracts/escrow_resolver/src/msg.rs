use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Uint128};

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: String,
    pub escrow_factory: String,
    pub authorized_relayers: Vec<String>,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Deploy a new source escrow and integrate with Dutch auction/LOP
    DeploySrc {
        maker: String,
        taker: Option<String>,
        secret_hash: String,
        timelock: u64,
        dst_chain_id: String,
        dst_asset: String,
        dst_amount: Uint128,
        // Dutch auction parameters
        initial_price: Option<Uint128>,
        price_decay_rate: Option<Uint128>,
        minimum_price: Option<Uint128>,
        // Partial fill parameters
        allow_partial_fill: bool,
        minimum_fill_amount: Option<Uint128>,
        // LOP integration
        lop_order_data: Option<String>,
        label: String,
    },
    /// Deploy a new destination escrow
    DeployDst {
        taker: String,
        maker: String,
        secret_hash: String,
        timelock: u64,
        src_chain_id: String,
        src_escrow_address: String,
        expected_amount: Uint128,
        label: String,
    },
    /// Withdraw from an escrow using the secret
    Withdraw {
        escrow_address: String,
        secret: String,
    },
    /// Partial withdraw from an escrow
    PartialWithdraw {
        escrow_address: String,
        secret: String,
        amount: Uint128,
    },
    /// Cancel an escrow
    Cancel {
        escrow_address: String,
    },
    /// Update Dutch auction price for an order
    UpdatePrice {
        escrow_address: String,
    },
    /// Process a cross-chain order (called by relayer)
    ProcessOrder {
        order_id: String,
        action: OrderAction,
        proof: Option<String>,
    },
    /// Add authorized relayer
    AddRelayer {
        relayer: String,
    },
    /// Remove authorized relayer
    RemoveRelayer {
        relayer: String,
    },
    /// Update owner
    UpdateOwner {
        new_owner: String,
    },
}

#[cw_serde]
pub enum OrderAction {
    /// Confirm source escrow on destination chain
    ConfirmSource {
        src_tx_hash: String,
        block_height: u64,
    },
    /// Execute swap
    ExecuteSwap {
        secret: String,
    },
    /// Cancel order
    CancelOrder,
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Get resolver configuration
    #[returns(ConfigResponse)]
    Config {},
    /// Get order information
    #[returns(OrderResponse)]
    Order { order_id: String },
    /// List active orders
    #[returns(OrderListResponse)]
    ActiveOrders {
        start_after: Option<String>,
        limit: Option<u32>,
    },
    /// Get Dutch auction current price
    #[returns(PriceResponse)]
    CurrentPrice { escrow_address: String },
    /// Check if relayer is authorized
    #[returns(RelayerResponse)]
    IsAuthorizedRelayer { relayer: String },
}

#[cw_serde]
pub struct ConfigResponse {
    pub owner: Addr,
    pub escrow_factory: Addr,
    pub authorized_relayers: Vec<Addr>,
}

#[cw_serde]
pub struct OrderResponse {
    pub order_id: String,
    pub escrow_address: Addr,
    pub maker: Addr,
    pub taker: Option<Addr>,
    pub status: OrderStatus,
    pub created_at: u64,
    pub updated_at: u64,
    pub dutch_auction: Option<DutchAuctionInfo>,
    pub partial_fill: Option<PartialFillInfo>,
}

#[cw_serde]
pub struct OrderListResponse {
    pub orders: Vec<OrderResponse>,
}

#[cw_serde]
pub struct PriceResponse {
    pub current_price: Uint128,
    pub initial_price: Option<Uint128>,
    pub minimum_price: Option<Uint128>,
    pub time_elapsed: u64,
}

#[cw_serde]
pub struct RelayerResponse {
    pub is_authorized: bool,
}

#[cw_serde]
pub struct DutchAuctionInfo {
    pub initial_price: Uint128,
    pub minimum_price: Uint128,
    pub price_decay_rate: Uint128,
    pub start_time: u64,
    pub current_price: Uint128,
}

#[cw_serde]
pub struct PartialFillInfo {
    pub allow_partial_fill: bool,
    pub minimum_fill_amount: Option<Uint128>,
    pub filled_amount: Uint128,
    pub remaining_amount: Uint128,
}

#[cw_serde]
pub enum OrderStatus {
    Active,
    Matched,
    Completed,
    Cancelled,
    Expired,
}

