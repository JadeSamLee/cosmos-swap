// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/cryptography/MerkleProof.sol";

contract DutchAuctionPartial is ReentrancyGuard {
    struct Order {
        address maker;
        IERC20 token;
        uint256 tokenAmount;
        uint256 startPrice;
        uint256 endPrice;
        uint256 duration;
        uint256 startTime;
        bytes32 merkleRoot; // Merkle root of secrets for partial fills
        mapping(bytes32 => bool) usedSecrets; // Track used secrets
        bool active;
        address winner;
        bool claimed;
    }

    uint256 public nextOrderId;
    mapping(uint256 => Order) private orders;

    event OrderCreated(uint256 indexed orderId, address indexed maker, address token, uint256 tokenAmount, uint256 startPrice, uint256 endPrice, uint256 duration, bytes32 merkleRoot);
    event PartialFill(uint256 indexed orderId, address indexed resolver, uint256 price, bytes32 secret);
    event OrderClaimed(uint256 indexed orderId, address indexed resolver);

    modifier onlyMaker(uint256 orderId) {
        require(msg.sender == orders[orderId].maker, "Not order maker");
        _;
    }

    modifier onlyWinner(uint256 orderId) {
        require(msg.sender == orders[orderId].winner, "Not winner");
        _;
    }

    function createOrder(
        IERC20 token,
        uint256 tokenAmount,
        uint256 startPrice,
        uint256 endPrice,
        uint256 duration,
        bytes32 merkleRoot
    ) external nonReentrant returns (uint256) {
        require(startPrice > endPrice, "Start price must be greater than end price");
        require(duration > 0, "Duration must be > 0");
        require(tokenAmount > 0, "Token amount must be > 0");

        uint256 orderId = nextOrderId++;
        Order storage order = orders[orderId];
        order.maker = msg.sender;
        order.token = token;
        order.tokenAmount = tokenAmount;
        order.startPrice = startPrice;
        order.endPrice = endPrice;
        order.duration = duration;
        order.startTime = block.timestamp;
        order.merkleRoot = merkleRoot;
        order.active = true;
        order.claimed = false;

        // Transfer tokens from maker to this contract (escrow)
        require(token.transferFrom(msg.sender, address(this), tokenAmount), "Token transfer failed");

        emit OrderCreated(orderId, msg.sender, address(token), tokenAmount, startPrice, endPrice, duration, merkleRoot);
        return orderId;
    }

    function getCurrentPrice(uint256 orderId) public view returns (uint256) {
        Order storage order = orders[orderId];
        require(order.active, "Order not active");

        uint256 elapsed = block.timestamp - order.startTime;
        if (elapsed >= order.duration) {
            return order.endPrice;
        } else {
            uint256 priceDiff = order.startPrice - order.endPrice;
            uint256 priceDecay = (priceDiff * elapsed) / order.duration;
            return order.startPrice - priceDecay;
        }
    }

    function partialFill(
        uint256 orderId,
        bytes32 secret,
        bytes32[] calldata merkleProof
    ) external nonReentrant {
        Order storage order = orders[orderId];
        require(order.active, "Order not active");
        require(!order.usedSecrets[secret], "Secret already used");

        // Verify the secret is part of the Merkle tree
        bytes32 leaf = keccak256(abi.encodePacked(secret));
        require(MerkleProof.verify(merkleProof, order.merkleRoot, leaf), "Invalid Merkle proof");

        uint256 currentPrice = getCurrentPrice(orderId);

        // Mark secret as used
        order.usedSecrets[secret] = true;

        // Set resolver as winner for this partial fill (could be extended for multiple winners)
        order.winner = msg.sender;

        emit PartialFill(orderId, msg.sender, currentPrice, secret);
    }

    function claimOrder(uint256 orderId) external nonReentrant onlyWinner(orderId) {
        Order storage order = orders[orderId];
        require(!order.claimed, "Order already claimed");

        // Transfer tokens between Maker and Resolver
        uint256 currentPrice = getCurrentPrice(orderId);
        uint256 makerAmount = order.tokenAmount;
        uint256 resolverAmount = (makerAmount * currentPrice) / order.startPrice;

        // Transfer Resolver's tokens to Maker
        require(order.token.transferFrom(msg.sender, order.maker, resolverAmount), "Resolver token transfer failed");

        // Transfer Maker's tokens to Resolver
        require(order.token.transfer(msg.sender, makerAmount), "Maker token transfer failed");

        order.claimed = true;
        order.active = false;

        emit OrderClaimed(orderId, msg.sender);
    }
}