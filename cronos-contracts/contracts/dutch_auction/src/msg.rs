use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Addr, Uint128};

#[cw_serde]
pub struct InstantiateMsg {
    pub owner: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Create a new Dutch auction
    CreateAuction {
        auction_id: String,
        seller: String,
        asset: String,
        amount: Uint128,
        initial_price: Uint128,
        minimum_price: Uint128,
        price_decay_rate: Uint128,
        duration: u64,
        escrow_address: Option<String>,
    },
    /// Place a bid on an auction
    PlaceBid {
        auction_id: String,
        bidder: String,
        bid_amount: Uint128,
    },
    /// Update auction price (called periodically)
    UpdatePrice {
        auction_id: String,
    },
    /// End an auction
    EndAuction {
        auction_id: String,
    },
    /// Cancel an auction (only by seller)
    CancelAuction {
        auction_id: String,
    },
    /// Update owner
    UpdateOwner {
        new_owner: String,
    },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    /// Get auction information
    #[returns(AuctionResponse)]
    Auction { auction_id: String },
    /// List active auctions
    #[returns(AuctionListResponse)]
    ActiveAuctions {
        start_after: Option<String>,
        limit: Option<u32>,
    },
    /// Get current price for an auction
    #[returns(PriceResponse)]
    CurrentPrice { auction_id: String },
    /// Get auction history
    #[returns(AuctionHistoryResponse)]
    AuctionHistory {
        auction_id: String,
        start_after: Option<String>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct AuctionResponse {
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

#[cw_serde]
pub struct AuctionListResponse {
    pub auctions: Vec<AuctionResponse>,
}

#[cw_serde]
pub struct PriceResponse {
    pub current_price: Uint128,
    pub time_remaining: u64,
    pub price_at_end: Uint128,
}

#[cw_serde]
pub struct AuctionHistoryResponse {
    pub bids: Vec<BidInfo>,
}

#[cw_serde]
pub struct BidInfo {
    pub bidder: Addr,
    pub amount: Uint128,
    pub timestamp: u64,
    pub price_at_bid: Uint128,
}

#[cw_serde]
pub enum AuctionStatus {
    Active,
    Ended,
    Cancelled,
}

