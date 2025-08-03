package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
)

// ModuleCdc references the global x/htlc module codec. Note, the codec should
// only be used in certain instances of tests and for JSON encoding.
var ModuleCdc = codec.NewLegacyAmino()

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateHTLC{}, "htlc/CreateHTLC", nil)
	cdc.RegisterConcrete(&MsgClaimHTLC{}, "htlc/ClaimHTLC", nil)
	cdc.RegisterConcrete(&MsgRefundHTLC{}, "htlc/RefundHTLC", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCreateHTLC{},
		&MsgClaimHTLC{},
		&MsgRefundHTLC{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
