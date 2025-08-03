#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, SubMsg,
    WasmMsg, ReplyOn, Reply, Uint128
};
use cw2::set_contract_version;
use cw_utils::parse_reply_instantiate_data;

use crate::error::ContractError;
use crate::msg::{
    ExecuteMsg, InstantiateMsg, QueryMsg, ConfigResponse, EscrowAddressResponse,
    EscrowListResponse, EscrowInfo, EscrowType
};
use crate::state::{Config, CONFIG, ESCROWS};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:escrow_factory";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

// Reply IDs
const INSTANTIATE_SOURCE_ESCROW_REPLY_ID: u64 = 1;
const INSTANTIATE_DESTINATION_ESCROW_REPLY_ID: u64 = 2;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let owner = deps.api.addr_validate(&msg.owner)?;

    let config = Config {
        owner: owner.clone(),
        source_escrow_code_id: msg.source_escrow_code_id,
        destination_escrow_code_id: msg.destination_escrow_code_id,
    };

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", owner)
        .add_attribute("source_escrow_code_id", msg.source_escrow_code_id.to_string())
        .add_attribute("destination_escrow_code_id", msg.destination_escrow_code_id.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::CreateSourceEscrow {
            maker,
            taker,
            secret_hash,
            timelock,
            dst_chain_id,
            dst_asset,
            dst_amount,
            initial_price,
            price_decay_rate,
            minimum_price,
            allow_partial_fill,
            minimum_fill_amount,
            label,
        } => execute_create_source_escrow(
            deps,
            env,
            info,
            maker,
            taker,
            secret_hash,
            timelock,
            dst_chain_id,
            dst_asset,
            dst_amount,
            initial_price,
            price_decay_rate,
            minimum_price,
            allow_partial_fill,
            minimum_fill_amount,
            label,
        ),
        ExecuteMsg::CreateDestinationEscrow {
            taker,
            maker,
            secret_hash,
            timelock,
            src_chain_id,
            src_escrow_address,
            expected_amount,
            label,
        } => execute_create_destination_escrow(
            deps,
            env,
            info,
            taker,
            maker,
            secret_hash,
            timelock,
            src_chain_id,
            src_escrow_address,
            expected_amount,
            label,
        ),
        ExecuteMsg::UpdateCodeIds {
            source_escrow_code_id,
            destination_escrow_code_id,
        } => execute_update_code_ids(deps, info, source_escrow_code_id, destination_escrow_code_id),
        ExecuteMsg::UpdateOwner { new_owner } => execute_update_owner(deps, info, new_owner),
    }
}

pub fn execute_create_source_escrow(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    maker: String,
    taker: Option<String>,
    secret_hash: String,
    timelock: u64,
    dst_chain_id: String,
    dst_asset: String,
    dst_amount: Uint128,
    initial_price: Option<Uint128>,
    price_decay_rate: Option<Uint128>,
    minimum_price: Option<Uint128>,
    allow_partial_fill: bool,
    minimum_fill_amount: Option<Uint128>,
    label: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    // Generate salt for deterministic address
    let salt = format!("{}:{}:{}", info.sender, env.block.time.nanos(), label);

    // Check if escrow already exists
    if ESCROWS.has(deps.storage, salt.clone()) {
        return Err(ContractError::EscrowAlreadyExists {});
    }

    let instantiate_msg = source_escrow::msg::InstantiateMsg {
        maker,
        taker,
        secret_hash,
        timelock,
        dst_chain_id,
        dst_asset,
        dst_amount,
        initial_price,
        price_decay_rate,
        minimum_price,
        allow_partial_fill,
        minimum_fill_amount,
    };

    let wasm_msg = WasmMsg::Instantiate {
        admin: Some(env.contract.address.to_string()),
        code_id: config.source_escrow_code_id,
        msg: to_binary(&instantiate_msg)?,
        funds: vec![],
        label: format!("source_escrow_{}", salt),
    };

    let sub_msg = SubMsg {
        id: INSTANTIATE_SOURCE_ESCROW_REPLY_ID,
        msg: wasm_msg.into(),
        gas_limit: None,
        reply_on: ReplyOn::Success,
    };

    // Store pending escrow info
    let escrow_info = EscrowInfo {
        address: deps.api.addr_validate("pending")?, // Will be updated in reply
        escrow_type: EscrowType::Source,
        creator: info.sender,
        created_at: env.block.time.seconds(),
        salt: salt.clone(),
    };
    ESCROWS.save(deps.storage, salt.clone(), &escrow_info)?;

    Ok(Response::new()
        .add_submessage(sub_msg)
        .add_attribute("method", "create_source_escrow")
        .add_attribute("salt", salt))
}

pub fn execute_create_destination_escrow(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    taker: String,
    maker: String,
    secret_hash: String,
    timelock: u64,
    src_chain_id: String,
    src_escrow_address: String,
    expected_amount: Uint128,
    label: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;

    // Generate salt for deterministic address
    let salt = format!("{}:{}:{}", info.sender, env.block.time.nanos(), label);

    // Check if escrow already exists
    if ESCROWS.has(deps.storage, salt.clone()) {
        return Err(ContractError::EscrowAlreadyExists {});
    }

    let instantiate_msg = destination_escrow::msg::InstantiateMsg {
        taker,
        maker,
        secret_hash,
        timelock,
        src_chain_id,
        src_escrow_address,
        expected_amount,
    };

    let wasm_msg = WasmMsg::Instantiate {
        admin: Some(env.contract.address.to_string()),
        code_id: config.destination_escrow_code_id,
        msg: to_binary(&instantiate_msg)?,
        funds: vec![],
        label: format!("destination_escrow_{}", salt),
    };

    let sub_msg = SubMsg {
        id: INSTANTIATE_DESTINATION_ESCROW_REPLY_ID,
        msg: wasm_msg.into(),
        gas_limit: None,
        reply_on: ReplyOn::Success,
    };

    // Store pending escrow info
    let escrow_info = EscrowInfo {
        address: deps.api.addr_validate("pending")?, // Will be updated in reply
        escrow_type: EscrowType::Destination,
        creator: info.sender,
        created_at: env.block.time.seconds(),
        salt: salt.clone(),
    };
    ESCROWS.save(deps.storage, salt.clone(), &escrow_info)?;

    Ok(Response::new()
        .add_submessage(sub_msg)
        .add_attribute("method", "create_destination_escrow")
        .add_attribute("salt", salt))
}

pub fn execute_update_code_ids(
    deps: DepsMut,
    info: MessageInfo,
    source_escrow_code_id: Option<u64>,
    destination_escrow_code_id: Option<u64>,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    if let Some(code_id) = source_escrow_code_id {
        config.source_escrow_code_id = code_id;
    }

    if let Some(code_id) = destination_escrow_code_id {
        config.destination_escrow_code_id = code_id;
    }

    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "update_code_ids")
        .add_attribute("source_escrow_code_id", config.source_escrow_code_id.to_string())
        .add_attribute("destination_escrow_code_id", config.destination_escrow_code_id.to_string()))
}

pub fn execute_update_owner(
    deps: DepsMut,
    info: MessageInfo,
    new_owner: String,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;

    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    let new_owner = deps.api.addr_validate(&new_owner)?;
    config.owner = new_owner.clone();

    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "update_owner")
        .add_attribute("new_owner", new_owner))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(deps: DepsMut, _env: Env, msg: Reply) -> Result<Response, ContractError> {
    match msg.id {
        INSTANTIATE_SOURCE_ESCROW_REPLY_ID | INSTANTIATE_DESTINATION_ESCROW_REPLY_ID => {
            handle_instantiate_reply(deps, msg)
        }
        id => Err(ContractError::Std(cosmwasm_std::StdError::generic_err(
            format!("Unknown reply id: {}", id),
        ))),
    }
}

fn handle_instantiate_reply(deps: DepsMut, msg: Reply) -> Result<Response, ContractError> {
    let reply = parse_reply_instantiate_data(msg)?;
    let contract_address = deps.api.addr_validate(&reply.contract_address)?;

    // Find the pending escrow and update its address
    // This is a simplified approach - in production, you might want to store the salt in the reply data
    let escrows: Vec<_> = ESCROWS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (salt, mut escrow_info) in escrows {
        if escrow_info.address == deps.api.addr_validate("pending")? {
            escrow_info.address = contract_address.clone();
            ESCROWS.save(deps.storage, salt, &escrow_info)?;
            break;
        }
    }

    Ok(Response::new()
        .add_attribute("method", "handle_instantiate_reply")
        .add_attribute("contract_address", contract_address))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Config {} => to_binary(&query_config(deps)?),
        QueryMsg::EscrowAddress { salt } => to_binary(&query_escrow_address(deps, salt)?),
        QueryMsg::EscrowList { start_after, limit } => {
            to_binary(&query_escrow_list(deps, start_after, limit)?)
        }
    }
}

fn query_config(deps: Deps) -> StdResult<ConfigResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(ConfigResponse {
        owner: config.owner,
        source_escrow_code_id: config.source_escrow_code_id,
        destination_escrow_code_id: config.destination_escrow_code_id,
    })
}

fn query_escrow_address(deps: Deps, salt: String) -> StdResult<EscrowAddressResponse> {
    let escrow_info = ESCROWS.load(deps.storage, salt)?;
    Ok(EscrowAddressResponse {
        address: escrow_info.address.to_string(),
    })
}

fn query_escrow_list(
    deps: Deps,
    start_after: Option<String>,
    limit: Option<u32>,
) -> StdResult<EscrowListResponse> {
    let limit = limit.unwrap_or(30).min(100) as usize;
    let start = start_after.as_ref().map(|s| cosmwasm_std::Bound::exclusive(s.as_str()));

    let escrows: StdResult<Vec<_>> = ESCROWS
        .range(deps.storage, start, None, cosmwasm_std::Order::Ascending)
        .take(limit)
        .map(|item| item.map(|(_, escrow_info)| escrow_info))
        .collect();

    Ok(EscrowListResponse {
        escrows: escrows?,
    })
}

