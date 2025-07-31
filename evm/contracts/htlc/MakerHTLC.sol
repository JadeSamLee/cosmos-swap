// SPDX-License-Identifier: MIT
pragma solidity ^0.8.23;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "./Timelock.sol";

/**
 * @title MakerHTLC
 * @dev HTLC contract for the maker side of the atomic swap.
 */
contract MakerHTLC is Timelock {
    event NewContract(
        bytes32 indexed contractId,
        address indexed sender,
        address indexed recipient,
        address token,
        uint256 amount,
        bytes32 hashlock,
        uint256 timelock
    );

    event Refunded(bytes32 indexed contractId);

    struct LockContract {
        address sender;
        address recipient;
        address token; // ERC20 token address or address(0) for native CRO
        uint256 amount;
        bytes32 hashlock;
        uint256 timelock;
        bool refunded;
        bytes32 secret;
    }

    mapping(bytes32 => LockContract) public contracts;

    /**
     * @dev Creates a new HTLC contract.
     */
    function newContract(
        address recipient,
        address token,
        uint256 amount,
        bytes32 hashlock,
        uint256 timelock
    ) public payable returns (bytes32 contractId) {
        require(recipient != address(0), "Recipient cannot be zero address");
        require(timelock > block.timestamp, "Timelock must be in the future");
        require(amount > 0, "Amount must be greater than 0");

        contractId = keccak256(
            abi.encodePacked(msg.sender, recipient, token, amount, hashlock, timelock)
        );

        require(contracts[contractId].sender == address(0), "Contract already exists");

        if (token == address(0)) {
            require(msg.value == amount, "Native currency amount sent does not match");
        } else {
            require(msg.value == 0, "Do not send native currency when locking an ERC20 token");
            bool success = IERC20(token).transferFrom(msg.sender, address(this), amount);
            require(success, "ERC20 transferFrom failed");
        }

        contracts[contractId] = LockContract({
            sender: msg.sender,
            recipient: recipient,
            token: token,
            amount: amount,
            hashlock: hashlock,
            timelock: timelock,
            refunded: false,
            secret: 0x0
        });

        emit NewContract(contractId, msg.sender, recipient, token, amount, hashlock, timelock);
    }

    /**
     * @dev Called by the sender to get a refund if the timelock has expired
     * and the funds have not been claimed.
     * @param contractId The ID of the contract to refund.
     */
    function refund(bytes32 contractId) public {
        LockContract storage c = contracts[contractId];

        require(c.sender != address(0), "Contract does not exist");
        require(!c.refunded, "Already refunded");
        require(c.sender == msg.sender, "Only sender can refund");
        require(isExpired(c.timelock), "Timelock has not expired yet");

        c.refunded = true;

        if (c.token == address(0)) {
            payable(c.sender).transfer(c.amount);
        } else {
            IERC20(c.token).transfer(c.sender, c.amount);
        }

        emit Refunded(contractId);
    }
}
