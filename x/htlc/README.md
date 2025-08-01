# HTLC Module Deployment and Testing Instructions

This document provides instructions to deploy and test the HTLC (Hashed Timelock Contract) module on existing Cronos testnet and Sepolia testnet.

## Deployment

1. **Build and Deploy the Module**

   - Integrate the `x/htlc` module into your Cronos chain codebase.
   - Add the module to the app's module manager and wiring.
   - Build the chain binary and deploy it to your testnet nodes.
   - Update the genesis file to include the HTLC module's genesis state.

2. **Upgrade the Chain**

   - Perform a chain upgrade with the new binary including the HTLC module.
   - Ensure the module is properly initialized on chain start.

## Testing HTLC Functionality

You can interact with the HTLC module using the CLI commands provided.

### CLI Commands

Assuming you have the CLI binary (e.g., `cronosd`) with the HTLC module commands registered:

- **Create HTLC**

  ```bash
  cronosd tx htlc create-htlc <receiver_address> <amount> <hashlock> <timelock> --from <sender_key> --chain-id <chain_id> --gas auto --fees <fees>
  ```

  - `receiver_address`: Bech32 address of the receiver.
  - `amount`: Amount to lock, e.g., `100cro`.
  - `hashlock`: SHA256 hash of the secret preimage (hex encoded).
  - `timelock`: Unix timestamp (seconds) after which refund is possible.

- **Claim HTLC**

  ```bash
  cronosd tx htlc claim-htlc <htlc_id> <preimage> --from <receiver_key> --chain-id <chain_id> --gas auto --fees <fees>
  ```

  - `htlc_id`: ID of the HTLC to claim.
  - `preimage`: Secret preimage that hashes to the hashlock.

- **Refund HTLC**

  ```bash
  cronosd tx htlc refund-htlc <htlc_id> --from <sender_key> --chain-id <chain_id> --gas auto --fees <fees>
  ```

  - `htlc_id`: ID of the HTLC to refund.

### Example Workflow

1. Generate a secret and compute its SHA256 hash.

2. Create an HTLC locking funds with the hashlock and timelock.

3. The receiver claims the HTLC by providing the preimage before timelock expiry.

4. If the receiver does not claim in time, the sender refunds the HTLC after timelock.

## Notes

- Ensure your CLI is configured to connect to the correct testnet RPC endpoint.

- Adjust gas and fees according to the network requirements.

- Use appropriate keys/accounts with sufficient funds for testing.

## Sepolia Testnet

- Use the Sepolia chain ID and RPC endpoints.

- Ensure the HTLC module is deployed and enabled on Sepolia.

## Cronos Testnet

- Use the Cronos testnet chain ID and RPC endpoints.

- Ensure the HTLC module is deployed and enabled on Cronos testnet.

---

For any issues or further assistance, please reach out with your specific setup details.
