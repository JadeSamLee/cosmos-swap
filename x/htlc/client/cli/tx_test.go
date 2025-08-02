package cli_test

import (
	"testing"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/client/cli"
	"github.com/stretchr/testify/require"
)

func TestTxCmds(t *testing.T) {
	// Test that the transaction commands are created correctly
	createCmd := cli.CmdCreateHTLC()
	require.NotNil(t, createCmd)
	require.Equal(t, "create-htlc", createCmd.Use)

	claimCmd := cli.CmdClaimHTLC()
	require.NotNil(t, claimCmd)
	require.Equal(t, "claim-htlc", claimCmd.Use)

	refundCmd := cli.CmdRefundHTLC()
	require.NotNil(t, refundCmd)
	require.Equal(t, "refund-htlc", refundCmd.Use)
}
