#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, Uint128,
    CosmosMsg, BankMsg, WasmMsg, from_binary
};
use cw2::set_contract_version;
use cw20::{Cw20ExecuteMsg, Cw20ReceiveMsg};

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg, ReceiveMsg, EscrowResponse};
use crate::state::{EscrowInfo, EscrowStatus, ESCROW_INFO};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:destination_escrow";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let taker = deps.api.addr_validate(&msg.taker)?;
    let maker = deps.api.addr_validate(&msg.maker)?;

    let escrow_info = EscrowInfo {
        taker: taker.clone(),
        maker: maker.clone(),
        secret_hash: msg.secret_hash,
        timelock: msg.timelock,
        src_chain_id: msg.src_chain_id,
        src_escrow_address: msg.src_escrow_address,
        expected_amount: msg.expected_amount,
        deposited_amount: Uint128::zero(),
        deposited_denom: None,
        cw20_contract: None,
        status: EscrowStatus::Active,
        created_at: env.block.time.seconds(),
        src_confirmed: false,
        src_tx_hash: None,
        src_block_height: None,
    };

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("taker", taker)
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
        ExecuteMsg::ConfirmSourceEscrow { src_tx_hash, block_height } => {
            execute_confirm_source_escrow(deps, env, info, src_tx_hash, block_height)
        }
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

    if info.sender != escrow_info.taker {
        return Err(ContractError::Unauthorized {});
    }

    if info.funds.len() != 1 {
        return Err(ContractError::InsufficientFunds {});
    }

    let coin = &info.funds[0];
    if coin.amount != escrow_info.expected_amount {
        return Err(ContractError::InvalidAmount {});
    }

    escrow_info.deposited_amount = coin.amount;
    escrow_info.deposited_denom = Some(coin.denom.clone());

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

            if sender != escrow_info.taker {
                return Err(ContractError::Unauthorized {});
            }

            if amount != escrow_info.expected_amount {
                return Err(ContractError::InvalidAmount {});
            }

            escrow_info.deposited_amount = amount;
            escrow_info.cw20_contract = Some(info.sender);

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

    // Only maker can withdraw
    if info.sender != escrow_info.maker {
        return Err(ContractError::Unauthorized {});
    }

    // Source escrow must be confirmed
    if !escrow_info.src_confirmed {
        return Err(ContractError::SourceEscrowNotConfirmed {});
    }

    // Verify secret hash
    let secret_hash = format!("{:x}", sha2::Sha256::digest(secret.as_bytes()));
    if secret_hash != escrow_info.secret_hash {
        return Err(ContractError::InvalidSecret {});
    }

    let mut messages = vec![];

    // Transfer tokens to maker
    if let Some(cw20_contract) = &escrow_info.cw20_contract {
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cw20_contract.to_string(),
            msg: to_binary(&Cw20ExecuteMsg::Transfer {
                recipient: escrow_info.maker.to_string(),
                amount: escrow_info.deposited_amount,
            })?,
            funds: vec![],
        }));
    } else if let Some(denom) = &escrow_info.deposited_denom {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: escrow_info.maker.to_string(),
            amount: vec![cosmwasm_std::Coin {
                denom: denom.clone(),
                amount: escrow_info.deposited_amount,
            }],
        }));
    }

    escrow_info.status = EscrowStatus::Withdrawn;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("method", "withdraw")
        .add_attribute("maker", escrow_info.maker)
        .add_attribute("amount", escrow_info.deposited_amount))
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

    if info.sender != escrow_info.taker {
        return Err(ContractError::Unauthorized {});
    }

    if env.block.time.seconds() < escrow_info.timelock {
        return Err(ContractError::TimelockNotExpired {});
    }

    let mut messages = vec![];

    // Return tokens to taker
    if let Some(cw20_contract) = &escrow_info.cw20_contract {
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cw20_contract.to_string(),
            msg: to_binary(&Cw20ExecuteMsg::Transfer {
                recipient: escrow_info.taker.to_string(),
                amount: escrow_info.deposited_amount,
            })?,
            funds: vec![],
        }));
    } else if let Some(denom) = &escrow_info.deposited_denom {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: escrow_info.taker.to_string(),
            amount: vec![cosmwasm_std::Coin {
                denom: denom.clone(),
                amount: escrow_info.deposited_amount,
            }],
        }));
    }

    escrow_info.status = EscrowStatus::Cancelled;
    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("method", "cancel")
        .add_attribute("taker", escrow_info.taker)
        .add_attribute("returned_amount", escrow_info.deposited_amount))
}

pub fn execute_confirm_source_escrow(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    src_tx_hash: String,
    block_height: u64,
) -> Result<Response, ContractError> {
    let mut escrow_info = ESCROW_INFO.load(deps.storage)?;

    // TODO: Add authorization check for relayer
    // if info.sender != authorized_relayer {
    //     return Err(ContractError::Unauthorized {});
    // }

    escrow_info.src_confirmed = true;
    escrow_info.src_tx_hash = Some(src_tx_hash.clone());
    escrow_info.src_block_height = Some(block_height);

    ESCROW_INFO.save(deps.storage, &escrow_info)?;

    Ok(Response::new()
        .add_attribute("method", "confirm_source_escrow")
        .add_attribute("src_tx_hash", src_tx_hash)
        .add_attribute("block_height", block_height.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Escrow {} => to_binary(&query_escrow(deps)?),
    }
}

fn query_escrow(deps: Deps) -> StdResult<EscrowResponse> {
    let escrow_info = ESCROW_INFO.load(deps.storage)?;
    Ok(EscrowResponse {
        taker: escrow_info.taker,
        maker: escrow_info.maker,
        secret_hash: escrow_info.secret_hash,
        timelock: escrow_info.timelock,
        src_chain_id: escrow_info.src_chain_id,
        src_escrow_address: escrow_info.src_escrow_address,
        expected_amount: escrow_info.expected_amount,
        deposited_amount: escrow_info.deposited_amount,
        deposited_denom: escrow_info.deposited_denom,
        cw20_contract: escrow_info.cw20_contract,
        status: escrow_info.status,
        created_at: escrow_info.created_at,
        src_confirmed: escrow_info.src_confirmed,
        src_tx_hash: escrow_info.src_tx_hash,
        src_block_height: escrow_info.src_block_height,
    })
}

