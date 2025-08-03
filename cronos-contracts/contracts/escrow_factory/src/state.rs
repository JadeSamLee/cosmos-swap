use cosmwasm_std::Addr;
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::msg::{EscrowInfo, EscrowType};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub owner: Addr,
    pub source_escrow_code_id: u64,
    pub destination_escrow_code_id: u64,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const ESCROWS: Map<String, EscrowInfo> = Map::new("escrows");

