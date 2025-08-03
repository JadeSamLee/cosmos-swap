// Package types defines the custom types and interfaces for the HTLC module.
package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

// HTLC represents a Hashed Time-Locked Contract
type HTLC struct {
	// Id is the unique identifier for the HTLC
	Id uint64 `json:"id" yaml:"id"`
	
	// Sender is the address of the account that created the HTLC
	Sender sdk.AccAddress `json:"sender" yaml:"sender"`
	
	// Receiver is the address of the account that can claim the HTLC
	Receiver sdk.AccAddress `json:"receiver" yaml:"receiver"`
	
	// Amount is the coins locked in the HTLC
	Amount sdk.Coins `json:"amount" yaml:"amount"`
	
	// HashLock is the SHA256 hash of the preimage
	HashLock []byte `json:"hash_lock" yaml:"hash_lock"`
	
	// TimeLock is the time after which the HTLC can be refunded
	TimeLock time.Time `json:"time_lock" yaml:"time_lock"`
	
	// Claimed indicates whether the HTLC has been claimed
	Claimed bool `json:"claimed" yaml:"claimed"`
	
	// Refunded indicates whether the HTLC has been refunded
	Refunded bool `json:"refunded" yaml:"refunded"`
}

// GenesisState represents the genesis state for the HTLC module
type GenesisState struct {
	// HTLCs is the list of HTLCs at genesis
	HTLCs []HTLC `json:"htlcs" yaml:"htlcs"`
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		HTLCs: []HTLC{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Add validation logic if needed
	return nil
}
