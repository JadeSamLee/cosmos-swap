package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"
)

const (
	TypeMsgCreateHTLC = "create_htlc"
	TypeMsgClaimHTLC  = "claim_htlc"
	TypeMsgRefundHTLC = "refund_htlc"
)

var (
	_ sdk.Msg = &MsgCreateHTLC{}
	_ sdk.Msg = &MsgClaimHTLC{}
	_ sdk.Msg = &MsgRefundHTLC{}
)

type MsgCreateHTLC struct {
	Sender   sdk.AccAddress `json:"sender" yaml:"sender"`
	Receiver sdk.AccAddress `json:"receiver" yaml:"receiver"`
	Amount   sdk.Coins      `json:"amount" yaml:"amount"`
	HashLock []byte         `json:"hash_lock" yaml:"hash_lock"`
	TimeLock int64          `json:"time_lock" yaml:"time_lock"` // unix timestamp
}

func NewMsgCreateHTLC(sender, receiver sdk.AccAddress, amount sdk.Coins, hashLock []byte, timeLock int64) *MsgCreateHTLC {
	return &MsgCreateHTLC{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
		HashLock: hashLock,
		TimeLock: timeLock,
	}
}

func (msg *MsgCreateHTLC) Route() string { return ModuleName }
func (msg *MsgCreateHTLC) Type() string  { return TypeMsgCreateHTLC }
func (msg *MsgCreateHTLC) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Sender}
}
func (msg *MsgCreateHTLC) GetSignBytes() []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}
func (msg *MsgCreateHTLC) ValidateBasic() error {
	if msg.Sender.Empty() {
		return fmt.Errorf("sender cannot be empty")
	}
	if msg.Receiver.Empty() {
		return fmt.Errorf("receiver cannot be empty")
	}
	if !msg.Amount.IsAllPositive() {
		return fmt.Errorf("amount must be positive")
	}
	if len(msg.HashLock) == 0 {
		return fmt.Errorf("hash lock cannot be empty")
	}
	if msg.TimeLock <= 0 {
		return fmt.Errorf("time lock must be positive unix timestamp")
	}
	return nil
}

type MsgClaimHTLC struct {
	Claimer  sdk.AccAddress `json:"claimer" yaml:"claimer"`
	HTLCId   uint64         `json:"htlc_id" yaml:"htlc_id"`
	Preimage []byte         `json:"preimage" yaml:"preimage"`
}

func NewMsgClaimHTLC(claimer sdk.AccAddress, htlcId uint64, preimage []byte) *MsgClaimHTLC {
	return &MsgClaimHTLC{
		Claimer:  claimer,
		HTLCId:   htlcId,
		Preimage: preimage,
	}
}

func (msg *MsgClaimHTLC) Route() string { return ModuleName }
func (msg *MsgClaimHTLC) Type() string  { return TypeMsgClaimHTLC }
func (msg *MsgClaimHTLC) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Claimer}
}
func (msg *MsgClaimHTLC) GetSignBytes() []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}
func (msg *MsgClaimHTLC) ValidateBasic() error {
	if msg.Claimer.Empty() {
		return fmt.Errorf("claimer cannot be empty")
	}
	if msg.HTLCId == 0 {
		return fmt.Errorf("htlc id cannot be zero")
	}
	if len(msg.Preimage) == 0 {
		return fmt.Errorf("preimage cannot be empty")
	}
	return nil
}

type MsgRefundHTLC struct {
	Refunder sdk.AccAddress `json:"refunder" yaml:"refunder"`
	HTLCId   uint64         `json:"htlc_id" yaml:"htlc_id"`
}

func NewMsgRefundHTLC(refunder sdk.AccAddress, htlcId uint64) *MsgRefundHTLC {
	return &MsgRefundHTLC{
		Refunder: refunder,
		HTLCId:   htlcId,
	}
}

func (msg *MsgRefundHTLC) Route() string { return ModuleName }
func (msg *MsgRefundHTLC) Type() string  { return TypeMsgRefundHTLC }
func (msg *MsgRefundHTLC) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.Refunder}
}
func (msg *MsgRefundHTLC) GetSignBytes() []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}
func (msg *MsgRefundHTLC) ValidateBasic() error {
	if msg.Refunder.Empty() {
		return fmt.Errorf("refunder cannot be empty")
	}
	if msg.HTLCId == 0 {
		return fmt.Errorf("htlc id cannot be zero")
	}
	return nil
}
