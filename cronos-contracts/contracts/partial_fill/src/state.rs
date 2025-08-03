use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::Map;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Order {
    pub maker: Addr,
    pub taker: Option<Addr>,
    pub total_amount: Uint128,
    pub filled_amount: Uint128,
    pub price: Uint128,
    pub is_active: bool,
}

impl Order {
    pub fn remaining_amount(&self) -> Uint128 {
        self.total_amount - self.filled_amount
    }

    pub fn is_fully_filled(&self) -> bool {
        self.filled_amount >= self.total_amount
    }

    pub fn fill_percentage(&self) -> u64 {
        if self.total_amount.is_zero() {
            return 0;
        }
        (self.filled_amount.u128() * 100 / self.total_amount.u128()) as u64
    }
}

pub const ORDERS: Map<String, Order> = Map::new("orders");
