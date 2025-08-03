package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrInvalidHashLock      = sdkerrors.Register(ModuleName, 1, "invalid hash lock")
	ErrInvalidTimeLock      = sdkerrors.Register(ModuleName, 2, "invalid time lock")
	ErrHTLCNotFound         = sdkerrors.Register(ModuleName, 3, "htlc not found")
	ErrHTLCClaimed          = sdkerrors.Register(ModuleName, 4, "htlc already claimed")
	ErrHTLCRefunded         = sdkerrors.Register(ModuleName, 5, "htlc already refunded")
	ErrInvalidPreimage      = sdkerrors.Register(ModuleName, 6, "invalid preimage")
	ErrUnauthorizedClaimer  = sdkerrors.Register(ModuleName, 7, "unauthorized claimer")
	ErrHTLCNotExpired       = sdkerrors.Register(ModuleName, 8, "htlc not expired")
	ErrUnauthorizedRefunder = sdkerrors.Register(ModuleName, 9, "unauthorized refunder")
	ErrHTLCExpired          = sdkerrors.Register(ModuleName, 10, "htlc expired")
)
