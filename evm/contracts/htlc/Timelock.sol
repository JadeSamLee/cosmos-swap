// SPDX-License-Identifier: MIT
pragma solidity ^0.8.23;

/**
 * @title Timelock
 * @dev Provides timelock functionality for HTLC contracts.
 */
contract Timelock {
    /**
     * @dev Checks if the timelock has expired.
     * @param timelock The Unix timestamp of the timelock expiration.
     * @return True if the current block timestamp is greater than or equal to the timelock.
     */
    function isExpired(uint256 timelock) public view returns (bool) {
        return block.timestamp >= timelock;
    }
}
