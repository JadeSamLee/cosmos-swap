use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized {},

    #[error("Invalid secret")]
    InvalidSecret {},

    #[error("Escrow already withdrawn")]
    AlreadyWithdrawn {},

    #[error("Escrow already cancelled")]
    AlreadyCancelled {},

    #[error("Cannot cancel before timelock expires")]
    TimelockNotExpired {},

    #[error("Insufficient funds")]
    InsufficientFunds {},

    #[error("Invalid partial fill amount")]
    InvalidPartialFillAmount {},

    #[error("Order fully filled")]
    OrderFullyFilled {},

    #[error("Dutch auction minimum price reached")]
    MinimumPriceReached {},

    #[error("Invalid dutch auction parameters")]
    InvalidDutchAuctionParams {},
}

