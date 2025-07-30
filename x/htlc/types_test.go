package htlc_test

import (
    "testing"
    "github.com/stretchr/testify/require"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/interchainx/x/htlc/types"
)

func TestMsgCreateHTLC_ValidateBasic(t *testing.T) {
    validAddr := sdk.AccAddress("addr1_______________")
    validCoins := sdk.NewCoins(sdk.NewInt64Coin("token", 100))

    tests := []struct {
        name string
        msg  types.MsgCreateHTLC
        wantErr bool
    }{
        {
            name: "valid message",
            msg: types.MsgCreateHTLC{
                Sender: validAddr,
                Recipient: validAddr,
                Amount: validCoins,
                HashLock: []byte{0x01, 0x02},
                MerkleRoot: []byte{},
                TimeLock: 10,
            },
            wantErr: false,
        },
        {
            name: "empty sender",
            msg: types.MsgCreateHTLC{
                Sender: nil,
                Recipient: validAddr,
                Amount: validCoins,
                HashLock: []byte{0x01},
                TimeLock: 10,
            },
            wantErr: true,
        },
        {
            name: "empty recipient",
            msg: types.MsgCreateHTLC{
                Sender: validAddr,
                Recipient: nil,
                Amount: validCoins,
                HashLock: []byte{0x01},
                TimeLock: 10,
            },
            wantErr: true,
        },
        {
            name: "empty hashlock",
            msg: types.MsgCreateHTLC{
                Sender: validAddr,
                Recipient: validAddr,
                Amount: validCoins,
                HashLock: []byte{},
                TimeLock: 10,
            },
            wantErr: true,
        },
        {
            name: "non-positive timelock",
            msg: types.MsgCreateHTLC{
                Sender: validAddr,
                Recipient: validAddr,
                Amount: validCoins,
                HashLock: []byte{0x01},
                TimeLock: 0,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.msg.ValidateBasic()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestMsgClaimHTLC_ValidateBasic(t *testing.T) {
    validAddr := sdk.AccAddress("addr1_______________")

    tests := []struct {
        name string
        msg  types.MsgClaimHTLC
        wantErr bool
    }{
        {
            name: "valid message",
            msg: types.MsgClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{0x01},
                Secret: []byte{0x02},
            },
            wantErr: false,
        },
        {
            name: "empty claimer",
            msg: types.MsgClaimHTLC{
                Claimer: nil,
                HTLCId: []byte{0x01},
                Secret: []byte{0x02},
            },
            wantErr: true,
        },
        {
            name: "empty htlc id",
            msg: types.MsgClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{},
                Secret: []byte{0x02},
            },
            wantErr: true,
        },
        {
            name: "empty secret",
            msg: types.MsgClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{0x01},
                Secret: []byte{},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.msg.ValidateBasic()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestMsgRefundHTLC_ValidateBasic(t *testing.T) {
    validAddr := sdk.AccAddress("addr1_______________")

    tests := []struct {
        name string
        msg  types.MsgRefundHTLC
        wantErr bool
    }{
        {
            name: "valid message",
            msg: types.MsgRefundHTLC{
                Sender: validAddr,
                HTLCId: []byte{0x01},
            },
            wantErr: false,
        },
        {
            name: "empty sender",
            msg: types.MsgRefundHTLC{
                Sender: nil,
                HTLCId: []byte{0x01},
            },
            wantErr: true,
        },
        {
            name: "empty htlc id",
            msg: types.MsgRefundHTLC{
                Sender: validAddr,
                HTLCId: []byte{},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.msg.ValidateBasic()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}

func TestMsgPartialClaimHTLC_ValidateBasic(t *testing.T) {
    validAddr := sdk.AccAddress("addr1_______________")

    tests := []struct {
        name string
        msg  types.MsgPartialClaimHTLC
        wantErr bool
    }{
        {
            name: "valid message",
            msg: types.MsgPartialClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{0x01},
                Secret: []byte{0x02},
                MerkleProof: [][]byte{{0x03}, {0x04}},
            },
            wantErr: false,
        },
        {
            name: "empty claimer",
            msg: types.MsgPartialClaimHTLC{
                Claimer: nil,
                HTLCId: []byte{0x01},
                Secret: []byte{0x02},
                MerkleProof: [][]byte{{0x03}},
            },
            wantErr: true,
        },
        {
            name: "empty htlc id",
            msg: types.MsgPartialClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{},
                Secret: []byte{0x02},
                MerkleProof: [][]byte{{0x03}},
            },
            wantErr: true,
        },
        {
            name: "empty secret",
            msg: types.MsgPartialClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{0x01},
                Secret: []byte{},
                MerkleProof: [][]byte{{0x03}},
            },
            wantErr: true,
        },
        {
            name: "empty merkle proof",
            msg: types.MsgPartialClaimHTLC{
                Claimer: validAddr,
                HTLCId: []byte{0x01},
                Secret: []byte{0x02},
                MerkleProof: [][]byte{},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.msg.ValidateBasic()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
