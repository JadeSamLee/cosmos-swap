package keeper

import (
	"context"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
)

// MsgServerMock is a mock implementation of the MsgServer interface for testing purposes
type MsgServerMock struct {
	CreateHTLCFunc func(context.Context, *types.MsgCreateHTLC) (*types.MsgCreateHTLCResponse, error)
	ClaimHTLCFunc  func(context.Context, *types.MsgClaimHTLC) (*types.MsgClaimHTLCResponse, error)
	RefundHTLCFunc func(context.Context, *types.MsgRefundHTLC) (*types.MsgRefundHTLCResponse, error)
}

// CreateHTLC is a mock implementation of the CreateHTLC method
func (m *MsgServerMock) CreateHTLC(ctx context.Context, msg *types.MsgCreateHTLC) (*types.MsgCreateHTLCResponse, error) {
	if m.CreateHTLCFunc != nil {
		return m.CreateHTLCFunc(ctx, msg)
	}
	return &types.MsgCreateHTLCResponse{}, nil
}

// ClaimHTLC is a mock implementation of the ClaimHTLC method
func (m *MsgServerMock) ClaimHTLC(ctx context.Context, msg *types.MsgClaimHTLC) (*types.MsgClaimHTLCResponse, error) {
	if m.ClaimHTLCFunc != nil {
		return m.ClaimHTLCFunc(ctx, msg)
	}
	return &types.MsgClaimHTLCResponse{}, nil
}

// RefundHTLC is a mock implementation of the RefundHTLC method
func (m *MsgServerMock) RefundHTLC(ctx context.Context, msg *types.MsgRefundHTLC) (*types.MsgRefundHTLCResponse, error) {
	if m.RefundHTLCFunc != nil {
		return m.RefundHTLCFunc(ctx, msg)
	}
	return &types.MsgRefundHTLCResponse{}, nil
}
