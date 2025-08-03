 #[error("Unauthorized")]
    Unauthorized {},

    #[error("Order not found")]
    OrderNotFound {},

    #[error("Order not active")]
    OrderNotActive {},

    #[error("Fill amount too large")]
    FillAmountTooLarge {},

    #[error("Order already filled")]
    OrderAlreadyFilled {},

    #[error("Invalid fill amount")]
    InvalidFillAmount {},
}
