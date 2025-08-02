# HTLC Module

## Overview

The HTLC (Hashed Time-Locked Contract) module implements atomic swap functionality using HTLCs (Hashed Time-Locked Contracts). This module allows users to create, claim, and refund HTLCs for cross-chain atomic swaps.

## Contents

- [Concepts](#concepts)
- [State](#state)
- [Messages](#messages)
  - [MsgCreateHTLC](#msgcreatehtlc)
  - [MsgClaimHTLC](#msgclaimhtlc)
  - [MsgRefundHTLC](#msgrefundhtlc)
- [Events](#events)
- [CLI](#cli)
  - [Transactions](#transactions)
  - [Queries](#queries)

## Concepts

The HTLC module implements the Hashed Time-Locked Contract (HTLC) pattern, which is a critical component for cross-chain atomic swaps. It allows two parties to create a time-locked contract where one party can lock funds with a hash lock, and the counterparty can claim the funds by providing the preimage of the hash.

### How it works

1. Alice wants to swap tokens with Bob.
2. Alice creates an HTLC, locking her tokens with a hash lock (H) and a time lock.
3. Bob can claim the funds by providing the preimage of the hash (h) such that H = Hash(h).
4. If Bob doesn't claim in time, Alice can refund after the time lock expires.

## State

### HTLC

- HTLC: `0x01 | BigEndian(id) -> ProtocolBuffer(HTLC)`

## Messages

### `MsgCreateHTLC`

Allows creating a new HTLC.

```protobuf
rpc CreateHTLC(MsgCreateHTLC) returns (MsgCreateHTLCResponse);
```

**State Modifications**
- Creates a new HTLC with a unique ID
- Transfers tokens from the sender to the module account
- Emits Event `create_htlc`

**State Modifications**
- Appends a new HTLC to the state
- Updates the next HTLC ID

**Expected Keepers/Assumptions**
- The sender has sufficient balance to cover the amount to be locked
- The time lock is in the future

### `MsgClaimHTLC`

Allows claiming an existing HTLC by providing the preimage of the hash lock.

```protobuf
rpc ClaimHTLC(MsgClaimHTLC) returns (MsgClaimHTLCResponse);
```

**State Modifications**
- Marks the HTLC as claimed
- Transfers tokens to the receiver

**Expected Keepers/Assumptions**
- The preimage must be the preimage of the hash lock
- The claimer is the receiver of the HTLC
- The HTLC has not been claimed or refunded
- The HTLC has not expired

### `MsgRefundHTLC`

Allows refunding an HTLC after the time lock has expired.

```protobuf
rpc RefundHTLC(MsgRefundHTLC) returns (MsgRefundHTLCResponse);
```

**State Modifications**
- Marks the HTLC as refunded
- Transfers tokens back to the sender

**Expected Keepers/Assumptions**
- The refunder is the original sender of the HTLC
- The HTLC has not been claimed or refunded
- The HTLC has expired

## Events

- `create_htlc`
  - Emitted when a new HTLC is created
  - Keys: "create_htlc"
  - Attributes:
    - "sender": The address of the account that created the HTLC
    - "receiver": The address of the account that can claim the HTLC
    - "htlc_id": The ID of the HTLC
    - "amount": The amount of coins locked in the HTLC
    - "hash_lock": The hash lock of the HTLC
    - "time_lock": The time lock of the HTLC

- `claim_htlc`
  - Emitted when an HTLC is claimed
  - Keys: "claim_htlc"
  - Attributes:
    - "htlc_id": The ID of the HTLC
    - "receiver": The address of the account that claimed the HTLC
    - "amount": The amount of coins claimed

- `refund_htlc`
  - Emitted when an HTLC is refunded
  - Keys: "refund_htlc"
  - Attributes:
    - "htlc_id": The ID of the HTLC
    - "sender": The address of the account that created the HTLC
    - "amount": The amount of coins refunded

## CLI

### Transactions

#### create-htlc

Create a new HTLC.

```text
create-htlc [receiver] [amount] [hashlock] [timelock]
```

Example:
`create-htlc cosmos1... 1000stake 0x1234567890abcdef... 1620000000`

#### claim-htlc

Claim an HTLC by providing the preimage.

```text
claim-htlc [htlc-id] [preimage]
```

Example:
`claim-htlc 1 0xabcdef1234567890...`

#### refund-htlc

Refund an HTLC after the time lock has expired.

```text
refund-htlc [htlc-id]
```

Example:
`refund-htlc 1`

### Queries

#### list-htlcs

List all HTLCs.

```text
list-htlcs
```

#### show-htlc

Show details of a specific HTLC by ID.

```text
show-htlc [id]
```

Example:
`show-htlc 1`
