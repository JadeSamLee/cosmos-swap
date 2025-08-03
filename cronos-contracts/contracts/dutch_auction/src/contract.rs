use cosmwasm_std::{
    entry_point, to_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response, StdResult,
    Uint128, BankMsg, CosmosMsg, coin
};
use cw2::set_contract_version;

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{Auction, AUCTION};

const CONTRACT_NAME: &str = "dutch-auction-simple";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    let auction = Auction {
        seller: info.sender.clone(),
        start_price: msg.start_price,
        end_price: msg.end_price,
        start_time: env.block.time.seconds(),
        end_time: env.block.time.seconds() + msg.duration,
        current_bidder: None,
        current_bid: Uint128::zero(),
        is_active: true,
    };

    AUCTION.save(deps.storage, &auction)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("seller", info.sender)
        .add_attribute("start_price", msg.start_price)
        .add_attribute("end_price", msg.end_price))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Bid {} => execute_bid(deps, env, info),
        ExecuteMsg::EndAuction {} => execute_end_auction(deps, env, info),
    }
}

pub fn execute_bid(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
) -> Result<Response, ContractError> {
    let mut auction = AUCTION.load(deps.storage)?;

    if !auction.is_active {
        return Err(ContractError::AuctionNotActive {});
    }

    if env.block.time.seconds() > auction.end_time {
        return Err(ContractError::AuctionEnded {});
    }

    let current_price = auction.get_current_price(env.block.time.seconds());
    let bid_amount = info.funds.iter().find(|c| c.denom == "uatom")
        .map(|c| c.amount)
        .unwrap_or_else(Uint128::zero);

    if bid_amount < current_price {
        return Err(ContractError::BidTooLow {});
    }

    let mut response = Response::new();

    // Refund previous bidder
    if let Some(prev_bidder) = &auction.current_bidder {
        let refund_msg = BankMsg::Send {
            to_address: prev_bidder.to_string(),
            amount: vec![coin(auction.current_bid.u128(), "uatom")],
        };
        response = response.add_message(CosmosMsg::Bank(refund_msg));
    }

    auction.current_bidder = Some(info.sender.clone());
    auction.current_bid = bid_amount;
    AUCTION.save(deps.storage, &auction)?;

    Ok(response
        .add_attribute("method", "bid")
        .add_attribute("bidder", info.sender)
        .add_attribute("amount", bid_amount))
}

pub fn execute_end_auction(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
) -> Result<Response, ContractError> {
    let mut auction = AUCTION.load(deps.storage)?;

    if !auction.is_active {
        return Err(ContractError::AuctionNotActive {});
    }

    if env.block.time.seconds() < auction.end_time {
        return Err(ContractError::AuctionEnded {});
    }

    auction.is_active = false;
    AUCTION.save(deps.storage, &auction)?;

    let mut response = Response::new();

    if let Some(winner) = &auction.current_bidder {
        // Send funds to seller
        let payment_msg = BankMsg::Send {
            to_address: auction.seller.to_string(),
            amount: vec![coin(auction.current_bid.u128(), "uatom")],
        };
        response = response.add_message(CosmosMsg::Bank(payment_msg));
    }

    Ok(response
        .add_attribute("method", "end_auction")
        .add_attribute("winner", auction.current_bidder.unwrap_or_default())
        .add_attribute("winning_bid", auction.current_bid))
}

#[entry_point]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetAuction {} => to_binary(&AUCTION.load(deps.storage)?),
        QueryMsg::GetCurrentPrice {} => {
            let auction = AUCTION.load(deps.storage)?;
            let current_price = auction.get_current_price(env.block.time.seconds());
            to_binary(&current_price)
        }
    }
}



