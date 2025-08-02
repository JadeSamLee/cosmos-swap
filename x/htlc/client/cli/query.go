package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s query commands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdListHTLCs())
	cmd.AddCommand(CmdShowHTLC())

	return cmd
}

func CmdListHTLCs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-htlcs",
		Short: "List all HTLCs",
		Long:  "List all HTLCs in the network",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.HTLCs(context.Background(), &types.QueryListHTLCsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowHTLC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-htlc [id]",
		Short: "Show a HTLC",
		Long:  "Show details of a specific HTLC by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.HTLC(context.Background(), &types.QueryGetHTLCRequest{Id: id})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
