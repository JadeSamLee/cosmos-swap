use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized {},

    #[error("Auction not found")]
    AuctionNotFound {},

    #[error("Auction already ended")]
    AuctionEnded {},

    #[error("Auction not started")]
    AuctionNotStarted {},

    #[error("Invalid bid amount")]
    InvalidBidAmount {},

    #[error("Auction still active")]
    AuctionStillActive {},

    #[error("Invalid auction parameters")]
    InvalidAuctionParameters {},

    #[error("Minimum price reached")]
    MinimumPriceReached {},
}

