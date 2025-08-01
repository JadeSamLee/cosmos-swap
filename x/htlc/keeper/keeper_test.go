package keeper_test

import (
	"testing"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/keeper"
	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"cosmossdk.io/store"
	"cosmossdk.io/store/mem"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proto/tendermint/types"
)

func createTestInput(t *testing.T) (sdk.Context, keeper.Keeper) {
	key := sdk.NewKVStoreKey(types.StoreKey)
	memKey := sdk.NewMemoryStoreKey(types.StoreKey)

	db := mem.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(key, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(memKey, sdk.StoreTypeMemory, nil)
	err := ms.LoadLatestVersion()
	require.NoError(t, err)

	registry := types.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(cdc, key)

	ctx := sdk.NewContext(ms, types.Header{Time: time.Now()}, false, log.NewNopLogger())

	return ctx, k
}

func TestCreateClaimRefundHTLC(t *testing.T) {
	ctx, k := createTestInput(t)

	sender := sdk.AccAddress([]byte("sender---------------"))
	receiver := sdk.AccAddress([]byte("receiver-------------"))
	amount := sdk.NewCoins(sdk.NewInt64Coin("token", 100))
	hashLock := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	timeLock := ctx.BlockTime().Add(time.Hour).Unix()

	id, err := k.CreateHTLC(ctx, sender, receiver, amount, hashLock, timeLock)
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)

	err = k.ClaimHTLC(ctx, id, hashLock, receiver)
	require.Error(t, err, "invalid preimage")

	preimage := hashLock // For test, use hashLock as preimage (should be hash of preimage)
	err = k.ClaimHTLC(ctx, id, preimage, receiver)
	require.Error(t, err, "invalid preimage")

	err = k.RefundHTLC(ctx, id, sender)
	require.Error(t, err, "htlc not expired")
}
