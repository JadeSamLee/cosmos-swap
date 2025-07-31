package htlc

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/codec"
    "github.com/cosmos/cosmos-sdk/codec/types"
    "github.com/gogo/protobuf/proto"
)

// HTLC defines a Hashed TimeLock Contract
type HTLC struct {
    Sender         sdk.AccAddress `json:"sender" yaml:"sender"`
    Recipient      sdk.AccAddress `json:"recipient" yaml:"recipient"`
    Amount         sdk.Coins      `json:"amount" yaml:"amount"`
    HashLock       []byte         `json:"hash_lock" yaml:"hash_lock"`
    MerkleRoot     []byte         `json:"merkle_root" yaml:"merkle_root"`
    TimeLock       int64          `json:"time_lock" yaml:"time_lock"` // block height
    SecretRevealed bool           `json:"secret_revealed" yaml:"secret_revealed"`
    ClaimedAmount  sdk.Coins      `json:"claimed_amount" yaml:"claimed_amount"`
}

// NewHTLC creates a new HTLC instance
func NewHTLC(sender, recipient sdk.AccAddress, amount sdk.Coins, hashLock []byte, merkleRoot []byte, timeLock int64) HTLC {
    return HTLC{
        Sender:         sender,
        Recipient:      recipient,
        Amount:         amount,
        HashLock:       hashLock,
        MerkleRoot:     merkleRoot,
        TimeLock:       timeLock,
        SecretRevealed: false,
    }
}

// RegisterLegacyAminoCodec registers the necessary types for amino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
    cdc.RegisterConcrete(HTLC{}, "htlc/HTLC", nil)
}

// RegisterInterfaces registers the interface types
func RegisterInterfaces(registry types.InterfaceRegistry) {
    registry.RegisterImplementations((*proto.Message)(nil),
        &HTLC{},
    )
}
