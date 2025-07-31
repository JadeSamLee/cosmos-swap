package htlc

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"

    sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

    "github.com/interchainx/x/htlc/types"
)

// NewHandler returns a handler for "htlc" type messages.
func NewHandler(k Keeper, bankKeeper types.BankKeeper) sdk.Handler {
    return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
        switch msg := msg.(type) {
        case *types.MsgCreateHTLC:
            return handleMsgCreateHTLC(ctx, k, bankKeeper, msg)
        case *types.MsgClaimHTLC:
            return handleMsgClaimHTLC(ctx, k, bankKeeper, msg)
        case *types.MsgRefundHTLC:
            return handleMsgRefundHTLC(ctx, k, bankKeeper, msg)
        case *types.MsgPartialClaimHTLC:
            return handleMsgPartialClaimHTLC(ctx, k, bankKeeper, msg)
        default:
            return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized htlc message type: %T", msg)
        }
    }
}

func handleMsgCreateHTLC(ctx sdk.Context, k Keeper, bankKeeper types.BankKeeper, msg *types.MsgCreateHTLC) (*sdk.Result, error) {
    if err := msg.ValidateBasic(); err != nil {
        return nil, err
    }

    err := bankKeeper.SendCoinsFromAccountToModule(ctx, msg.Sender, types.ModuleName, msg.Amount)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
    }

    htlc := types.NewHTLC(msg.Sender, msg.Recipient, msg.Amount, msg.HashLock, msg.MerkleRoot, msg.TimeLock)

    err = k.SetHTLC(ctx, htlc)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, fmt.Sprintf("failed to store HTLC: %s", err.Error()))
    }

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeCreateHTLC,
            sdk.NewAttribute(types.AttributeKeySender, msg.Sender.String()),
            sdk.NewAttribute(types.AttributeKeyRecipient, msg.Recipient.String()),
            sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
            sdk.NewAttribute(types.AttributeKeyHashLock, fmt.Sprintf("%X", msg.HashLock)),
            sdk.NewAttribute(types.AttributeKeyMerkleRoot, fmt.Sprintf("%X", msg.MerkleRoot)),
            sdk.NewAttribute(types.AttributeKeyTimeLock, fmt.Sprintf("%d", msg.TimeLock)),
        ),
    )

    return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func handleMsgClaimHTLC(ctx sdk.Context, k Keeper, bankKeeper types.BankKeeper, msg *types.MsgClaimHTLC) (*sdk.Result, error) {
    if err := msg.ValidateBasic(); err != nil {
        return nil, err
    }

    htlc, found, err := k.GetHTLC(ctx, msg.HTLCId)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
    }
    if !found {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "htlc not found")
    }

    // Verify secret against hashlock
    computedHash := types.ComputeHash(msg.Secret)
    if !equalBytes(computedHash, htlc.HashLock) {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "invalid secret")
    }

    // Transfer tokens from module to recipient
    err = bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Recipient, htlc.Amount)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
    }

    // Mark HTLC as claimed
    htlc.SecretRevealed = true
    err = k.SetHTLC(ctx, htlc)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
    }

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeClaimHTLC,
            sdk.NewAttribute(types.AttributeKeyClaimer, msg.Claimer.String()),
            sdk.NewAttribute(types.AttributeKeyRecipient, htlc.Recipient.String()),
            sdk.NewAttribute(types.AttributeKeyHashLock, fmt.Sprintf("%X", htlc.HashLock)),
        ),
    )

    return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func handleMsgRefundHTLC(ctx sdk.Context, k Keeper, bankKeeper types.BankKeeper, msg *types.MsgRefundHTLC) (*sdk.Result, error) {
    if err := msg.ValidateBasic(); err != nil {
        return nil, err
    }

    htlc, found, err := k.GetHTLC(ctx, msg.HTLCId)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
    }
    if !found {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "htlc not found")
    }

    // Check timelock expiration
    if ctx.BlockHeight() < htlc.TimeLock {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "timelock has not expired")
    }

    // Check if HTLC already claimed
    if htlc.SecretRevealed {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "htlc already claimed")
    }

    // Transfer tokens from module back to sender
    err = bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, htlc.Sender, htlc.Amount)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, err.Error())
    }

    // Mark HTLC as refunded (could add a status field if needed)
    // For now, mark SecretRevealed true to indicate closed
    htlc.SecretRevealed = true
    err = k.SetHTLC(ctx, htlc)
    if err != nil {
        return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
    }

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeRefundHTLC,
            sdk.NewAttribute(types.AttributeKeySender, msg.Sender.String()),
            sdk.NewAttribute(types.AttributeKeyHashLock, fmt.Sprintf("%X", htlc.HashLock)),
        ),
    )

    return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}

func equalBytes(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}
</create_file>
