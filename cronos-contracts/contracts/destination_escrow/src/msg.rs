use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Uint128};
use cw20::Cw20ReceiveMsg;

#[cw_serde]
pub struct InstantiateMsg {
    pub taker: String,
    pub maker: String,
    pub secret_hash: String,
    pub timelock: u64,
    pub src_chain_id: String,
    pub src_escrow_address: String,
    pub expected_amount: Uint128,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Deposit native tokens to the escrow
    Deposit {},
    /// Deposit CW20 tokens to the escrow
    Receive(Cw20ReceiveMsg),
    /// Withdraw tokens using the secret (for maker)
    Withdraw { secret: String },
    /// Cancel the escrow after timelock expires (for taker)
    Cancel {},
    /// Confirm source escrow (called by relayer)
    ConfirmSourceEscrow { 
        src_tx_hash: String,
        block_height: u64,
    },
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
}

#[cw_serde]
pub struct EscrowResponse {
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

#[cw_serde]
pub enum EscrowStatus {
    Active,
    Withdrawn,
    Cancelled,
}

