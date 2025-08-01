package types_test

import (
	"testing"
	"time"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	genesis := types.DefaultGenesis()
	err := genesis.Validate()
	require.NoError(t, err)

	htlc := types.HTLC{
		Id:       1,
		Sender:   nil,
		Receiver: nil,
		Amount:   nil,
		HashLock: []byte{},
		TimeLock: time.Now(),
		Claimed:  false,
		Refunded: false,
	}
	genesis.HTLCs = append(genesis.HTLCs, htlc)
	err = genesis.Validate()
	require.NoError(t, err)
}
