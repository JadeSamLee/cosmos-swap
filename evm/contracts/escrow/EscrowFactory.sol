// SPDX-License-Identifier: MIT
pragma solidity 0.8.23;

import {Ownable} from "openzeppelin-contracts/access/Ownable.sol";
import {Create2} from "openzeppelin-contracts/utils/Create2.sol";
import {Escrow} from "./Escrow.sol";

/**
 * @title EscrowFactory
 * @dev Factory contract for creating deterministic escrow contracts
 */
contract EscrowFactory is Ownable {
    
    // Events
    event EscrowCreated(
        address indexed escrow,
        address indexed maker,
        address indexed taker,
        bytes32 secretHash,
        uint256 timelock,
        bytes32 salt
    );

    event SourceEscrowCreated(
        address indexed escrow,
        address indexed maker,
        string dstChainId,
        bytes32 salt
    );

    event DestinationEscrowCreated(
        address indexed escrow,
        address indexed taker,
        address indexed maker,
        string srcChainId,
        bytes32 salt
    );

    // Errors
    error EscrowCreationFailed();
    error InvalidParameters();

    constructor(address initialOwner) Ownable(initialOwner) {}

    /**
     * @dev Create a new destination escrow contract
     */
    function createDestinationEscrow(
        address maker,
        address taker,
        bytes32 secretHash,
        uint256 timelock,
        string calldata srcChainId,
        string calldata srcEscrowAddress,
        uint256 expectedAmount,
        bool allowPartialFill,
        uint256 minimumFillAmount,
        bytes32 salt
    ) external returns (address escrow) {
        if (maker == address(0) || taker == address(0)) revert InvalidParameters();
        if (timelock <= block.timestamp) revert InvalidParameters();
        if (expectedAmount == 0) revert InvalidParameters();

        // Create escrow using CREATE2 for deterministic address
        bytes memory bytecode = abi.encodePacked(
            type(Escrow).creationCode,
            abi.encode(
                maker,
                taker,
                secretHash,
                timelock,
                srcChainId,
                srcEscrowAddress,
                expectedAmount,
                allowPartialFill,
                minimumFillAmount
            )
        );

        escrow = Create2.deploy(0, salt, bytecode);
        
        if (escrow == address(0)) revert EscrowCreationFailed();

        emit EscrowCreated(escrow, maker, taker, secretHash, timelock, salt);
        emit DestinationEscrowCreated(escrow, taker, maker, srcChainId, salt);

        return escrow;
    }

    /**
     * @dev Create a source escrow contract (for completeness, though typically done on non-EVM side)
     */
    function createSourceEscrow(
        address maker,
        address taker,
        bytes32 secretHash,
        uint256 timelock,
        string calldata dstChainId,
        string calldata dstAsset,
        uint256 dstAmount,
        bool allowPartialFill,
        uint256 minimumFillAmount,
        bytes32 salt
    ) external returns (address escrow) {
        if (maker == address(0)) revert InvalidParameters();
        if (timelock <= block.timestamp) revert InvalidParameters();
        if (dstAmount == 0) revert InvalidParameters();

        // For source escrow, we use a simplified version
        // In practice, this might be different or not used at all on EVM side
        bytes memory bytecode = abi.encodePacked(
            type(Escrow).creationCode,
            abi.encode(
                maker,
                taker,
                secretHash,
                timelock,
                dstChainId,
                "", // No source escrow address for source escrow
                dstAmount,
                allowPartialFill,
                minimumFillAmount
            )
        );

        escrow = Create2.deploy(0, salt, bytecode);
        
        if (escrow == address(0)) revert EscrowCreationFailed();

        emit EscrowCreated(escrow, maker, taker, secretHash, timelock, salt);
        emit SourceEscrowCreated(escrow, maker, dstChainId, salt);

        return escrow;
    }

    /**
     * @dev Compute the address of an escrow contract before deployment
     */
    function computeEscrowAddress(
        address maker,
        address taker,
        bytes32 secretHash,
        uint256 timelock,
        string calldata chainId,
        string calldata escrowAddress,
        uint256 expectedAmount,
        bool allowPartialFill,
        uint256 minimumFillAmount,
        bytes32 salt
    ) external view returns (address) {
        bytes memory bytecode = abi.encodePacked(
            type(Escrow).creationCode,
            abi.encode(
                maker,
                taker,
                secretHash,
                timelock,
                chainId,
                escrowAddress,
                expectedAmount,
                allowPartialFill,
                minimumFillAmount
            )
        );

        return Create2.computeAddress(salt, keccak256(bytecode), address(this));
    }

    /**
     * @dev Batch create multiple escrows
     */
    function batchCreateDestinationEscrows(
        address[] calldata makers,
        address[] calldata takers,
        bytes32[] calldata secretHashes,
        uint256[] calldata timelocks,
        string[] calldata srcChainIds,
        string[] calldata srcEscrowAddresses,
        uint256[] calldata expectedAmounts,
        bool[] calldata allowPartialFills,
        uint256[] calldata minimumFillAmounts,
        bytes32[] calldata salts
    ) external returns (address[] memory escrows) {
        uint256 length = makers.length;
        if (
            length != takers.length ||
            length != secretHashes.length ||
            length != timelocks.length ||
            length != srcChainIds.length ||
            length != srcEscrowAddresses.length ||
            length != expectedAmounts.length ||
            length != allowPartialFills.length ||
            length != minimumFillAmounts.length ||
            length != salts.length
        ) revert InvalidParameters();

        escrows = new address[](length);

        for (uint256 i = 0; i < length; i++) {
            escrows[i] = createDestinationEscrow(
                makers[i],
                takers[i],
                secretHashes[i],
                timelocks[i],
                srcChainIds[i],
                srcEscrowAddresses[i],
                expectedAmounts[i],
                allowPartialFills[i],
                minimumFillAmounts[i],
                salts[i]
            );
        }

        return escrows;
    }

    /**
     * @dev Emergency function to recover ETH sent to factory
     */
    function recoverETH() external onlyOwner {
        uint256 balance = address(this).balance;
        if (balance > 0) {
            (bool success, ) = owner().call{value: balance}("");
            require(success, "ETH recovery failed");
        }
    }

    /**
     * @dev Emergency function to recover ERC20 tokens sent to factory
     */
    function recoverToken(address token, uint256 amount) external onlyOwner {
        IERC20(token).transfer(owner(), amount);
    }

    // Allow factory to receive ETH (though it shouldn't normally)
    receive() external payable {}
}

