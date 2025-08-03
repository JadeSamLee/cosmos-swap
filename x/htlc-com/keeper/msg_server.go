package keeper

import (
	"context"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (k msgServer) CreateHTLC(goCtx context.Context, msg *types.MsgCreateHTLC) (*types.MsgCreateHTLCResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	id, err := k.CreateHTLC(ctx, msg.Sender, msg.Receiver, msg.Amount, msg.HashLock, msg.TimeLock)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateHTLCResponse{Id: id}, nil
}

func (k msgServer) ClaimHTLC(goCtx context.Context, msg *types.MsgClaimHTLC) (*types.MsgClaimHTLCResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.ClaimHTLC(ctx, msg.HTLCId, msg.Preimage, msg.Claimer)
	if err != nil {
		return nil, err
	}

	return &types.MsgClaimHTLCResponse{}, nil
}

func (k msgServer) RefundHTLC(goCtx context.Context, msg *types.MsgRefundHTLC) (*types.MsgRefundHTLCResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.RefundHTLC(ctx, msg.HTLCId, msg.Refunder)
	if err != nil {
		return nil, err
	}

	return &types.MsgRefundHTLCResponse{}, nil
}
