// SPDX-License-Identifier: MIT
pragma solidity 0.8.23;

import {Ownable} from "openzeppelin-contracts/access/Ownable.sol";
import {IERC20} from "openzeppelin-contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "openzeppelin-contracts/token/ERC20/utils/SafeERC20.sol";

import {IOrderMixin} from "limit-order-protocol/interfaces/IOrderMixin.sol";
import {TakerTraits} from "limit-order-protocol/libraries/TakerTraitsLib.sol";

import {EscrowFactory} from "./EscrowFactory.sol";
import {Escrow} from "./Escrow.sol";
import {IBCHandler} from "./IBCHandler.sol";

/**
 * @title Resolver
 * @dev Enhanced resolver contract for cross-chain swaps with IBC integration
 * Based on 1inch cross-chain resolver but adapted for Cronos-Ethereum bridge
 */
contract Resolver is Ownable {
    using SafeERC20 for IERC20;

    struct Immutables {
        address maker;
        address taker;
        bytes32 secretHash;
        uint256 timelock;
        string srcChainId;
        string srcEscrowAddress;
        uint256 expectedAmount;
        bool allowPartialFill;
        uint256 minimumFillAmount;
        uint256 safetyDeposit;
    }

    EscrowFactory public immutable ESCROW_FACTORY;
    IOrderMixin public immutable LIMIT_ORDER_PROTOCOL;
    IBCHandler public immutable IBC_HANDLER;

    // Events
    event SourceEscrowDeployed(
        address indexed escrow,
        address indexed maker,
        bytes32 indexed secretHash,
        string dstChainId
    );

    event DestinationEscrowDeployed(
        address indexed escrow,
        address indexed taker,
        address indexed maker,
        string srcChainId
    );

    event EscrowWithdrawn(
        address indexed escrow,
        address indexed recipient,
        uint256 amount,
        bytes32 secret
    );

    event EscrowCancelled(
        address indexed escrow,
        address indexed canceller,
        uint256 returnedAmount
    );

    event IBCTransferInitiated(
        string indexed dstChainId,
        address indexed recipient,
        uint256 amount,
        string denom
    );

    // Errors
    error InvalidLength();
    error LengthMismatch();
    error InsufficientSafetyDeposit();
    error EscrowDeploymentFailed();
    error UnauthorizedCaller();

    constructor(
        EscrowFactory escrowFactory,
        IOrderMixin limitOrderProtocol,
        IBCHandler ibcHandler,
        address initialOwner
    ) Ownable(initialOwner) {
        ESCROW_FACTORY = escrowFactory;
        LIMIT_ORDER_PROTOCOL = limitOrderProtocol;
        IBC_HANDLER = ibcHandler;
    }

    receive() external payable {} // Allow receiving ETH for safety deposits

    /**
     * @dev Deploy source escrow and fill limit order atomically
     * Adapted from 1inch resolver's deploySrc function
     */
    function deploySrc(
        Immutables calldata immutables,
        IOrderMixin.Order calldata order,
        bytes32 r,
        bytes32 vs,
        uint256 amount,
        TakerTraits takerTraits,
        bytes calldata args
    ) external payable onlyOwner {
        // Validate safety deposit
        if (msg.value < immutables.safetyDeposit) revert InsufficientSafetyDeposit();

        // Generate salt for deterministic address
        bytes32 salt = keccak256(abi.encodePacked(
            immutables.maker,
            immutables.secretHash,
            block.timestamp
        ));

        // Deploy source escrow
        address escrow = ESCROW_FACTORY.createSourceEscrow(
            immutables.maker,
            immutables.taker,
            immutables.secretHash,
            immutables.timelock,
            immutables.srcChainId,
            "", // dstAsset - to be filled
            immutables.expectedAmount,
            immutables.allowPartialFill,
            immutables.minimumFillAmount,
            salt
        );

        // Send safety deposit to escrow
        if (immutables.safetyDeposit > 0) {
            (bool success, ) = escrow.call{value: immutables.safetyDeposit}("");
            if (!success) revert EscrowDeploymentFailed();
        }

        // Fill limit order with escrow as target
        // Set _ARGS_HAS_TARGET flag (bit 251)
        takerTraits = TakerTraits.wrap(TakerTraits.unwrap(takerTraits) | uint256(1 << 251));
        bytes memory argsWithTarget = abi.encodePacked(escrow, args);
        
        LIMIT_ORDER_PROTOCOL.fillOrderArgs(order, r, vs, amount, takerTraits, argsWithTarget);

        emit SourceEscrowDeployed(escrow, immutables.maker, immutables.secretHash, immutables.srcChainId);
    }

    /**
     * @dev Deploy destination escrow
     * Adapted from 1inch resolver's deployDst function
     */
    function deployDst(
        Immutables calldata immutables,
        uint256 srcCancellationTimestamp
    ) external payable onlyOwner {
        // Generate salt for deterministic address
        bytes32 salt = keccak256(abi.encodePacked(
            immutables.taker,
            immutables.maker,
            immutables.secretHash,
            block.timestamp
        ));

        // Deploy destination escrow
        address escrow = ESCROW_FACTORY.createDestinationEscrow(
            immutables.maker,
            immutables.taker,
            immutables.secretHash,
            immutables.timelock,
            immutables.srcChainId,
            immutables.srcEscrowAddress,
            immutables.expectedAmount,
            immutables.allowPartialFill,
            immutables.minimumFillAmount,
            salt
        );

        // Forward any ETH sent with the call to the escrow
        if (msg.value > 0) {
            (bool success, ) = escrow.call{value: msg.value}("");
            if (!success) revert EscrowDeploymentFailed();
        }

        emit DestinationEscrowDeployed(escrow, immutables.taker, immutables.maker, immutables.srcChainId);
    }

    /**
     * @dev Withdraw from escrow using secret
     */
    function withdraw(
        Escrow escrow,
        bytes32 secret,
        Immutables calldata immutables
    ) external {
        // Verify caller is authorized (maker or owner)
        if (msg.sender != immutables.maker && msg.sender != owner()) {
            revert UnauthorizedCaller();
        }

        escrow.withdraw(secret);

        emit EscrowWithdrawn(
            address(escrow),
            immutables.maker,
            immutables.expectedAmount,
            secret
        );
    }

    /**
     * @dev Partial withdraw from escrow
     */
    function partialWithdraw(
        Escrow escrow,
        bytes32 secret,
        uint256 amount,
        Immutables calldata immutables
    ) external {
        // Verify caller is authorized (maker or owner)
        if (msg.sender != immutables.maker && msg.sender != owner()) {
            revert UnauthorizedCaller();
        }

        escrow.partialWithdraw(secret, amount);

        emit EscrowWithdrawn(
            address(escrow),
            immutables.maker,
            amount,
            secret
        );
    }

    /**
     * @dev Cancel escrow
     */
    function cancel(
        Escrow escrow,
        Immutables calldata immutables
    ) external {
        // Verify caller is authorized (taker or owner)
        if (msg.sender != immutables.taker && msg.sender != owner()) {
            revert UnauthorizedCaller();
        }

        escrow.cancel();

        emit EscrowCancelled(
            address(escrow),
            immutables.taker,
            immutables.expectedAmount
        );
    }

    /**
     * @dev Initiate IBC transfer to destination chain
     */
    function initiateIBCTransfer(
        string calldata dstChainId,
        address recipient,
        uint256 amount,
        string calldata denom,
        string calldata channelId,
        uint64 timeoutTimestamp
    ) external onlyOwner {
        // Transfer tokens to IBC handler
        if (keccak256(bytes(denom)) == keccak256(bytes("ETH"))) {
            // Transfer ETH
            IBC_HANDLER.sendPacket{value: amount}(
                channelId,
                abi.encode(recipient, amount, denom),
                timeoutTimestamp
            );
        } else {
            // Transfer ERC20 token
            IERC20(recipient).safeTransferFrom(msg.sender, address(IBC_HANDLER), amount);
            IBC_HANDLER.sendPacket(
                channelId,
                abi.encode(recipient, amount, denom),
                timeoutTimestamp
            );
        }

        emit IBCTransferInitiated(dstChainId, recipient, amount, denom);
    }

    /**
     * @dev Confirm source escrow on destination chain
     */
    function confirmSourceEscrow(
        Escrow escrow,
        string calldata srcTxHash,
        uint256 srcBlockHeight
    ) external onlyOwner {
        escrow.confirmSourceEscrow(srcTxHash, srcBlockHeight);
    }

    /**
     * @dev Batch operations for multiple escrows
     */
    function batchWithdraw(
        Escrow[] calldata escrows,
        bytes32[] calldata secrets,
        Immutables[] calldata immutables
    ) external {
        uint256 length = escrows.length;
        if (length != secrets.length || length != immutables.length) {
            revert LengthMismatch();
        }

        for (uint256 i = 0; i < length; i++) {
            withdraw(escrows[i], secrets[i], immutables[i]);
        }
    }

    /**
     * @dev Batch cancel operations
     */
    function batchCancel(
        Escrow[] calldata escrows,
        Immutables[] calldata immutables
    ) external {
        uint256 length = escrows.length;
        if (length != immutables.length) revert LengthMismatch();

        for (uint256 i = 0; i < length; i++) {
            cancel(escrows[i], immutables[i]);
        }
    }

    /**
     * @dev Execute arbitrary calls (for flexibility and upgrades)
     * Similar to 1inch resolver's arbitraryCalls
     */
    function arbitraryCalls(
        address[] calldata targets,
        bytes[] calldata arguments
    ) external onlyOwner {
        uint256 length = targets.length;
        if (targets.length != arguments.length) revert LengthMismatch();
        
        for (uint256 i = 0; i < length; ++i) {
            (bool success, bytes memory result) = targets[i].call(arguments[i]);
            if (!success) {
                // Forward revert reason
                if (result.length > 0) {
                    assembly {
                        revert(add(32, result), mload(result))
                    }
                } else {
                    revert("Arbitrary call failed");
                }
            }
        }
    }

    /**
     * @dev Compute escrow address before deployment
     */
    function computeEscrowAddress(
        Immutables calldata immutables,
        bytes32 salt
    ) external view returns (address) {
        return ESCROW_FACTORY.computeEscrowAddress(
            immutables.maker,
            immutables.taker,
            immutables.secretHash,
            immutables.timelock,
            immutables.srcChainId,
            immutables.srcEscrowAddress,
            immutables.expectedAmount,
            immutables.allowPartialFill,
            immutables.minimumFillAmount,
            salt
        );
    }

    /**
     * @dev Emergency recovery function
     */
    function emergencyRecovery(address token, uint256 amount) external onlyOwner {
        if (token == address(0)) {
            // Recover ETH
            (bool success, ) = owner().call{value: amount}("");
            require(success, "ETH recovery failed");
        } else {
            // Recover ERC20 token
            IERC20(token).safeTransfer(owner(), amount);
        }
    }

    /**
     * @dev Update escrow factory (for upgrades)
     */
    function updateEscrowFactory(EscrowFactory newFactory) external onlyOwner {
        // This would require more sophisticated upgrade logic in production
        // For now, just emit an event
        emit DestinationEscrowDeployed(address(newFactory), address(0), address(0), "");
    }
}

