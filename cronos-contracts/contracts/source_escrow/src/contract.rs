#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, Uint128,
    CosmosMsg, BankMsg, WasmMsg, from_binary, Addr
};
use cw2::set_contract_version;
use cw20::{Cw20ExecuteMsg, Cw20ReceiveMsg};

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg, ReceiveMsg, EscrowResponse, PriceResponse, FillStatusResponse};
use crate::state::{EscrowInfo, EscrowStatus, ESCROW_INFO};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:source_escrow";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let maker = deps.api.addr_validate(&msg.maker)?;
    let taker = msg.taker.map(|t| deps.api.addr_validate(&t)).transpose()?;

    // Validate dutch auction parameters
    if let (Some(initial_price), Some(minimum_price)) = (&msg.initial_price, &msg.minimum_price) {
        if initial_price <= minimum_price {
            return Err(ContractError::InvalidDutchAuctionParams {});
        }
    }

    let escrow_info = EscrowInfo {
        maker: maker.clone(),
        taker,
        secret_hash: msg.secret_hash,
        timelock: msg.timelock,
        dst_chain_id: msg.dst_chain_id,
        dst_asset: msg.dst_asset,
        dst_amount: msg.dst_amount,
        deposited_amount: Uint128::zero(),
        deposited_denom: None,
        cw20_contract: None,
        status: EscrowStatus::Active,
        created_at: env.block.time.seconds(),
        initial_price: msg.initial_price,
        price_decay_rate: msg.price_decay_rate,
        minimum_price: msg.minimum_price,
        allow_partial_fill: msg.allow_partial_fill,
        minimum_fill_amount: msg.minimum_fill_amount,
        filled_amount: Uint128::zero(),
        remaining_amount: Uint128::zero(), // Will be set when deposit is made
    };

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("maker", maker)
        .add_attribute("timelock", msg.timelock.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Deposit {} => execute_deposit(deps, env, info),
        ExecuteMsg::Receive(msg) => execute_receive(deps, env, info, msg),
        ExecuteMsg::Withdraw { secret } => execute_withdraw(deps, env, info, secret),
        ExecuteMsg::Cancel {} => execute_cancel(deps, env, info),
        ExecuteMsg::PartialWithdraw { secret, amount } => {
            execute_partial_withdraw(deps, env, info, secret, amount)
        }
        ExecuteMsg::UpdatePrice {} => execute_update_price(deps, env, info),
    }
}

pub fn execute_deposit(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
) -> Result<Response, ContractError> {
    let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

    if escrow_info.status != EscrowStatus::Active {
        return Err(ContractError::AlreadyWithdrawn {});
    }

    if info.sender != escrow_info.maker {
        return Err(ContractError::Unauthorized {});
    }

    if info.funds.len() != 1 {
        return Err(ContractError::InsufficientFunds {});
    }

    let coin = &info.funds[0];
    escrow_info.deposited_amount = coin.amount;
    escrow_info.deposited_denom = Some(coin.denom.clone());
    escrow_info.remaining_amount = coin.amount;

    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_attribute("method", "deposit")
        .add_attribute("amount", coin.amount)
        .add_attribute("denom", &coin.denom))
}

pub fn execute_receive(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    wrapper: Cw20ReceiveMsg,
) -> Result<Response, ContractError> {
    let msg: ReceiveMsg = from_binary(&wrapper.msg)?;
    let amount = wrapper.amount;
    let sender = deps.api.addr_validate(&wrapper.sender)?;

    match msg {
        ReceiveMsg::Deposit {} => {
            let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

            if escrow_info.status != EscrowStatus::Active {
                return Err(ContractError::AlreadyWithdrawn {});
            }

            if sender != escrow_info.maker {
                return Err(ContractError::Unauthorized {});
            }

            escrow_info.deposited_amount = amount;
            escrow_info.cw20_contract = Some(info.sender);
            escrow_info.remaining_amount = amount;

            ESCROW_INFO.save(deps.storage, &escrow_info)?;

            Ok(Response::new()
                .add_attribute("method", "receive_deposit")
                .add_attribute("amount", amount)
                .add_attribute("from", sender))
        }
    }
}

pub fn execute_withdraw(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    secret: String,
) -> Result<Response, ContractError> {
    let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

    if escrow_info.status == EscrowStatus::Withdrawn {
        return Err(ContractError::AlreadyWithdrawn {});
    }

    if escrow_info.status == EscrowStatus::Cancelled {
        return Err(ContractError::AlreadyCancelled {});
    }

    // Verify secret hash
    let secret_hash = format!("{:x}", sha2::Sha256::digest(secret.as_bytes()));
    if secret_hash != escrow_info.secret_hash {
        return Err(ContractError::InvalidSecret {});
    }

    let withdraw_amount = if escrow_info.allow_partial_fill {
        escrow_info.remaining_amount
    } else {
        escrow_info.deposited_amount
    };

    let mut messages = vec![];

    // Transfer tokens to taker or sender
    let recipient = escrow_info.taker.as_ref().unwrap_or(&info.sender);
    
    if let Some(cw20_contract) = &escrow_info.cw20_contract {
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cw20_contract.to_string(),
            msg: to_binary(&Cw20ExecuteMsg::Transfer {
                recipient: recipient.to_string(),
                amount: withdraw_amount,
            })?,
            funds: vec![],
        }));
    } else if let Some(denom) = &escrow_info.deposited_denom {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: recipient.to_string(),
            amount: vec![cosmwasm_std::Coin {
                denom: denom.clone(),
                amount: withdraw_amount,
            }],
        }));
    }

    escrow_info.status = EscrowStatus::Withdrawn;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("method", "withdraw")
        .add_attribute("recipient", recipient)
        .add_attribute("amount", withdraw_amount))
}

pub fn execute_partial_withdraw(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    secret: String,
    amount: Uint128,
) -> Result<Response, ContractError> {
    let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

    if !escrow_info.allow_partial_fill {
        return Err(ContractError::InvalidPartialFillAmount {});
    }

    if escrow_info.status == EscrowStatus::Withdrawn {
        return Err(ContractError::AlreadyWithdrawn {});
    }

    if escrow_info.status == EscrowStatus::Cancelled {
        return Err(ContractError::AlreadyCancelled {});
    }

    if amount > escrow_info.remaining_amount {
        return Err(ContractError::InsufficientFunds {});
    }

    if let Some(min_fill) = escrow_info.minimum_fill_amount {
        if amount < min_fill {
            return Err(ContractError::InvalidPartialFillAmount {});
        }
    }

    // Verify secret hash
    let secret_hash = format!("{:x}", sha2::Sha256::digest(secret.as_bytes()));
    if secret_hash != escrow_info.secret_hash {
        return Err(ContractError::InvalidSecret {});
    }

    let mut messages = vec![];

    // Transfer tokens to taker or sender
    let recipient = escrow_info.taker.as_ref().unwrap_or(&info.sender);
    
    if let Some(cw20_contract) = &escrow_info.cw20_contract {
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cw20_contract.to_string(),
            msg: to_binary(&Cw20ExecuteMsg::Transfer {
                recipient: recipient.to_string(),
                amount,
            })?,
            funds: vec![],
        }));
    } else if let Some(denom) = &escrow_info.deposited_denom {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: recipient.to_string(),
            amount: vec![cosmwasm_std::Coin {
                denom: denom.clone(),
                amount,
            }],
        }));
    }

    // Update escrow state
    escrow_info.filled_amount += amount;
    escrow_info.remaining_amount -= amount;

    if escrow_info.remaining_amount.is_zero() {
        escrow_info.status = EscrowStatus::Withdrawn;
    } else {
        escrow_info.status = EscrowStatus::PartiallyFilled;
    }

    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("method", "partial_withdraw")
        .add_attribute("recipient", recipient)
        .add_attribute("amount", amount)
        .add_attribute("remaining", escrow_info.remaining_amount))
}

pub fn execute_cancel(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
) -> Result<Response, ContractError> {
    let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

    if escrow_info.status == EscrowStatus::Withdrawn {
        return Err(ContractError::AlreadyWithdrawn {});
    }

    if escrow_info.status == EscrowStatus::Cancelled {
        return Err(ContractError::AlreadyCancelled {});
    }

    if info.sender != escrow_info.maker {
        return Err(ContractError::Unauthorized {});
    }

    if env.block.time.seconds() < escrow_info.timelock {
        return Err(ContractError::TimelockNotExpired {});
    }

    let mut messages = vec![];

    // Return remaining tokens to maker
    let return_amount = escrow_info.remaining_amount;
    
    if let Some(cw20_contract) = &escrow_info.cw20_contract {
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cw20_contract.to_string(),
            msg: to_binary(&Cw20ExecuteMsg::Transfer {
                recipient: escrow_info.maker.to_string(),
                amount: return_amount,
            })?,
            funds: vec![],
        }));
    } else if let Some(denom) = &escrow_info.deposited_denom {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: escrow_info.maker.to_string(),
            amount: vec![cosmwasm_std::Coin {
                denom: denom.clone(),
                amount: return_amount,
            }],
        }));
    }

    escrow_info.status = EscrowStatus::Cancelled;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("method", "cancel")
        .add_attribute("maker", escrow_info.maker)
        .add_attribute("returned_amount", return_amount))
}

pub fn execute_update_price(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
) -> Result<Response, ContractError> {
    let escrow_info = ESCROW_INFO.load(deps.storage)?;
    
    let current_price = calculate_current_price(&escrow_info, env.block.time.seconds())?;
    
    Ok(Response::new()
        .add_attribute("method", "update_price")
        .add_attribute("current_price", current_price))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Escrow {} => to_binary(&query_escrow(deps)?),
        QueryMsg::CurrentPrice {} => to_binary(&query_current_price(deps, env)?),
        QueryMsg::FillStatus {} => to_binary(&query_fill_status(deps)?),
    }
}

fn query_escrow(deps: Deps) -> StdResult<EscrowResponse> {
    let escrow_info = ESCROW_INFO.load(deps.storage)?;
    Ok(EscrowResponse {
        maker: escrow_info.maker,
        taker: escrow_info.taker,
        secret_hash: escrow_info.secret_hash,
        timelock: escrow_info.timelock,
        dst_chain_id: escrow_info.dst_chain_id,
        dst_asset: escrow_info.dst_asset,
        dst_amount: escrow_info.dst_amount,
        deposited_amount: escrow_info.deposited_amount,
        deposited_denom: escrow_info.deposited_denom,
        cw20_contract: escrow_info.cw20_contract,
        status: escrow_info.status,
        created_at: escrow_info.created_at,
        allow_partial_fill: escrow_info.allow_partial_fill,
        filled_amount: escrow_info.filled_amount,
        remaining_amount: escrow_info.remaining_amount,
    })
}

fn query_current_price(deps: Deps, env: Env) -> StdResult<PriceResponse> {
    let escrow_info = ESCROW_INFO.load(deps.storage)?;
    let current_time = env.block.time.seconds();
    
    let current_price = calculate_current_price(&escrow_info, current_time)
        .unwrap_or(escrow_info.initial_price.unwrap_or(Uint128::zero()));
    
    Ok(PriceResponse {
        current_price,
        initial_price: escrow_info.initial_price,
        minimum_price: escrow_info.minimum_price,
        price_decay_rate: escrow_info.price_decay_rate,
        time_elapsed: current_time - escrow_info.created_at,
    })
}

fn query_fill_status(deps: Deps) -> StdResult<FillStatusResponse> {
    let escrow_info = ESCROW_INFO.load(deps.storage)?;
    Ok(FillStatusResponse {
        total_amount: escrow_info.deposited_amount,
        filled_amount: escrow_info.filled_amount,
        remaining_amount: escrow_info.remaining_amount,
        is_fully_filled: escrow_info.remaining_amount.is_zero(),
        allow_partial_fill: escrow_info.allow_partial_fill,
    })
}

fn calculate_current_price(escrow_info: &EscrowInfo, current_time: u64) -> Result<Uint128, ContractError> {
    if let (Some(initial_price), Some(decay_rate), Some(min_price)) = (
        &escrow_info.initial_price,
        &escrow_info.price_decay_rate,
        &escrow_info.minimum_price,
    ) {
        let time_elapsed = current_time - escrow_info.created_at;
        let price_decrease = decay_rate.checked_mul(Uint128::from(time_elapsed))
            .map_err(|_| ContractError::InvalidDutchAuctionParams {})?;
        
        let current_price = if price_decrease >= *initial_price {
            *min_price
        } else {
            initial_price.checked_sub(price_decrease)
                .map_err(|_| ContractError::InvalidDutchAuctionParams {})?
                .max(*min_price)
        };
        
        Ok(current_price)
    } else {
        Ok(escrow_info.initial_price.unwrap_or(Uint128::zero()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};
    use cosmwasm_std::{coins, from_binary};

    #[test]
    fn proper_initialization() {
        let mut deps = mock_dependencies();

        let msg = InstantiateMsg {
            maker: "maker".to_string(),
            taker: Some("taker".to_string()),
            secret_hash: "hash123".to_string(),
            timelock: 1000,
            dst_chain_id: "ethereum-1".to_string(),
            dst_asset: "ETH".to_string(),
            dst_amount: Uint128::from(100u128),
            initial_price: Some(Uint128::from(200u128)),
            price_decay_rate: Some(Uint128::from(1u128)),
            minimum_price: Some(Uint128::from(100u128)),
            allow_partial_fill: true,
            minimum_fill_amount: Some(Uint128::from(10u128)),
        };
        let info = mock_info("creator", &coins(1000, "earth"));

        let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(0, res.messages.len());
    }
}

