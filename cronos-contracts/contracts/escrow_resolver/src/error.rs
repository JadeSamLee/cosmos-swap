use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized {},

    #[error("Invalid escrow address")]
    InvalidEscrowAddress {},

    #[error("Escrow operation failed")]
    EscrowOperationFailed {},

    #[error("Invalid order parameters")]
    InvalidOrderParameters {},

    #[error("Dutch auction not active")]
    DutchAuctionNotActive {},

    #[error("Partial fill not allowed")]
    PartialFillNotAllowed {},

    #[error("Invalid relayer")]
    InvalidRelayer {},
}

