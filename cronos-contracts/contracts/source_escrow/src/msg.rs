use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Coin, Uint128};
use cw20::Cw20ReceiveMsg;

#[cw_serde]
pub struct InstantiateMsg {
    pub maker: String,
    pub taker: Option<String>,
    pub secret_hash: String,
    pub timelock: u64,
    pub dst_chain_id: String,
    pub dst_asset: String,
    pub dst_amount: Uint128,
    // Dutch auction parameters
    pub initial_price: Option<Uint128>,
    pub price_decay_rate: Option<Uint128>, // per second
    pub minimum_price: Option<Uint128>,
    // Partial fill parameters
    pub allow_partial_fill: bool,
    pub minimum_fill_amount: Option<Uint128>,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Deposit native tokens to the escrow
    Deposit {},
    /// Deposit CW20 tokens to the escrow
    Receive(Cw20ReceiveMsg),
    /// Withdraw tokens using the secret
    Withdraw { secret: String },
    /// Cancel the escrow after timelock expires
    Cancel {},
    /// Partial withdraw for partial fills
    PartialWithdraw { 
        secret: String, 
        amount: Uint128 
    },
    /// Update the current price (Dutch auction)
    UpdatePrice {},
}

#[cw_serde]
pub enum ReceiveMsg {
    /// Deposit CW20 tokens
    Deposit {},
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Get escrow details
    #[returns(EscrowResponse)]
    Escrow {},
    /// Get current price (Dutch auction)
    #[returns(PriceResponse)]
    CurrentPrice {},
    /// Get fill status
    #[returns(FillStatusResponse)]
    FillStatus {},
}

#[cw_serde]
pub struct EscrowResponse {
    pub maker: Addr,
    pub taker: Option<Addr>,
    pub secret_hash: String,
    pub timelock: u64,
    pub dst_chain_id: String,
    pub dst_asset: String,
    pub dst_amount: Uint128,
    pub deposited_amount: Uint128,
    pub deposited_denom: Option<String>,
    pub cw20_contract: Option<Addr>,
    pub status: EscrowStatus,
    pub created_at: u64,
    pub allow_partial_fill: bool,
    pub filled_amount: Uint128,
    pub remaining_amount: Uint128,
}

#[cw_serde]
pub struct PriceResponse {
    pub current_price: Uint128,
    pub initial_price: Option<Uint128>,
    pub minimum_price: Option<Uint128>,
    pub price_decay_rate: Option<Uint128>,
    pub time_elapsed: u64,
}

#[cw_serde]
pub struct FillStatusResponse {
    pub total_amount: Uint128,
    pub filled_amount: Uint128,
    pub remaining_amount: Uint128,
    pub is_fully_filled: bool,
    pub allow_partial_fill: bool,
}

#[cw_serde]
pub enum EscrowStatus {
    Active,
    Withdrawn,
    Cancelled,
    PartiallyFilled,
}

