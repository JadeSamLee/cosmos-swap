package types_test

import (
	"testing"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	// Test HTLCKeyPrefix
	require.Equal(t, []byte{0x01}, types.HTLCKeyPrefix)

	// Test HTLCCountKey
	require.Equal(t, []byte{0x02}, types.HTLCCountKey)
}
