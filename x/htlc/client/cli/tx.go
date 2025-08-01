package cli

import (
	"fmt"
	"strconv"

	"github.com/crypto-org-chain/cronos/v2/x/htlc/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdCreateHTLC())
	cmd.AddCommand(CmdClaimHTLC())
	cmd.AddCommand(CmdRefundHTLC())

	return cmd
}

func CmdCreateHTLC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-htlc [receiver] [amount] [hashlock] [timelock]",
		Short: "Create a new HTLC",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			receiver, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinsNormalized(args[1])
			if err != nil {
				return err
			}

			hashLock := []byte(args[2])

			timeLock, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateHTLC(clientCtx.GetFromAddress(), receiver, amount, hashLock, timeLock)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdClaimHTLC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-htlc [htlc-id] [preimage]",
		Short: "Claim an HTLC",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			htlcId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			preimage := []byte(args[1])

			msg := types.NewMsgClaimHTLC(clientCtx.GetFromAddress(), htlcId, preimage)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdRefundHTLC() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund-htlc [htlc-id]",
		Short: "Refund an HTLC",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			htlcId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			msg := types.NewMsgRefundHTLC(clientCtx.GetFromAddress(), htlcId)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
