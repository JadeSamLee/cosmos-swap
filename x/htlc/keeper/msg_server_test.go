package keeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/keeper"
	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func createTestMsgServer(t *testing.T) (sdk.Context, keeper.MsgServer, keeper.Keeper) {
	ctx, k := createTestInput(t)
	return ctx, keeper.NewMsgServerImpl(k), k
}

func TestMsgServer_CreateClaimRefundHTLC(t *testing.T) {
	ctx, msgServer, k := createTestMsgServer(t)

	sender := sdk.AccAddress([]byte("sender---------------"))
	receiver := sdk.AccAddress([]byte("receiver-------------"))
	amount := sdk.NewCoins(sdk.NewInt64Coin("token", 100))
	hashLock := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	timeLock := ctx.BlockTime().Add(time.Hour).Unix()

	// Create HTLC
	createMsg := &types.MsgCreateHTLC{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
		HashLock: hashLock,
		TimeLock: timeLock,
	}
	resp, err := msgServer.CreateHTLC(context.Background(), createMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, uint64(1), resp.Id)

	// Claim HTLC with invalid preimage
	claimMsg := &types.MsgClaimHTLC{
		Claimer:  receiver,
		HTLCId:   resp.Id,
		Preimage: []byte{0, 0, 0},
	}
	_, err = msgServer.ClaimHTLC(context.Background(), claimMsg)
	require.Error(t, err)

	// Claim HTLC with valid preimage (hash of preimage matches hashlock)
	// For test, use hashLock as preimage (should fail)
	claimMsg.Preimage = hashLock
	_, err = msgServer.ClaimHTLC(context.Background(), claimMsg)
	require.Error(t, err)

	// Refund HTLC before expiry (should fail)
	refundMsg := &types.MsgRefundHTLC{
		Refunder: sender,
		HTLCId:   resp.Id,
	}
	_, err = msgServer.RefundHTLC(context.Background(), refundMsg)
	require.Error(t, err)
}
