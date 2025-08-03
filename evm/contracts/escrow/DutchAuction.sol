// SPDX-License-Identifier: MIT
pragma solidity ^0.8.23;

contract DutchAuction {
    struct Order {
        address maker;
        uint256 startPrice;
        uint256 endPrice;
        uint256 duration;
        uint256 startTime;
        address resolver;
        bool claimed;
        bool active;
    }

    uint256 public nextOrderId;
    mapping(uint256 => Order) public orders;

    event OrderCreated(uint256 indexed orderId, address indexed maker, uint256 startPrice, uint256 endPrice, uint256 duration);
    event BidPlaced(uint256 indexed orderId, address indexed resolver, uint256 price);
    event OrderClaimed(uint256 indexed orderId, address indexed resolver);

    modifier onlyMaker(uint256 orderId) {
        require(msg.sender == orders[orderId].maker, "Not order maker");
        _;
    }

    modifier onlyResolver(uint256 orderId) {
        require(msg.sender == orders[orderId].resolver, "Not resolver");
        _;
    }

    function createOrder(uint256 startPrice, uint256 endPrice, uint256 duration) external returns (uint256) {
        require(startPrice > endPrice, "Start price must be greater than end price");
        require(duration > 0, "Duration must be > 0");

        uint256 orderId = nextOrderId++;
        orders[orderId] = Order({
            maker: msg.sender,
            startPrice: startPrice,
            endPrice: endPrice,
            duration: duration,
            startTime: block.timestamp,
            resolver: address(0),
            claimed: false,
            active: true
        });

        emit OrderCreated(orderId, msg.sender, startPrice, endPrice, duration);
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

    function bid(uint256 orderId) external {
        Order storage order = orders[orderId];
        require(order.active, "Order not active");
        require(order.resolver == address(0), "Order already won");

        uint256 currentPrice = getCurrentPrice(orderId);
        order.resolver = msg.sender;
        order.active = false;

        emit BidPlaced(orderId, msg.sender, currentPrice);
    }

    function claimOrder(uint256 orderId) external onlyResolver(orderId) {
        Order storage order = orders[orderId];
        require(!order.claimed, "Order already claimed");

        order.claimed = true;

        emit OrderClaimed(orderId, msg.sender);
    }
}
