use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{Order, ORDERS};

const CONTRACT_NAME: &str = "partial-fill-simple";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", info.sender))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::CreateOrder { order_id, total_amount, price } => {
            execute_create_order(deps, env, info, order_id, total_amount, price)
        }
        ExecuteMsg::PartialFill { order_id, fill_amount } => {
            execute_partial_fill(deps, env, info, order_id, fill_amount)
        }
        ExecuteMsg::CancelOrder { order_id } => {
            execute_cancel_order(deps, env, info, order_id)
        }
    }
}

pub fn execute_create_order(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    order_id: String,
    total_amount: Uint128,
    price: Uint128,
) -> Result<Response, ContractError> {
    // Check if order already exists
    if ORDERS.may_load(deps.storage, order_id.clone())?.is_some() {
        return Err(ContractError::OrderNotFound {});
    }

    let order = Order {
        maker: info.sender.clone(),
        taker: None,
        total_amount,
        filled_amount: Uint128::zero(),
        price,
        is_active: true,
    };

    ORDERS.save(deps.storage, order_id.clone(), &order)?;

    Ok(Response::new()
        .add_attribute("method", "create_order")
        .add_attribute("order_id", order_id)
        .add_attribute("maker", info.sender)
        .add_attribute("total_amount", total_amount)
        .add_attribute("price", price))
}

pub fn execute_partial_fill(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    order_id: String,
    fill_amount: Uint128,
) -> Result<Response, ContractError> {
    let mut order = ORDERS.load(deps.storage, order_id.clone())?;

    if !order.is_active {
        return Err(ContractError::OrderNotActive {});
    }

    if order.is_fully_filled() {
        return Err(ContractError::OrderAlreadyFilled {});
    }

    if fill_amount.is_zero() {
        return Err(ContractError::InvalidFillAmount {});
    }

    if fill_amount > order.remaining_amount() {
        return Err(ContractError::FillAmountTooLarge {});
    }

    // Calculate payment required
    let payment_required = fill_amount * order.price;
    let payment_received = info.funds.iter()
        .find(|c| c.denom == "uatom")
        .map(|c| c.amount)
        .unwrap_or_else(Uint128::zero);

    if payment_received < payment_required {
        return Err(ContractError::InvalidFillAmount {});
    }

    // Update order
    order.filled_amount += fill_amount;
    if order.taker.is_none() {
        order.taker = Some(info.sender.clone());
    }

    // Check if fully filled
    if order.is_fully_filled() {
        order.is_active = false;
    }

    ORDERS.save(deps.storage, order_id.clone(), &order)?;

    // Send payment to maker
    let payment_msg = BankMsg::Send {
        to_address: order.maker.to_string(),
        amount: vec![coin(payment_required.u128(), "uatom")],
    };

    // Refund excess payment if any
    let mut response = Response::new().add_message(CosmosMsg::Bank(payment_msg));
    
    if payment_received > payment_required {
        let refund_amount = payment_received - payment_required;
        let refund_msg = BankMsg::Send {
            to_address: info.sender.to_string(),
            amount: vec![coin(refund_amount.u128(), "uatom")],
        };
        response = response.add_message(CosmosMsg::Bank(refund_msg));
    }

    Ok(response
        .add_attribute("method", "partial_fill")
        .add_attribute("order_id", order_id)
        .add_attribute("taker", info.sender)
        .add_attribute("fill_amount", fill_amount)
        .add_attribute("filled_amount", order.filled_amount)
        .add_attribute("is_fully_filled", order.is_fully_filled().to_string()))
}

pub fn execute_cancel_order(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    order_id: String,
) -> Result<Response, ContractError> {
    let mut order = ORDERS.load(deps.storage, order_id.clone())?;

    if order.maker != info.sender {
        return Err(ContractError::Unauthorized {});
    }

    if !order.is_active {
        return Err(ContractError::OrderNotActive {});
    }

    order.is_active = false;
    ORDERS.save(deps.storage, order_id.clone(), &order)?;

    Ok(Response::new()
        .add_attribute("method", "cancel_order")
        .add_attribute("order_id", order_id)
        .add_attribute("maker", info.sender))
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetOrder { order_id } => {
            let order = ORDERS.load(deps.storage, order_id)?;
            to_binary(&order)
        }
        QueryMsg::GetOrderStatus { order_id } => {
            let order = ORDERS.load(deps.storage, order_id)?;
            let status = serde_json::json!({
                "is_active": order.is_active,
                "is_fully_filled": order.is_fully_filled(),
                "fill_percentage": order.fill_percentage(),
                "remaining_amount": order.remaining_amount()
            });
            to_binary(&status)
        }
    }
}
