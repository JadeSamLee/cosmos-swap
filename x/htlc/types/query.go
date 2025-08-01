package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	QueryGetHTLC = "htlc"
	QueryListHTLCs = "htlcs"
)

type QueryGetHTLCRequest struct {
	Id uint64 `json:"id"`
}

type QueryGetHTLCResponse struct {
	HTLC HTLC `json:"htlc"`
}

type QueryListHTLCsRequest struct {}

type QueryListHTLCsResponse struct {
	HTLCs []HTLC `json:"htlcs"`
}
