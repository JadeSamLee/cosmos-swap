package cli

import (
	"fmt"
	"strconv"
	"time"

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
		Long: `Create a new HTLC (Hashed Time-Locked Contract) with the specified parameters.
		
Arguments:
  [receiver]  The address of the receiver who can claim the HTLC
  [amount]    The amount of coins to lock in the HTLC
  [hashlock]  The SHA256 hash of the preimage (32 bytes in hex)
  [timelock]  The Unix timestamp when the HTLC expires and can be refunded
		
Example:
  create-htlc cosmos1... 1000stake 0x1234567890abcdef... 1620000000`,
		Args: cobra.ExactArgs(4),
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

			// Validate hashLock length
			if len(hashLock) != 32 {
				return fmt.Errorf("hashLock must be 32 bytes (SHA256 hash)")
			}

			timeLock, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return err
			}

			// Validate timeLock is in the future
			currentTime := time.Now().Unix()
			if timeLock <= currentTime {
				return fmt.Errorf("timeLock must be in the future")
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
		Long: `Claim an HTLC by providing the preimage that matches the hash lock.
		
Arguments:
  [htlc-id]   The ID of the HTLC to claim
  [preimage]  The preimage that matches the hash lock of the HTLC
		
Example:
  claim-htlc 1 0xabcdef1234567890...`,
		Args: cobra.ExactArgs(2),
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

			// Validate preimage is not empty
			if len(preimage) == 0 {
				return fmt.Errorf("preimage cannot be empty")
			}

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
		Long: `Refund an HTLC after the time lock has expired.
		
Arguments:
  [htlc-id]  The ID of the HTLC to refund
		
Example:
  refund-htlc 1`,
		Args: cobra.ExactArgs(1),
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
