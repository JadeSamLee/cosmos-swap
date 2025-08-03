package cli_test

import (
	"testing"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/client/cli"
	"github.com/stretchr/testify/require"
)

func TestQueryCmds(t *testing.T) {
	// Test that the query commands are created correctly
	listCmd := cli.CmdListHTLCs()
	require.NotNil(t, listCmd)
	require.Equal(t, "list-htlcs", listCmd.Use)

	showCmd := cli.CmdShowHTLC()
	require.NotNil(t, showCmd)
	require.Equal(t, "show-htlc", showCmd.Use)
}
