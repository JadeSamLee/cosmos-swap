// SPDX-License-Identifier: MIT
pragma solidity 0.8.23;
 
 import "@openzeppelin/contracts/utils/cryptography/Keccak256.sol";
 import "@openzeppelin/contracts/access/Ownable.sol";
 import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
 /**
 * @title DutchAuctionManager
 * @notice Manages Dutch auction pricing for cross-chain swaps
 * @dev Implements decreasing price mechanism over time
 */
 contract DutchAuctionManager is Ownable, ReentrancyGuard {
    
    struct AuctionParams {
        uint256 startPrice;
        uint256 endPrice;
        uint256 startTime;
        uint256 duration;
        uint256 priceDecayRate;
    }
    
    struct Auction {
        bytes32 orderId;
        AuctionParams params;
        bool isActive;
        address winner;
        uint256 winningPrice;
        uint256 endTime;
    }
    
    mapping(bytes32 => Auction) public auctions;
    mapping(address => bool) public authorizedBidders;
    
    event AuctionStarted(
        bytes32 indexed orderId,
        uint256 startPrice,
        uint256 endPrice,
        uint256 duration
    );
    
    event BidPlaced(
        bytes32 indexed orderId,
        address indexed bidder,
        uint256 price,
        uint256 timestamp
    );
    
    event AuctionEnded(
        bytes32 indexed orderId,
        address indexed winner,
        uint256 winningPrice
    );
    modifier onlyAuthorizedBidder() {
        require(authorizedBidders[msg.sender], "DutchAuctionManager: Unauthorized bidder");
        _;
    }
    
    modifier activeAuction(bytes32 orderId) {
        require(auctions[orderId].isActive, "DutchAuctionManager: Auction not active");
        require(block.timestamp <= auctions[orderId].endTime, 
"DutchAuctionManager: Auction expired");
        _;
    }
    /**
     * @notice Start a Dutch auction for a swap order
     * @param orderId Unique identifier for the swap order
     * @param startPrice Starting price for the auction
     * @param endPrice Minimum price for the auction
     * @param duration Duration of the auction in seconds
     */
    function startAuction(
        bytes32 orderId,
        uint256 startPrice,
        uint256 endPrice,
        uint256 duration
    ) external onlyOwner {
        require(auctions[orderId].orderId == bytes32(0), 
"DutchAuctionManager: Auction already exists");
        require(startPrice > endPrice, "DutchAuctionManager: Invalid price range");
        require(duration > 0, "DutchAuctionManager: Invalid duration");
        uint256 priceDecayRate = (startPrice - endPrice) / duration;
        
        auctions[orderId] = Auction({
            orderId: orderId,
            params: AuctionParams({
                startPrice: startPrice,
                endPrice: endPrice,
                startTime: block.timestamp,
                duration: duration,
                priceDecayRate: priceDecayRate
            }),
            isActive: true,
            winner: address(0),
            winningPrice: 0,
            endTime: block.timestamp + duration
        });
        emit AuctionStarted(orderId, startPrice, endPrice, duration);
    }
    /**
     * @notice Place a bid in the Dutch auction
     * @param orderId The auction identifier
     */
    function placeBid(bytes32 orderId) external nonReentrant 
onlyAuthorizedBidder activeAuction(orderId) {
        Auction storage auction = auctions[orderId];
        uint256 currentPrice = getCurrentPrice(orderId);
        
        // End the auction with this bidder as the winner
        auction.isActive = false;
        auction.winner = msg.sender;
        auction.winningPrice = currentPrice;
        emit BidPlaced(orderId, msg.sender, currentPrice, block.timestamp);
        emit AuctionEnded(orderId, msg.sender, currentPrice);
    }
    /**
     * @notice Get the current price for an auction
     * @param orderId The auction identifier
     * @return currentPrice The current price based on time elapsed
     */
    function getCurrentPrice(bytes32 orderId) public view returns (uint256 
currentPrice) {
        Auction storage auction = auctions[orderId];
        require(auction.orderId != bytes32(0), "DutchAuctionManager: Auction does not exist");
        
        if (!auction.isActive) {
            return auction.winningPrice;
        }
        
        uint256 timeElapsed = block.timestamp - auction.params.startTime;
        
        if (timeElapsed >= auction.params.duration) {
            return auction.params.endPrice;
        }
        
        uint256 priceDecrease = auction.params.priceDecayRate * timeElapsed;
        return auction.params.startPrice - priceDecrease;
    }
    /**
     * @notice Authorize a bidder to participate in auctions
     * @param bidder Address of the bidder to authorize
     */
    function authorizeBidder(address bidder) external onlyOwner {
        require(bidder != address(0), "DutchAuctionManager: Invalid bidder address");
        authorizedBidders[bidder] = true;
    }
    /**
     * @notice Revoke authorization for a bidder
     * @param bidder Address of the bidder to revoke
     */
    function revokeBidder(address bidder) external onlyOwner {
        authorizedBidders[bidder] = false;
    }
    /**
     * @notice Get auction details
     * @param orderId The auction identifier
     * @return auction The auction details
     */
    function getAuction(bytes32 orderId) external view returns (Auction 
memory auction) {
        return auctions[orderId];
    }
    /**
     * @notice Check if an auction is active
     * @param orderId The auction identifier
     * @return active True if the auction is active
     */
    function isAuctionActive(bytes32 orderId) external view returns (bool 
active) {
        Auction storage auction = auctions[orderId];
        return auction.isActive && block.timestamp <= auction.endTime;
    }
 }