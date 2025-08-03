use cosmwasm_std::{Addr, Uint128};
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::msg::{AuctionStatus, BidInfo};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Config {
    pub owner: Addr,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Auction {
    pub auction_id: String,
    pub seller: Addr,
    pub asset: String,
    pub amount: Uint128,
    pub initial_price: Uint128,
    pub minimum_price: Uint128,
    pub current_price: Uint128,
    pub price_decay_rate: Uint128,
    pub start_time: u64,
    pub end_time: u64,
    pub duration: u64,
    pub status: AuctionStatus,
    pub winner: Option<Addr>,
    pub winning_bid: Option<Uint128>,
    pub escrow_address: Option<Addr>,
}

pub const CONFIG: Item<Config> = Item::new("config");
pub const AUCTIONS: Map<String, Auction> = Map::new("auctions");
pub const AUCTION_BIDS: Map<(String, u64), BidInfo> = Map::new("auction_bids");
pub const AUCTION_BID_COUNT: Map<String, u64> = Map::new("auction_bid_count");

