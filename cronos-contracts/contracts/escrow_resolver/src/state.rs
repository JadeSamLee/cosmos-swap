use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::msg::{OrderStatus, DutchAuctionInfo, PartialFillInfo};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub owner: Addr,
    pub escrow_factory: Addr,
    pub authorized_relayers: Vec<Addr>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Order {
    pub order_id: String,
    pub escrow_address: Addr,
    pub maker: Addr,
    pub taker: Option<Addr>,
    pub status: OrderStatus,
    pub created_at: u64,
    pub updated_at: u64,
    pub dutch_auction: Option<DutchAuctionInfo>,
    pub partial_fill: Option<PartialFillInfo>,
    pub lop_order_data: Option<String>,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const ORDERS: Map<String, Order> = Map::new("orders");
pub const ORDER_COUNT: Item<u64> = Item::new("order_count");

