package cli_test

import (
	"testing"
	"strconv"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/client/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetTxCmd(t *testing.T) {
	cmd := cli.GetTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "htlc", cmd.Use)
	require.Len(t, cmd.Commands(), 3)
}

func TestCmdCreateHTLC(t *testing.T) {
	cmd := cli.CmdCreateHTLC()
	require.NotNil(t, cmd)
	require.Equal(t, "create-htlc", cmd.Use)
	require.Equal(t, 4, cmd.ArgsLen())
}

func TestCmdClaimHTLC(t *testing.T) {
	cmd := cli.CmdClaimHTLC()
	require.NotNil(t, cmd)
	require.Equal(t, "claim-htlc", cmd.Use)
	require.Equal(t, 2, cmd.ArgsLen())
}

func TestCmdRefundHTLC(t *testing.T) {
	cmd := cli.CmdRefundHTLC()
	require.NotNil(t, cmd)
	require.Equal(t, "refund-htlc", cmd.Use)
	require.Equal(t, 1, cmd.ArgsLen())
}
