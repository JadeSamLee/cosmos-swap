// SPDX-License-Identifier: MIT
pragma solidity 0.8.23;

import {IERC20} from "openzeppelin-contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "openzeppelin-contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "openzeppelin-contracts/security/ReentrancyGuard.sol";

/**
 * @title Escrow
 * @dev Escrow contract for cross-chain atomic swaps with support for partial fills and Dutch auctions
 */
contract Escrow is ReentrancyGuard {
    using SafeERC20 for IERC20;

    enum EscrowStatus {
        Active,
        Withdrawn,
        Cancelled
    }

    struct EscrowInfo {
        address maker;
        address taker;
        bytes32 secretHash;
        uint256 timelock;
        string srcChainId;
        string srcEscrowAddress;
        uint256 expectedAmount;
        uint256 depositedAmount;
        address tokenAddress;
        EscrowStatus status;
        uint256 createdAt;
        bool srcConfirmed;
        string srcTxHash;
        uint256 srcBlockHeight;
        // Partial fill support
        bool allowPartialFill;
        uint256 filledAmount;
        uint256 remainingAmount;
        uint256 minimumFillAmount;
    }

    EscrowInfo public escrowInfo;
    
    // Events
    event EscrowCreated(
        address indexed maker,
        address indexed taker,
        bytes32 indexed secretHash,
        uint256 timelock,
        uint256 expectedAmount
    );
    
    event Deposited(
        address indexed depositor,
        uint256 amount,
        address tokenAddress
    );
    
    event Withdrawn(
        address indexed recipient,
        uint256 amount,
        bytes32 secret
    );
    
    event PartialWithdrawn(
        address indexed recipient,
        uint256 amount,
        uint256 remainingAmount,
        bytes32 secret
    );
    
    event Cancelled(
        address indexed canceller,
        uint256 returnedAmount
    );
    
    event SourceEscrowConfirmed(
        string srcTxHash,
        uint256 srcBlockHeight
    );

    // Errors
    error Unauthorized();
    error InvalidSecret();
    error AlreadyWithdrawn();
    error AlreadyCancelled();
    error TimelockNotExpired();
    error InsufficientFunds();
    error InvalidAmount();
    error SourceEscrowNotConfirmed();
    error PartialFillNotAllowed();
    error InvalidPartialFillAmount();
    error OrderFullyFilled();

    modifier onlyMaker() {
        if (msg.sender != escrowInfo.maker) revert Unauthorized();
        _;
    }

    modifier onlyTaker() {
        if (msg.sender != escrowInfo.taker) revert Unauthorized();
        _;
    }

    modifier onlyActive() {
        if (escrowInfo.status != EscrowStatus.Active) {
            if (escrowInfo.status == EscrowStatus.Withdrawn) revert AlreadyWithdrawn();
            if (escrowInfo.status == EscrowStatus.Cancelled) revert AlreadyCancelled();
        }
        _;
    }

    constructor(
        address _maker,
        address _taker,
        bytes32 _secretHash,
        uint256 _timelock,
        string memory _srcChainId,
        string memory _srcEscrowAddress,
        uint256 _expectedAmount,
        bool _allowPartialFill,
        uint256 _minimumFillAmount
    ) {
        escrowInfo = EscrowInfo({
            maker: _maker,
            taker: _taker,
            secretHash: _secretHash,
            timelock: _timelock,
            srcChainId: _srcChainId,
            srcEscrowAddress: _srcEscrowAddress,
            expectedAmount: _expectedAmount,
            depositedAmount: 0,
            tokenAddress: address(0),
            status: EscrowStatus.Active,
            createdAt: block.timestamp,
            srcConfirmed: false,
            srcTxHash: "",
            srcBlockHeight: 0,
            allowPartialFill: _allowPartialFill,
            filledAmount: 0,
            remainingAmount: 0,
            minimumFillAmount: _minimumFillAmount
        });

        emit EscrowCreated(_maker, _taker, _secretHash, _timelock, _expectedAmount);
    }

    /**
     * @dev Deposit ETH to the escrow
     */
    function depositETH() external payable onlyTaker onlyActive {
        if (msg.value != escrowInfo.expectedAmount) revert InvalidAmount();
        if (escrowInfo.depositedAmount > 0) revert AlreadyWithdrawn(); // Already deposited

        escrowInfo.depositedAmount = msg.value;
        escrowInfo.remainingAmount = msg.value;
        escrowInfo.tokenAddress = address(0); // ETH

        emit Deposited(msg.sender, msg.value, address(0));
    }

    /**
     * @dev Deposit ERC20 tokens to the escrow
     */
    function depositToken(address tokenAddress, uint256 amount) external onlyTaker onlyActive {
        if (amount != escrowInfo.expectedAmount) revert InvalidAmount();
        if (escrowInfo.depositedAmount > 0) revert AlreadyWithdrawn(); // Already deposited

        IERC20(tokenAddress).safeTransferFrom(msg.sender, address(this), amount);

        escrowInfo.depositedAmount = amount;
        escrowInfo.remainingAmount = amount;
        escrowInfo.tokenAddress = tokenAddress;

        emit Deposited(msg.sender, amount, tokenAddress);
    }

    /**
     * @dev Withdraw funds using the secret (for maker)
     */
    function withdraw(bytes32 secret) external onlyMaker onlyActive nonReentrant {
        if (!escrowInfo.srcConfirmed) revert SourceEscrowNotConfirmed();
        if (keccak256(abi.encodePacked(secret)) != escrowInfo.secretHash) revert InvalidSecret();

        uint256 withdrawAmount = escrowInfo.allowPartialFill ? 
            escrowInfo.remainingAmount : escrowInfo.depositedAmount;

        escrowInfo.status = EscrowStatus.Withdrawn;
        
        _transferFunds(escrowInfo.maker, withdrawAmount);

        emit Withdrawn(escrowInfo.maker, withdrawAmount, secret);
    }

    /**
     * @dev Partial withdraw for partial fills (for maker)
     */
    function partialWithdraw(bytes32 secret, uint256 amount) external onlyMaker onlyActive nonReentrant {
        if (!escrowInfo.allowPartialFill) revert PartialFillNotAllowed();
        if (!escrowInfo.srcConfirmed) revert SourceEscrowNotConfirmed();
        if (keccak256(abi.encodePacked(secret)) != escrowInfo.secretHash) revert InvalidSecret();
        if (amount > escrowInfo.remainingAmount) revert InsufficientFunds();
        if (amount < escrowInfo.minimumFillAmount && escrowInfo.minimumFillAmount > 0) {
            revert InvalidPartialFillAmount();
        }

        escrowInfo.filledAmount += amount;
        escrowInfo.remainingAmount -= amount;

        if (escrowInfo.remainingAmount == 0) {
            escrowInfo.status = EscrowStatus.Withdrawn;
        }

        _transferFunds(escrowInfo.maker, amount);

        emit PartialWithdrawn(escrowInfo.maker, amount, escrowInfo.remainingAmount, secret);
    }

    /**
     * @dev Cancel the escrow after timelock expires (for taker)
     */
    function cancel() external onlyTaker onlyActive nonReentrant {
        if (block.timestamp < escrowInfo.timelock) revert TimelockNotExpired();

        uint256 returnAmount = escrowInfo.remainingAmount > 0 ? 
            escrowInfo.remainingAmount : escrowInfo.depositedAmount;

        escrowInfo.status = EscrowStatus.Cancelled;

        _transferFunds(escrowInfo.taker, returnAmount);

        emit Cancelled(escrowInfo.taker, returnAmount);
    }

    /**
     * @dev Confirm source escrow (called by authorized relayer)
     */
    function confirmSourceEscrow(
        string calldata srcTxHash,
        uint256 srcBlockHeight
    ) external {
        // TODO: Add proper authorization check for relayer
        // For now, anyone can confirm - in production, restrict to authorized relayer
        
        escrowInfo.srcConfirmed = true;
        escrowInfo.srcTxHash = srcTxHash;
        escrowInfo.srcBlockHeight = srcBlockHeight;

        emit SourceEscrowConfirmed(srcTxHash, srcBlockHeight);
    }

    /**
     * @dev Internal function to transfer funds
     */
    function _transferFunds(address recipient, uint256 amount) internal {
        if (escrowInfo.tokenAddress == address(0)) {
            // Transfer ETH
            (bool success, ) = recipient.call{value: amount}("");
            if (!success) revert InsufficientFunds();
        } else {
            // Transfer ERC20 token
            IERC20(escrowInfo.tokenAddress).safeTransfer(recipient, amount);
        }
    }

    /**
     * @dev Get escrow information
     */
    function getEscrowInfo() external view returns (EscrowInfo memory) {
        return escrowInfo;
    }

    /**
     * @dev Get fill status for partial fills
     */
    function getFillStatus() external view returns (
        uint256 totalAmount,
        uint256 filledAmount,
        uint256 remainingAmount,
        bool isFullyFilled,
        bool allowPartialFill
    ) {
        return (
            escrowInfo.depositedAmount,
            escrowInfo.filledAmount,
            escrowInfo.remainingAmount,
            escrowInfo.remainingAmount == 0,
            escrowInfo.allowPartialFill
        );
    }

    /**
     * @dev Check if secret is valid (view function for testing)
     */
    function isValidSecret(bytes32 secret) external view returns (bool) {
        return keccak256(abi.encodePacked(secret)) == escrowInfo.secretHash;
    }

    /**
     * @dev Emergency function to recover stuck funds (only after significant time delay)
     */
    function emergencyRecovery() external {
        // Only allow recovery after 30 days past timelock
        if (block.timestamp < escrowInfo.timelock + 30 days) revert TimelockNotExpired();
        
        uint256 balance;
        if (escrowInfo.tokenAddress == address(0)) {
            balance = address(this).balance;
            if (balance > 0) {
                (bool success, ) = escrowInfo.taker.call{value: balance}("");
                require(success, "ETH transfer failed");
            }
        } else {
            balance = IERC20(escrowInfo.tokenAddress).balanceOf(address(this));
            if (balance > 0) {
                IERC20(escrowInfo.tokenAddress).safeTransfer(escrowInfo.taker, balance);
            }
        }
    }

    // Allow contract to receive ETH
    receive() external payable {}
}

