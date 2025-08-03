use cosmwasm_std::StdError;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),

    #[error("Unauthorized")]
    Unauthorized {},

    #[error("Invalid escrow type")]
    InvalidEscrowType {},

    #[error("Escrow already exists")]
    EscrowAlreadyExists {},
}

