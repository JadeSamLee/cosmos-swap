#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult, Uint128,
    WasmMsg, CosmosMsg
};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{
    ExecuteMsg, InstantiateMsg, QueryMsg, OrderAction, ConfigResponse, OrderResponse,
    OrderListResponse, PriceResponse, RelayerResponse, OrderStatus, DutchAuctionInfo,
    PartialFillInfo
};
use crate::state::{Config, Order, CONFIG, ORDERS, ORDER_COUNT};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:escrow_resolver";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let owner = deps.api.addr_validate(&msg.owner)?;
    let escrow_factory = deps.api.addr_validate(&msg.escrow_factory)?;
    
    let mut authorized_relayers = Vec::new();
    for relayer in msg.authorized_relayers {
        authorized_relayers.push(deps.api.addr_validate(&relayer)?);
    }

    let config = Config {
        owner: owner.clone(),
        escrow_factory,
        authorized_relayers,
    };

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;
    CONFIG.save(deps.storage, &config)?;
    ORDER_COUNT.save(deps.storage, &0u64)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("owner", owner)
        .add_attribute("escrow_factory", config.escrow_factory))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::DeploySrc {
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
            lop_order_data,
            label,
        } => execute_deploy_src(
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
            lop_order_data,
            label,
        ),
        ExecuteMsg::DeployDst {
            taker,
            maker,
            secret_hash,
            timelock,
            src_chain_id,
            src_escrow_address,
            expected_amount,
            label,
        } => execute_deploy_dst(
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
        ExecuteMsg::Withdraw { escrow_address, secret } => {
            execute_withdraw(deps, env, info, escrow_address, secret)
        }
        ExecuteMsg::PartialWithdraw { escrow_address, secret, amount } => {
            execute_partial_withdraw(deps, env, info, escrow_address, secret, amount)
        }
        ExecuteMsg::Cancel { escrow_address } => {
            execute_cancel(deps, env, info, escrow_address)
        }
        ExecuteMsg::UpdatePrice { escrow_address } => {
            execute_update_price(deps, env, info, escrow_address)
        }
        ExecuteMsg::ProcessOrder { order_id, action, proof } => {
            execute_process_order(deps, env, info, order_id, action, proof)
        }
        ExecuteMsg::AddRelayer { relayer } => {
            execute_add_relayer(deps, info, relayer)
        }
        ExecuteMsg::RemoveRelayer { relayer } => {
            execute_remove_relayer(deps, info, relayer)
        }
        ExecuteMsg::UpdateOwner { new_owner } => {
            execute_update_owner(deps, info, new_owner)
        }
    }
}

pub fn execute_deploy_src(
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
    lop_order_data: Option<String>,
    label: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    
    // Only owner or authorized relayers can deploy escrows
    if info.sender != config.owner && !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    // Generate order ID
    let mut order_count = ORDER_COUNT.load(deps.storage)?;
    order_count += 1;
    ORDER_COUNT.save(deps.storage, &order_count)?;
    let order_id = format!("order_{}", order_count);

    // Create escrow through factory
    let create_escrow_msg = WasmMsg::Execute {
        contract_addr: config.escrow_factory.to_string(),
        msg: to_binary(&escrow_factory::msg::ExecuteMsg::CreateSourceEscrow {
            maker: maker.clone(),
            taker: taker.clone(),
            secret_hash: secret_hash.clone(),
            timelock,
            dst_chain_id: dst_chain_id.clone(),
            dst_asset,
            dst_amount,
            initial_price,
            price_decay_rate,
            minimum_price,
            allow_partial_fill,
            minimum_fill_amount,
            label: label.clone(),
        })?,
        funds: vec![],
    };

    // Create Dutch auction info if parameters provided
    let dutch_auction = if let (Some(init_price), Some(min_price), Some(decay_rate)) = 
        (initial_price, minimum_price, price_decay_rate) {
        Some(DutchAuctionInfo {
            initial_price: init_price,
            minimum_price: min_price,
            price_decay_rate: decay_rate,
            start_time: env.block.time.seconds(),
            current_price: init_price,
        })
    } else {
        None
    };

    // Create partial fill info if enabled
    let partial_fill = if allow_partial_fill {
        Some(PartialFillInfo {
            allow_partial_fill: true,
            minimum_fill_amount,
            filled_amount: Uint128::zero(),
            remaining_amount: dst_amount,
        })
    } else {
        None
    };

    // Store order information
    let order = Order {
        order_id: order_id.clone(),
        escrow_address: deps.api.addr_validate("pending")?, // Will be updated when escrow is created
        maker: deps.api.addr_validate(&maker)?,
        taker: taker.as_ref().map(|t| deps.api.addr_validate(t)).transpose()?,
        status: OrderStatus::Active,
        created_at: env.block.time.seconds(),
        updated_at: env.block.time.seconds(),
        dutch_auction,
        partial_fill,
        lop_order_data,
    };

    ORDERS.save(deps.storage, order_id.clone(), &order)?;

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(create_escrow_msg))
        .add_attribute("method", "deploy_src")
        .add_attribute("order_id", order_id)
        .add_attribute("maker", maker)
        .add_attribute("dst_chain_id", dst_chain_id))
}

pub fn execute_deploy_dst(
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
    
    // Only owner or authorized relayers can deploy escrows
    if info.sender != config.owner && !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    // Generate order ID
    let mut order_count = ORDER_COUNT.load(deps.storage)?;
    order_count += 1;
    ORDER_COUNT.save(deps.storage, &order_count)?;
    let order_id = format!("order_{}", order_count);

    // Create escrow through factory
    let create_escrow_msg = WasmMsg::Execute {
        contract_addr: config.escrow_factory.to_string(),
        msg: to_binary(&escrow_factory::msg::ExecuteMsg::CreateDestinationEscrow {
            taker: taker.clone(),
            maker: maker.clone(),
            secret_hash: secret_hash.clone(),
            timelock,
            src_chain_id: src_chain_id.clone(),
            src_escrow_address: src_escrow_address.clone(),
            expected_amount,
            label: label.clone(),
        })?,
        funds: vec![],
    };

    // Store order information
    let order = Order {
        order_id: order_id.clone(),
        escrow_address: deps.api.addr_validate("pending")?, // Will be updated when escrow is created
        maker: deps.api.addr_validate(&maker)?,
        taker: Some(deps.api.addr_validate(&taker)?),
        status: OrderStatus::Active,
        created_at: env.block.time.seconds(),
        updated_at: env.block.time.seconds(),
        dutch_auction: None,
        partial_fill: None,
        lop_order_data: None,
    };

    ORDERS.save(deps.storage, order_id.clone(), &order)?;

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(create_escrow_msg))
        .add_attribute("method", "deploy_dst")
        .add_attribute("order_id", order_id)
        .add_attribute("taker", taker)
        .add_attribute("maker", maker)
        .add_attribute("src_chain_id", src_chain_id))
}

pub fn execute_withdraw(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    escrow_address: String,
    secret: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    
    // Only owner or authorized relayers can execute withdrawals
    if info.sender != config.owner && !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    let escrow_addr = deps.api.addr_validate(&escrow_address)?;

    // Execute withdrawal on escrow contract
    let withdraw_msg = WasmMsg::Execute {
        contract_addr: escrow_address.clone(),
        msg: to_binary(&source_escrow::msg::ExecuteMsg::Withdraw { secret })?,
        funds: vec![],
    };

    // Update order status if found
    let orders: Vec<_> = ORDERS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (order_id, mut order) in orders {
        if order.escrow_address == escrow_addr {
            order.status = OrderStatus::Completed;
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id, &order)?;
            break;
        }
    }

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(withdraw_msg))
        .add_attribute("method", "withdraw")
        .add_attribute("escrow_address", escrow_address))
}

pub fn execute_partial_withdraw(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    escrow_address: String,
    secret: String,
    amount: Uint128,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    
    // Only owner or authorized relayers can execute withdrawals
    if info.sender != config.owner && !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    let escrow_addr = deps.api.addr_validate(&escrow_address)?;

    // Execute partial withdrawal on escrow contract
    let withdraw_msg = WasmMsg::Execute {
        contract_addr: escrow_address.clone(),
        msg: to_binary(&source_escrow::msg::ExecuteMsg::PartialWithdraw { secret, amount })?,
        funds: vec![],
    };

    // Update order partial fill info if found
    let orders: Vec<_> = ORDERS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (order_id, mut order) in orders {
        if order.escrow_address == escrow_addr {
            if let Some(ref mut partial_fill) = order.partial_fill {
                partial_fill.filled_amount += amount;
                partial_fill.remaining_amount -= amount;
                
                if partial_fill.remaining_amount.is_zero() {
                    order.status = OrderStatus::Completed;
                }
            }
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id, &order)?;
            break;
        }
    }

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(withdraw_msg))
        .add_attribute("method", "partial_withdraw")
        .add_attribute("escrow_address", escrow_address)
        .add_attribute("amount", amount))
}

pub fn execute_cancel(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    escrow_address: String,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    
    // Only owner or authorized relayers can cancel escrows
    if info.sender != config.owner && !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::Unauthorized {});
    }

    let escrow_addr = deps.api.addr_validate(&escrow_address)?;

    // Execute cancellation on escrow contract
    let cancel_msg = WasmMsg::Execute {
        contract_addr: escrow_address.clone(),
        msg: to_binary(&source_escrow::msg::ExecuteMsg::Cancel {})?,
        funds: vec![],
    };

    // Update order status if found
    let orders: Vec<_> = ORDERS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (order_id, mut order) in orders {
        if order.escrow_address == escrow_addr {
            order.status = OrderStatus::Cancelled;
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id, &order)?;
            break;
        }
    }

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(cancel_msg))
        .add_attribute("method", "cancel")
        .add_attribute("escrow_address", escrow_address))
}

pub fn execute_update_price(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    escrow_address: String,
) -> Result<Response, ContractError> {
    let escrow_addr = deps.api.addr_validate(&escrow_address)?;

    // Update Dutch auction price for the order
    let orders: Vec<_> = ORDERS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (order_id, mut order) in orders {
        if order.escrow_address == escrow_addr {
            if let Some(ref mut dutch_auction) = order.dutch_auction {
                let current_time = env.block.time.seconds();
                let time_elapsed = current_time - dutch_auction.start_time;
                
                // Calculate new price: price = initial_price - (decay_rate * time_elapsed)
                let price_decrease = dutch_auction.price_decay_rate.checked_mul(Uint128::from(time_elapsed))
                    .map_err(|_| ContractError::InvalidOrderParameters {})?;
                
                let new_price = if price_decrease >= dutch_auction.initial_price {
                    dutch_auction.minimum_price
                } else {
                    dutch_auction.initial_price.checked_sub(price_decrease)
                        .map_err(|_| ContractError::InvalidOrderParameters {})?
                        .max(dutch_auction.minimum_price)
                };
                
                dutch_auction.current_price = new_price;
                order.updated_at = current_time;
                ORDERS.save(deps.storage, order_id, &order)?;
            }
            break;
        }
    }

    Ok(Response::new()
        .add_attribute("method", "update_price")
        .add_attribute("escrow_address", escrow_address))
}

pub fn execute_process_order(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    order_id: String,
    action: OrderAction,
    _proof: Option<String>,
) -> Result<Response, ContractError> {
    let config = CONFIG.load(deps.storage)?;
    
    // Only authorized relayers can process orders
    if !config.authorized_relayers.contains(&info.sender) {
        return Err(ContractError::InvalidRelayer {});
    }

    let mut order = ORDERS.load(deps.storage, order_id.clone())?;

    match action {
        OrderAction::ConfirmSource { src_tx_hash, block_height } => {
            // Confirm source escrow on destination chain
            let confirm_msg = WasmMsg::Execute {
                contract_addr: order.escrow_address.to_string(),
                msg: to_binary(&destination_escrow::msg::ExecuteMsg::ConfirmSourceEscrow {
                    src_tx_hash,
                    block_height,
                })?,
                funds: vec![],
            };

            order.status = OrderStatus::Matched;
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id.clone(), &order)?;

            Ok(Response::new()
                .add_message(CosmosMsg::Wasm(confirm_msg))
                .add_attribute("method", "process_order")
                .add_attribute("action", "confirm_source")
                .add_attribute("order_id", order_id))
        }
        OrderAction::ExecuteSwap { secret } => {
            // Execute the swap by withdrawing from escrow
            let withdraw_msg = WasmMsg::Execute {
                contract_addr: order.escrow_address.to_string(),
                msg: to_binary(&source_escrow::msg::ExecuteMsg::Withdraw { secret })?,
                funds: vec![],
            };

            order.status = OrderStatus::Completed;
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id.clone(), &order)?;

            Ok(Response::new()
                .add_message(CosmosMsg::Wasm(withdraw_msg))
                .add_attribute("method", "process_order")
                .add_attribute("action", "execute_swap")
                .add_attribute("order_id", order_id))
        }
        OrderAction::CancelOrder => {
            // Cancel the order
            let cancel_msg = WasmMsg::Execute {
                contract_addr: order.escrow_address.to_string(),
                msg: to_binary(&source_escrow::msg::ExecuteMsg::Cancel {})?,
                funds: vec![],
            };

            order.status = OrderStatus::Cancelled;
            order.updated_at = env.block.time.seconds();
            ORDERS.save(deps.storage, order_id.clone(), &order)?;

            Ok(Response::new()
                .add_message(CosmosMsg::Wasm(cancel_msg))
                .add_attribute("method", "process_order")
                .add_attribute("action", "cancel_order")
                .add_attribute("order_id", order_id))
        }
    }
}

pub fn execute_add_relayer(
    deps: DepsMut,
    info: MessageInfo,
    relayer: String,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    
    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    let relayer_addr = deps.api.addr_validate(&relayer)?;
    
    if !config.authorized_relayers.contains(&relayer_addr) {
        config.authorized_relayers.push(relayer_addr.clone());
        CONFIG.save(deps.storage, &config)?;
    }

    Ok(Response::new()
        .add_attribute("method", "add_relayer")
        .add_attribute("relayer", relayer_addr))
}

pub fn execute_remove_relayer(
    deps: DepsMut,
    info: MessageInfo,
    relayer: String,
) -> Result<Response, ContractError> {
    let mut config = CONFIG.load(deps.storage)?;
    
    if info.sender != config.owner {
        return Err(ContractError::Unauthorized {});
    }

    let relayer_addr = deps.api.addr_validate(&relayer)?;
    config.authorized_relayers.retain(|addr| addr != &relayer_addr);
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "remove_relayer")
        .add_attribute("relayer", relayer_addr))
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

    let new_owner_addr = deps.api.addr_validate(&new_owner)?;
    config.owner = new_owner_addr.clone();
    CONFIG.save(deps.storage, &config)?;

    Ok(Response::new()
        .add_attribute("method", "update_owner")
        .add_attribute("new_owner", new_owner_addr))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Config {} => to_binary(&query_config(deps)?),
        QueryMsg::Order { order_id } => to_binary(&query_order(deps, order_id)?),
        QueryMsg::ActiveOrders { start_after, limit } => {
            to_binary(&query_active_orders(deps, start_after, limit)?)
        }
        QueryMsg::CurrentPrice { escrow_address } => {
            to_binary(&query_current_price(deps, env, escrow_address)?)
        }
        QueryMsg::IsAuthorizedRelayer { relayer } => {
            to_binary(&query_is_authorized_relayer(deps, relayer)?)
        }
    }
}

fn query_config(deps: Deps) -> StdResult<ConfigResponse> {
    let config = CONFIG.load(deps.storage)?;
    Ok(ConfigResponse {
        owner: config.owner,
        escrow_factory: config.escrow_factory,
        authorized_relayers: config.authorized_relayers,
    })
}

fn query_order(deps: Deps, order_id: String) -> StdResult<OrderResponse> {
    let order = ORDERS.load(deps.storage, order_id)?;
    Ok(OrderResponse {
        order_id: order.order_id,
        escrow_address: order.escrow_address,
        maker: order.maker,
        taker: order.taker,
        status: order.status,
        created_at: order.created_at,
        updated_at: order.updated_at,
        dutch_auction: order.dutch_auction,
        partial_fill: order.partial_fill,
    })
}

fn query_active_orders(
    deps: Deps,
    start_after: Option<String>,
    limit: Option<u32>,
) -> StdResult<OrderListResponse> {
    let limit = limit.unwrap_or(30).min(100) as usize;
    let start = start_after.as_ref().map(|s| cosmwasm_std::Bound::exclusive(s.as_str()));

    let orders: StdResult<Vec<_>> = ORDERS
        .range(deps.storage, start, None, cosmwasm_std::Order::Ascending)
        .take(limit)
        .map(|item| {
            item.map(|(_, order)| OrderResponse {
                order_id: order.order_id,
                escrow_address: order.escrow_address,
                maker: order.maker,
                taker: order.taker,
                status: order.status,
                created_at: order.created_at,
                updated_at: order.updated_at,
                dutch_auction: order.dutch_auction,
                partial_fill: order.partial_fill,
            })
        })
        .collect();

    Ok(OrderListResponse {
        orders: orders?,
    })
}

fn query_current_price(deps: Deps, env: Env, escrow_address: String) -> StdResult<PriceResponse> {
    let escrow_addr = deps.api.addr_validate(&escrow_address)?;
    
    // Find order with matching escrow address
    let orders: Vec<_> = ORDERS
        .range(deps.storage, None, None, cosmwasm_std::Order::Ascending)
        .collect::<StdResult<Vec<_>>>()?;

    for (_, order) in orders {
        if order.escrow_address == escrow_addr {
            if let Some(dutch_auction) = order.dutch_auction {
                let current_time = env.block.time.seconds();
                let time_elapsed = current_time - dutch_auction.start_time;
                
                return Ok(PriceResponse {
                    current_price: dutch_auction.current_price,
                    initial_price: Some(dutch_auction.initial_price),
                    minimum_price: Some(dutch_auction.minimum_price),
                    time_elapsed,
                });
            }
        }
    }

    Ok(PriceResponse {
        current_price: Uint128::zero(),
        initial_price: None,
        minimum_price: None,
        time_elapsed: 0,
    })
}

fn query_is_authorized_relayer(deps: Deps, relayer: String) -> StdResult<RelayerResponse> {
    let config = CONFIG.load(deps.storage)?;
    let relayer_addr = deps.api.addr_validate(&relayer)?;
    
    Ok(RelayerResponse {
        is_authorized: config.authorized_relayers.contains(&relayer_addr),
    })
}

