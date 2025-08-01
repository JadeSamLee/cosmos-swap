package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"time"
)

type HTLC struct {
	Id        uint64         `json:"id" yaml:"id"`
	Sender    sdk.AccAddress `json:"sender" yaml:"sender"`
	Receiver  sdk.AccAddress `json:"receiver" yaml:"receiver"`
	Amount    sdk.Coins      `json:"amount" yaml:"amount"`
	HashLock  []byte         `json:"hash_lock" yaml:"hash_lock"`
	TimeLock  time.Time      `json:"time_lock" yaml:"time_lock"`
	Claimed   bool           `json:"claimed" yaml:"claimed"`
	Refunded  bool           `json:"refunded" yaml:"refunded"`
}

type GenesisState struct {
	HTLCs []HTLC `json:"htlcs" yaml:"htlcs"`
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		HTLCs: []HTLC{},
	}
}

func (gs GenesisState) Validate() error {
	// Add validation logic if needed
	return nil
}
