use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::Item;
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct EscrowInfo {
    pub taker: Addr,
    pub maker: Addr,
    pub secret_hash: String,
    pub timelock: u64,
    pub src_chain_id: String,
    pub src_escrow_address: String,
    pub expected_amount: Uint128,
    pub deposited_amount: Uint128,
    pub deposited_denom: Option<String>,
    pub cw20_contract: Option<Addr>,
    pub status: EscrowStatus,
    pub created_at: u64,
    pub src_confirmed: bool,
    pub src_tx_hash: Option<String>,
    pub src_block_height: Option<u64>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum EscrowStatus {
    Active,
    Withdrawn,
    Cancelled,
}

pub const ESCROW_INFO: Item<EscrowInfo> = Item::new("escrow_info");

