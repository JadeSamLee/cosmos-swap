package keeper

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/stretchr/testify/require"
)

func TestMsgServerMock(t *testing.T) {
	mock := NewMsgServerMock()
	ctx := context.Background()

	// Create HTLC
	createMsg := &types.MsgCreateHTLC{
		Id:       "htlc1",
		Sender:   "sender1",
		Receiver: "receiver1",
		HashLock: "hashlock1",
		TimeLock: 10, // 10 seconds
		Amount:   sdk.NewCoins(sdk.NewInt64Coin("cro", 100)),
	}
	_, err := mock.CreateHTLC(ctx, createMsg)
	require.NoError(t, err)

	// Claim HTLC with correct secret
	claimMsg := &types.MsgClaimHTLC{
		Id:     "htlc1",
		Secret: "hashlock1",
	}
	_, err = mock.ClaimHTLC(ctx, claimMsg)
	require.NoError(t, err)

	// Refund HTLC (should fail because already claimed)
	refundMsg := &types.MsgRefundHTLC{
		Id: "htlc1",
	}
	_, err = mock.RefundHTLC(ctx, refundMsg)
	require.Error(t, err)

	// Create another HTLC for refund test
	createMsg2 := &types.MsgCreateHTLC{
		Id:       "htlc2",
		Sender:   "sender2",
		Receiver: "receiver2",
		HashLock: "hashlock2",
		TimeLock: 1, // 1 second
		Amount:   sdk.NewCoins(sdk.NewInt64Coin("cro", 50)),
	}
	_, err = mock.CreateHTLC(ctx, createMsg2)
	require.NoError(t, err)

	// Wait for expiry
	time.Sleep(2 * time.Second)

	// Refund HTLC after expiry
	refundMsg2 := &types.MsgRefundHTLC{
		Id: "htlc2",
	}
	_, err = mock.RefundHTLC(ctx, refundMsg2)
	require.NoError(t, err)
}
