package types_test

import (
	"testing"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/stretchr/testify/require"
)

func TestMsgCreateHTLC_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  types.MsgCreateHTLC
		err  error
	}{
		{
			name: "invalid sender",
			msg: types.MsgCreateHTLC{
				Sender:   []byte{},
				Receiver: []byte("receiver"),
				Amount:   nil,
				HashLock: []byte("hashlock"),
				TimeLock: time.Now().Add(time.Hour).Unix(),
			},
			err: types.ErrInvalidSender,
		},
		{
			name: "invalid receiver",
			msg: types.MsgCreateHTLC{
				Sender:   []byte("sender"),
				Receiver: []byte{},
				Amount:   nil,
				HashLock: []byte("hashlock"),
				TimeLock: time.Now().Add(time.Hour).Unix(),
			},
			err: types.ErrInvalidReceiver,
		},
		{
			name: "empty hash lock",
			msg: types.MsgCreateHTLC{
				Sender:   []byte("sender"),
				Receiver: []byte("receiver"),
				Amount:   nil,
				HashLock: []byte{},
				TimeLock: time.Now().Add(time.Hour).Unix(),
			},
			err: types.ErrInvalidHashLock,
		},
		{
			name: "invalid hash lock length",
			msg: types.MsgCreateHTLC{
				Sender:   []byte("sender"),
				Receiver: []byte("receiver"),
				Amount:   nil,
				HashLock: []byte("short"),
				TimeLock: time.Now().Add(time.Hour).Unix(),
			},
			err: types.ErrInvalidHashLock,
		},
		{
			name: "invalid time lock",
			msg: types.MsgCreateHTLC{
				Sender:   []byte("sender"),
				Receiver: []byte("receiver"),
				Amount:   nil,
				HashLock: []byte("hashlockhashlockhashlockhashlock"),
				TimeLock: 0,
			},
			err: types.ErrInvalidTimeLock,
		},
		{
			name: "valid message",
			msg: types.MsgCreateHTLC{
				Sender:   []byte("sender"),
				Receiver: []byte("receiver"),
				Amount:   nil,
				HashLock: []byte("hashlockhashlockhashlockhashlock"),
				TimeLock: time.Now().Add(time.Hour).Unix(),
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgClaimHTLC_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  types.MsgClaimHTLC
		err  error
	}{
		{
			name: "invalid claimer",
			msg: types.MsgClaimHTLC{
				Claimer:  []byte{},
				HTLCId:   1,
				Preimage: []byte("preimage"),
			},
			err: types.ErrInvalidClaimer,
		},
		{
			name: "invalid htlc id",
			msg: types.MsgClaimHTLC{
				Claimer:  []byte("claimer"),
				HTLCId:   0,
				Preimage: []byte("preimage"),
			},
			err: types.ErrInvalidHTLCID,
		},
		{
			name: "empty preimage",
			msg: types.MsgClaimHTLC{
				Claimer:  []byte("claimer"),
				HTLCId:   1,
				Preimage: []byte{},
			},
			err: types.ErrInvalidPreimage,
		},
		{
			name: "valid message",
			msg: types.MsgClaimHTLC{
				Claimer:  []byte("claimer"),
				HTLCId:   1,
				Preimage: []byte("preimage"),
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgRefundHTLC_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  types.MsgRefundHTLC
		err  error
	}{
		{
			name: "invalid refunder",
			msg: types.MsgRefundHTLC{
				Refunder: []byte{},
				HTLCId:   1,
			},
			err: types.ErrInvalidRefunder,
		},
		{
			name: "invalid htlc id",
			msg: types.MsgRefundHTLC{
				Refunder: []byte("refunder"),
				HTLCId:   0,
			},
			err: types.ErrInvalidHTLCID,
		},
		{
			name: "valid message",
			msg: types.MsgRefundHTLC{
				Refunder: []byte("refunder"),
				HTLCId:   1,
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
