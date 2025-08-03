use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::Item;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct EscrowInfo {
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
    // Dutch auction fields
    pub initial_price: Option<Uint128>,
    pub price_decay_rate: Option<Uint128>, // per second
    pub minimum_price: Option<Uint128>,
    // Partial fill fields
    pub allow_partial_fill: bool,
    pub minimum_fill_amount: Option<Uint128>,
    pub filled_amount: Uint128,
    pub remaining_amount: Uint128,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum EscrowStatus {
    Active,
    Withdrawn,
    Cancelled,
    PartiallyFilled,
}

pub const ESCROW_INFO: Item<EscrowInfo> = Item::new("escrow_info");

