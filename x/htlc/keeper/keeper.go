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
		return 0, fmt.Errorf("hashLock must be sha256 hash")
	}
	if timeLock <= ctx.BlockTime().Unix() {
		return 0, fmt.Errorf("timeLock must be in the future")
	}

	// send coins from sender to module account to lock
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return 0, fmt.Errorf("failed to lock coins: %w", err)
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
	return id, nil
}

func (k Keeper) ClaimHTLC(ctx sdk.Context, id uint64, preimage []byte, claimer sdk.AccAddress) error {
	htlc, found := k.GetHTLC(ctx, id)
	if !found {
		return fmt.Errorf("htlc not found")
	}
	if htlc.Claimed {
		return fmt.Errorf("htlc already claimed")
	}
	if htlc.Refunded {
		return fmt.Errorf("htlc already refunded")
	}
	if !bytes.Equal(sha256.Sum256(preimage)[:], htlc.HashLock) {
		return fmt.Errorf("invalid preimage")
	}
	if !claimer.Equals(htlc.Receiver) {
		return fmt.Errorf("only receiver can claim")
	}
	if ctx.BlockTime().After(htlc.TimeLock) {
		return fmt.Errorf("htlc expired")
	}

	htlc.Claimed = true
	k.SetHTLC(ctx, htlc)

	// transfer coins to receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Receiver, htlc.Amount); err != nil {
		return fmt.Errorf("failed to send coins to receiver: %w", err)
	}

	return nil
}

func (k Keeper) RefundHTLC(ctx sdk.Context, id uint64, refunder sdk.AccAddress) error {
	htlc, found := k.GetHTLC(ctx, id)
	if !found {
		return fmt.Errorf("htlc not found")
	}
	if htlc.Claimed {
		return fmt.Errorf("htlc already claimed")
	}
	if htlc.Refunded {
		return fmt.Errorf("htlc already refunded")
	}
	if !refunder.Equals(htlc.Sender) {
		return fmt.Errorf("only sender can refund")
	}
	if ctx.BlockTime().Before(htlc.TimeLock) {
		return fmt.Errorf("htlc not expired")
	}

	htlc.Refunded = true
	k.SetHTLC(ctx, htlc)

	// refund coins to sender
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Sender, htlc.Amount); err != nil {
		return fmt.Errorf("failed to refund coins to sender: %w", err)
	}

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
