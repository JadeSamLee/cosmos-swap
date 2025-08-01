package cli_test

import (
	"testing"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/client/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := cli.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "htlc", cmd.Use)
	require.Len(t, cmd.Commands(), 2)
}

func TestCmdListHTLCs(t *testing.T) {
	cmd := cli.CmdListHTLCs()
	require.NotNil(t, cmd)
	require.Equal(t, "list-htlcs", cmd.Use)
}

func TestCmdShowHTLC(t *testing.T) {
	cmd := cli.CmdShowHTLC()
	require.NotNil(t, cmd)
	require.Equal(t, "show-htlc", cmd.Use)
}
