use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Binary, Uint128};

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: String,
    pub source_escrow_code_id: u64,
    pub destination_escrow_code_id: u64,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Create a new source escrow
    CreateSourceEscrow {
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
        label: String,
    },
    /// Create a new destination escrow
    CreateDestinationEscrow {
        taker: String,
        maker: String,
        secret_hash: String,
        timelock: u64,
        src_chain_id: String,
        src_escrow_address: String,
        expected_amount: Uint128,
        label: String,
    },
    /// Update code IDs (owner only)
    UpdateCodeIds {
        source_escrow_code_id: Option<u64>,
        destination_escrow_code_id: Option<u64>,
    },
    /// Update owner
    UpdateOwner { new_owner: String },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Get factory config
    #[returns(ConfigResponse)]
    Config {},
    /// Get escrow address by salt
    #[returns(EscrowAddressResponse)]
    EscrowAddress { salt: String },
    /// List all created escrows
    #[returns(EscrowListResponse)]
    EscrowList {
        start_after: Option<String>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct ConfigResponse {
    pub owner: Addr,
    pub source_escrow_code_id: u64,
    pub destination_escrow_code_id: u64,
}

#[cw_serde]
pub struct EscrowAddressResponse {
    pub address: String,
}

#[cw_serde]
pub struct EscrowListResponse {
    pub escrows: Vec<EscrowInfo>,
}

#[cw_serde]
pub struct EscrowInfo {
    pub address: Addr,
    pub escrow_type: EscrowType,
    pub creator: Addr,
    pub created_at: u64,
    pub salt: String,
}

#[cw_serde]
pub enum EscrowType {
    Source,
    Destination,
}

