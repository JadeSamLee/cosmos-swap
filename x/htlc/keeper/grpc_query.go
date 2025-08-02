package keeper

import (
	"context"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
)
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (q queryServer) HTLC(c context.Context, req *types.QueryGetHTLCRequest) (*types.QueryGetHTLCResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	htlc, found := q.GetHTLC(ctx, req.Id)
	if !found {
		return nil, types.ErrHTLCNotFound
	}
	return &types.QueryGetHTLCResponse{HTLC: htlc}, nil
}

func (q queryServer) HTLCs(c context.Context, req *types.QueryListHTLCsRequest) (*types.QueryListHTLCsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(q.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.KeyPrefixHTLC))
	defer iterator.Close()

	var htlcs []types.HTLC
	for ; iterator.Valid(); iterator.Next() {
		var htlc types.HTLC
		q.cdc.MustUnmarshal(iterator.Value(), &htlc)
		htlcs = append(htlcs, htlc)
	}

	return &types.QueryListHTLCsResponse{HTLCs: htlcs}, nil
}
