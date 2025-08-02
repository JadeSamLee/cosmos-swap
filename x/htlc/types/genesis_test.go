package types_test

import (
	"testing"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				HTLCs: []types.HTLC{
					{
						Id:       1,
						Sender:   []byte("sender"),
						Receiver: []byte("receiver"),
						Amount:   nil,
						HashLock: []byte("hashlock"),
						TimeLock: time.Now().Add(time.Hour),
						Claimed:  false,
						Refunded: false,
					},
				},
			},
			valid: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
