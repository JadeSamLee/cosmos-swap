// SPDX-License-Identifier: MIT
pragma solidity 0.8.23;

import {Ownable} from "openzeppelin-contracts/access/Ownable.sol";
import {IERC20} from "openzeppelin-contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "openzeppelin-contracts/token/ERC20/utils/SafeERC20.sol";
import {ReentrancyGuard} from "openzeppelin-contracts/security/ReentrancyGuard.sol";

/**
 * @title IBCHandler
 * @dev Simplified IBC handler for cross-chain communication between Ethereum and Cronos
 * This is a simplified implementation - in production, you'd use a full IBC client
 */
contract IBCHandler is Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    struct Channel {
        string channelId;
        string portId;
        string counterpartyChannelId;
        string counterpartyPortId;
        bool isActive;
        uint64 nextSequenceSend;
        uint64 nextSequenceRecv;
    }

    struct Packet {
        uint64 sequence;
        string sourcePort;
        string sourceChannel;
        string destinationPort;
        string destinationChannel;
        bytes data;
        uint64 timeoutTimestamp;
        uint256 blockHeight;
        bytes32 commitment;
    }

    struct PacketData {
        address sender;
        address receiver;
        uint256 amount;
        string denom;
    }

    // State variables
    mapping(string => Channel) public channels;
    mapping(bytes32 => bool) public packetCommitments;
    mapping(bytes32 => bool) public packetReceipts;
    mapping(bytes32 => bool) public packetAcknowledgements;
    
    // Relayer management
    mapping(address => bool) public authorizedRelayers;
    
    // Events
    event ChannelCreated(
        string indexed channelId,
        string portId,
        string counterpartyChannelId,
        string counterpartyPortId
    );
    
    event PacketSent(
        uint64 indexed sequence,
        string indexed sourceChannel,
        string destinationChannel,
        bytes data,
        uint64 timeoutTimestamp
    );
    
    event PacketReceived(
        uint64 indexed sequence,
        string indexed sourceChannel,
        string destinationChannel,
        bytes data
    );
    
    event PacketAcknowledged(
        uint64 indexed sequence,
        string indexed sourceChannel,
        bytes acknowledgement
    );
    
    event PacketTimeout(
        uint64 indexed sequence,
        string indexed sourceChannel
    );

    event RelayerAuthorized(address indexed relayer);
    event RelayerRevoked(address indexed relayer);

    // Errors
    error ChannelNotFound();
    error ChannelNotActive();
    error InvalidPacket();
    error PacketAlreadyReceived();
    error PacketNotFound();
    error UnauthorizedRelayer();
    error InvalidProof();
    error PacketTimeout();

    modifier onlyAuthorizedRelayer() {
        if (!authorizedRelayers[msg.sender] && msg.sender != owner()) {
            revert UnauthorizedRelayer();
        }
        _;
    }

    constructor(address initialOwner) Ownable(initialOwner) {
        // Authorize the owner as a relayer by default
        authorizedRelayers[initialOwner] = true;
    }

    /**
     * @dev Create a new IBC channel
     */
    function createChannel(
        string calldata channelId,
        string calldata portId,
        string calldata counterpartyChannelId,
        string calldata counterpartyPortId
    ) external onlyOwner {
        channels[channelId] = Channel({
            channelId: channelId,
            portId: portId,
            counterpartyChannelId: counterpartyChannelId,
            counterpartyPortId: counterpartyPortId,
            isActive: true,
            nextSequenceSend: 1,
            nextSequenceRecv: 1
        });

        emit ChannelCreated(channelId, portId, counterpartyChannelId, counterpartyPortId);
    }

    /**
     * @dev Send an IBC packet
     */
    function sendPacket(
        string calldata channelId,
        bytes calldata data,
        uint64 timeoutTimestamp
    ) external payable nonReentrant {
        Channel storage channel = channels[channelId];
        if (bytes(channel.channelId).length == 0) revert ChannelNotFound();
        if (!channel.isActive) revert ChannelNotActive();

        uint64 sequence = channel.nextSequenceSend;
        channel.nextSequenceSend++;

        // Create packet commitment
        bytes32 commitment = keccak256(abi.encodePacked(
            sequence,
            channel.portId,
            channelId,
            channel.counterpartyPortId,
            channel.counterpartyChannelId,
            data,
            timeoutTimestamp
        ));

        packetCommitments[commitment] = true;

        // Handle ETH transfers
        if (msg.value > 0) {
            // Lock ETH in this contract
            // In a real implementation, this would be handled by the transfer module
        }

        emit PacketSent(
            sequence,
            channelId,
            channel.counterpartyChannelId,
            data,
            timeoutTimestamp
        );
    }

    /**
     * @dev Receive an IBC packet (called by relayer)
     */
    function recvPacket(
        Packet calldata packet,
        bytes calldata proof,
        uint256 proofHeight
    ) external onlyAuthorizedRelayer nonReentrant {
        // Verify channel exists and is active
        Channel storage channel = channels[packet.destinationChannel];
        if (bytes(channel.channelId).length == 0) revert ChannelNotFound();
        if (!channel.isActive) revert ChannelNotActive();

        // Create packet receipt key
        bytes32 receiptKey = keccak256(abi.encodePacked(
            packet.destinationPort,
            packet.destinationChannel,
            packet.sequence
        ));

        // Check if packet already received
        if (packetReceipts[receiptKey]) revert PacketAlreadyReceived();

        // Verify proof (simplified - in production, this would verify Merkle proofs)
        if (!_verifyPacketProof(packet, proof, proofHeight)) revert InvalidProof();

        // Check timeout
        if (packet.timeoutTimestamp > 0 && block.timestamp >= packet.timeoutTimestamp) {
            revert PacketTimeout();
        }

        // Mark packet as received
        packetReceipts[receiptKey] = true;
        channel.nextSequenceRecv++;

        // Process packet data
        _processPacketData(packet.data);

        emit PacketReceived(
            packet.sequence,
            packet.sourceChannel,
            packet.destinationChannel,
            packet.data
        );
    }

    /**
     * @dev Acknowledge a packet (called by relayer)
     */
    function acknowledgePacket(
        Packet calldata packet,
        bytes calldata acknowledgement,
        bytes calldata proof,
        uint256 proofHeight
    ) external onlyAuthorizedRelayer {
        // Verify packet commitment exists
        bytes32 commitment = keccak256(abi.encodePacked(
            packet.sequence,
            packet.sourcePort,
            packet.sourceChannel,
            packet.destinationPort,
            packet.destinationChannel,
            packet.data,
            packet.timeoutTimestamp
        ));

        if (!packetCommitments[commitment]) revert PacketNotFound();

        // Verify acknowledgement proof (simplified)
        if (!_verifyAckProof(packet, acknowledgement, proof, proofHeight)) {
            revert InvalidProof();
        }

        // Mark as acknowledged
        bytes32 ackKey = keccak256(abi.encodePacked(
            packet.sourcePort,
            packet.sourceChannel,
            packet.sequence
        ));
        packetAcknowledgements[ackKey] = true;

        // Remove commitment
        delete packetCommitments[commitment];

        emit PacketAcknowledged(packet.sequence, packet.sourceChannel, acknowledgement);
    }

    /**
     * @dev Handle packet timeout
     */
    function timeoutPacket(
        Packet calldata packet,
        bytes calldata proof,
        uint256 proofHeight
    ) external onlyAuthorizedRelayer {
        // Verify packet commitment exists
        bytes32 commitment = keccak256(abi.encodePacked(
            packet.sequence,
            packet.sourcePort,
            packet.sourceChannel,
            packet.destinationPort,
            packet.destinationChannel,
            packet.data,
            packet.timeoutTimestamp
        ));

        if (!packetCommitments[commitment]) revert PacketNotFound();

        // Verify timeout proof (simplified)
        if (!_verifyTimeoutProof(packet, proof, proofHeight)) revert InvalidProof();

        // Check if actually timed out
        if (packet.timeoutTimestamp == 0 || block.timestamp < packet.timeoutTimestamp) {
            revert InvalidPacket();
        }

        // Remove commitment and refund
        delete packetCommitments[commitment];
        _refundPacket(packet);

        emit PacketTimeout(packet.sequence, packet.sourceChannel);
    }

    /**
     * @dev Authorize a relayer
     */
    function authorizeRelayer(address relayer) external onlyOwner {
        authorizedRelayers[relayer] = true;
        emit RelayerAuthorized(relayer);
    }

    /**
     * @dev Revoke relayer authorization
     */
    function revokeRelayer(address relayer) external onlyOwner {
        authorizedRelayers[relayer] = false;
        emit RelayerRevoked(relayer);
    }

    /**
     * @dev Set channel active status
     */
    function setChannelActive(string calldata channelId, bool active) external onlyOwner {
        Channel storage channel = channels[channelId];
        if (bytes(channel.channelId).length == 0) revert ChannelNotFound();
        channel.isActive = active;
    }

    /**
     * @dev Process incoming packet data
     */
    function _processPacketData(bytes calldata data) internal {
        // Decode packet data
        PacketData memory packetData = abi.decode(data, (PacketData));

        // Handle token transfer
        if (keccak256(bytes(packetData.denom)) == keccak256(bytes("ETH"))) {
            // Transfer ETH
            (bool success, ) = packetData.receiver.call{value: packetData.amount}("");
            require(success, "ETH transfer failed");
        } else {
            // Handle ERC20 token transfer
            // In a real implementation, this would involve minting/burning tokens
            // or transferring from a pool
            IERC20(packetData.receiver).safeTransfer(packetData.receiver, packetData.amount);
        }
    }

    /**
     * @dev Refund packet on timeout
     */
    function _refundPacket(Packet calldata packet) internal {
        PacketData memory packetData = abi.decode(packet.data, (PacketData));

        // Refund to sender
        if (keccak256(bytes(packetData.denom)) == keccak256(bytes("ETH"))) {
            (bool success, ) = packetData.sender.call{value: packetData.amount}("");
            require(success, "ETH refund failed");
        } else {
            IERC20(packetData.sender).safeTransfer(packetData.sender, packetData.amount);
        }
    }

    /**
     * @dev Verify packet proof (simplified implementation)
     */
    function _verifyPacketProof(
        Packet calldata packet,
        bytes calldata proof,
        uint256 proofHeight
    ) internal pure returns (bool) {
        // In a real implementation, this would verify Merkle proofs against
        // the counterparty chain's state root
        // For now, we just check that proof is not empty
        return proof.length > 0 && proofHeight > 0;
    }

    /**
     * @dev Verify acknowledgement proof (simplified implementation)
     */
    function _verifyAckProof(
        Packet calldata packet,
        bytes calldata acknowledgement,
        bytes calldata proof,
        uint256 proofHeight
    ) internal pure returns (bool) {
        // Simplified verification
        return proof.length > 0 && acknowledgement.length > 0 && proofHeight > 0;
    }

    /**
     * @dev Verify timeout proof (simplified implementation)
     */
    function _verifyTimeoutProof(
        Packet calldata packet,
        bytes calldata proof,
        uint256 proofHeight
    ) internal pure returns (bool) {
        // Simplified verification
        return proof.length > 0 && proofHeight > 0;
    }

    /**
     * @dev Get channel information
     */
    function getChannel(string calldata channelId) external view returns (Channel memory) {
        return channels[channelId];
    }

    /**
     * @dev Check if packet commitment exists
     */
    function hasPacketCommitment(bytes32 commitment) external view returns (bool) {
        return packetCommitments[commitment];
    }

    /**
     * @dev Check if packet receipt exists
     */
    function hasPacketReceipt(bytes32 receiptKey) external view returns (bool) {
        return packetReceipts[receiptKey];
    }

    /**
     * @dev Emergency recovery function
     */
    function emergencyRecovery(address token, uint256 amount) external onlyOwner {
        if (token == address(0)) {
            (bool success, ) = owner().call{value: amount}("");
            require(success, "ETH recovery failed");
        } else {
            IERC20(token).safeTransfer(owner(), amount);
        }
    }

    // Allow contract to receive ETH
    receive() external payable {}
}

