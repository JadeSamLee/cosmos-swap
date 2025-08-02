package keeper

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "cosmossdk.io/store/types"
)

// Event types
const (
	EventTypeCreateHTLC = "create_htlc"
	EventTypeClaimHTLC  = "claim_htlc"
	EventTypeRefundHTLC = "refund_htlc"

	AttributeKeySender    = "sender"
	AttributeKeyReceiver  = "receiver"
	AttributeKeyHTLCID    = "htlc_id"
	AttributeKeyAmount    = "amount"
	AttributeKeyHashLock = "hash_lock"
	AttributeKeyTimeLock  = "time_lock"
)

type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	bankKeeper types.BankKeeper
}

func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, bankKeeper types.BankKeeper) Keeper {
	return Keeper{
		storeKey:   storeKey,
		cdc:        cdc,
		bankKeeper: bankKeeper,
	}
}

func (k Keeper) GetHTLC(ctx sdk.Context, id uint64) (types.HTLC, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetHTLCKey(id))
	if bz == nil {
		return types.HTLC{}, false
	}
	var htlc types.HTLC
	k.cdc.MustUnmarshal(bz, &htlc)
	return htlc, true
}

func (k Keeper) SetHTLC(ctx sdk.Context, htlc types.HTLC) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&htlc)
	store.Set(types.GetHTLCKey(htlc.Id), bz)
}

func (k Keeper) DeleteHTLC(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetHTLCKey(id))
}

func (k Keeper) CreateHTLC(ctx sdk.Context, sender, receiver sdk.AccAddress, amount sdk.Coins, hashLock []byte, timeLock int64) (uint64, error) {
	if len(hashLock) != sha256.Size {
		return 0, types.ErrInvalidHashLock
	}
	if timeLock <= ctx.BlockTime().Unix() {
		return 0, types.ErrInvalidTimeLock
	}

	// send coins from sender to module account to lock
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return 0, err
	}

	id := k.GetNextHTLCId(ctx)
	htlc := types.HTLC{
		Id:       id,
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
		HashLock: hashLock,
		TimeLock: time.Unix(timeLock, 0),
		Claimed:  false,
		Refunded: false,
	}

	k.SetHTLC(ctx, htlc)
	k.IncrementNextHTLCId(ctx)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeCreateHTLC,
			sdk.NewAttribute(AttributeKeySender, sender.String()),
			sdk.NewAttribute(AttributeKeyReceiver, receiver.String()),
			sdk.NewAttribute(AttributeKeyHTLCID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(AttributeKeyAmount, amount.String()),
			sdk.NewAttribute(AttributeKeyHashLock, fmt.Sprintf("%x", hashLock)),
			sdk.NewAttribute(AttributeKeyTimeLock, time.Unix(timeLock, 0).String()),
		),
	)

	return id, nil
}

func (k Keeper) ClaimHTLC(ctx sdk.Context, id uint64, preimage []byte, claimer sdk.AccAddress) error {
	htlc, found := k.GetHTLC(ctx, id)
	if !found {
		return types.ErrHTLCNotFound
	}
	if htlc.Claimed {
		return types.ErrHTLCClaimed
	}
	if htlc.Refunded {
		return types.ErrHTLCRefunded
	}
	if !bytes.Equal(sha256.Sum256(preimage)[:], htlc.HashLock) {
		return types.ErrInvalidPreimage
	}
	if !claimer.Equals(htlc.Receiver) {
		return types.ErrUnauthorizedClaimer
	}
	if ctx.BlockTime().After(htlc.TimeLock) {
		return types.ErrHTLCExpired
	}

	htlc.Claimed = true
	k.SetHTLC(ctx, htlc)

	// transfer coins to receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Receiver, htlc.Amount); err != nil {
		return err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeClaimHTLC,
			sdk.NewAttribute(AttributeKeyHTLCID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(AttributeKeyReceiver, claimer.String()),
			sdk.NewAttribute(AttributeKeyAmount, htlc.Amount.String()),
		),
	)

	return nil
}

func (k Keeper) RefundHTLC(ctx sdk.Context, id uint64, refunder sdk.AccAddress) error {
	htlc, found := k.GetHTLC(ctx, id)
	if !found {
		return types.ErrHTLCNotFound
	}
	if htlc.Claimed {
		return types.ErrHTLCClaimed
	}
	if htlc.Refunded {
		return types.ErrHTLCRefunded
	}
	if !refunder.Equals(htlc.Sender) {
		return types.ErrUnauthorizedRefunder
	}
	if ctx.BlockTime().Before(htlc.TimeLock) {
		return types.ErrHTLCNotExpired
	}

	htlc.Refunded = true
	k.SetHTLC(ctx, htlc)

	// refund coins to sender
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Sender, htlc.Amount); err != nil {
		return err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeRefundHTLC,
			sdk.NewAttribute(AttributeKeyHTLCID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(AttributeKeySender, refunder.String()),
			sdk.NewAttribute(AttributeKeyAmount, htlc.Amount.String()),
		),
	)

	return nil
}

func (k Keeper) GetNextHTLCId(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyNextHTLCId)
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) IncrementNextHTLCId(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	id := k.GetNextHTLCId(ctx) + 1
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	store.Set(types.KeyNextHTLCId, bz)
}
