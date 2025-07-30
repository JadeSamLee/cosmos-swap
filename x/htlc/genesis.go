package htlc

import (
    "encoding/json"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenesisState defines the htlc module's genesis state.
type GenesisState struct {
    HTLCs []HTLC `json:"htlcs" yaml:"htlcs"`
}

// DefaultGenesisState returns the default genesis state for the htlc module.
func DefaultGenesisState() GenesisState {
    return GenesisState{
        HTLCs: []HTLC{},
    }
}

// ValidateGenesis validates the genesis state.
func ValidateGenesis(data GenesisState) error {
    // Add validation logic here if needed
    return nil
}
