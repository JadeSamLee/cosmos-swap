package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
)

// MsgServerMock is a mock implementation of the HTLC MsgServer interface for testing
type MsgServerMock struct {
	store map[string]types.HTLC
}

// NewMsgServerMock creates a new MsgServerMock instance
func NewMsgServerMock() *MsgServerMock {
	return &MsgServerMock{
		store: make(map[string]types.HTLC),
	}
}

// CreateHTLC mocks creating a new HTLC
func (m *MsgServerMock) CreateHTLC(goCtx context.Context, msg *types.MsgCreateHTLC) (*types.MsgCreateHTLCResponse, error) {
	htlc := types.HTLC{
		Id:          msg.Id,
		Sender:      msg.Sender,
		Receiver:    msg.Receiver,
		HashLock:    msg.HashLock,
		TimeLock:    msg.TimeLock,
		Amount:      msg.Amount,
		Secret:      "",
		State:       types.HTLCStateOpen,
		ExpireTime:  time.Now().Add(time.Duration(msg.TimeLock) * time.Second).Unix(),
	}
	m.store[msg.Id] = htlc
	return &types.MsgCreateHTLCResponse{}, nil
}

// ClaimHTLC mocks claiming an HTLC with the secret
func (m *MsgServerMock) ClaimHTLC(goCtx context.Context, msg *types.MsgClaimHTLC) (*types.MsgClaimHTLCResponse, error) {
	htlc, found := m.store[msg.Id]
	if !found {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, "HTLC not found")
	}
	if htlc.State != types.HTLCStateOpen {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "HTLC not open")
	}
	if msg.Secret != "" && msg.Secret == htlc.HashLock { // Simplified check for demo
		htlc.State = types.HTLCStateClaimed
		htlc.Secret = msg.Secret
		m.store[msg.Id] = htlc
		return &types.MsgClaimHTLCResponse{}, nil
	}
	return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "Invalid secret")
}

// RefundHTLC mocks refunding an HTLC after expiry
func (m *MsgServerMock) RefundHTLC(goCtx context.Context, msg *types.MsgRefundHTLC) (*types.MsgRefundHTLCResponse, error) {
	htlc, found := m.store[msg.Id]
	if !found {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, "HTLC not found")
	}
	if htlc.State != types.HTLCStateOpen {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "HTLC not open")
	}
	if time.Now().Unix() > htlc.ExpireTime {
		htlc.State = types.HTLCStateRefunded
		m.store[msg.Id] = htlc
		return &types.MsgRefundHTLCResponse{}, nil
	}
	return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "HTLC not expired")
}
