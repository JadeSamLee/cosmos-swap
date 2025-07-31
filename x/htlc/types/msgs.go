type MsgCreateHTLC struct {
    Sender    sdk.AccAddress `json:"sender" yaml:"sender"`
    Recipient sdk.AccAddress `json:"recipient" yaml:"recipient"`
    Amount    sdk.Coins      `json:"amount" yaml:"amount"`
    HashLock  []byte         `json:"hash_lock" yaml:"hash_lock"`
    MerkleRoot []byte        `json:"merkle_root" yaml:"merkle_root"`
    TimeLock  int64          `json:"time_lock" yaml:"time_lock"` // block height
}

// MsgClaimHTLC defines a message to claim an existing HTLC by revealing the secret
type MsgClaimHTLC struct {
    Claimer sdk.AccAddress `json:"claimer" yaml:"claimer"`
    HTLCId  []byte         `json:"htlc_id" yaml:"htlc_id"`
    Secret  []byte         `json:"secret" yaml:"secret"`
}

// Route returns the name of the module
func (msg MsgClaimHTLC) Route() string { return RouterKey }

// Type returns the action name
func (msg MsgClaimHTLC) Type() string { return "claim_htlc" }

// ValidateBasic performs stateless checks
func (msg MsgClaimHTLC) ValidateBasic() error {
    if msg.Claimer.Empty() {
        return errors.Wrap(errors.ErrInvalidAddress, "claimer cannot be empty")
    }
    if len(msg.HTLCId) == 0 {
        return fmt.Errorf("htlc_id cannot be empty")
    }
    if len(msg.Secret) == 0 {
        return fmt.Errorf("secret cannot be empty")
    }
    return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgClaimHTLC) GetSignBytes() []byte {
    bz := ModuleCdc.MustMarshalJSON(&msg)
    return sdk.MustSortJSON(bz)
}

// GetSigners defines whose signature is required
func (msg MsgClaimHTLC) GetSigners() []sdk.AccAddress {
    return []sdk.AccAddress{msg.Claimer}
}

// MsgRefundHTLC defines a message to refund an existing HTLC after timelock expiry
type MsgRefundHTLC struct {
    Sender sdk.AccAddress `json:"sender" yaml:"sender"`
    HTLCId []byte         `json:"htlc_id" yaml:"htlc_id"`
}

// Route returns the name of the module
func (msg MsgRefundHTLC) Route() string { return RouterKey }

// Type returns the action name
func (msg MsgRefundHTLC) Type() string { return "refund_htlc" }

// ValidateBasic performs stateless checks
func (msg MsgRefundHTLC) ValidateBasic() error {
    if msg.Sender.Empty() {
        return errors.Wrap(errors.ErrInvalidAddress, "sender cannot be empty")
    }
    if len(msg.HTLCId) == 0 {
        return fmt.Errorf("htlc_id cannot be empty")
    }
    return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRefundHTLC) GetSignBytes() []byte {
    bz := ModuleCdc.MustMarshalJSON(&msg)
    return sdk.MustSortJSON(bz)
}

// GetSigners defines whose signature is required
func (msg MsgRefundHTLC) GetSigners() []sdk.AccAddress {
    return []sdk.AccAddress{msg.Sender}
}

// Register interfaces for protobuf
func RegisterInterfaces(registry types.InterfaceRegistry) {
    registry.RegisterImplementations((*sdk.Msg)(nil), &MsgCreateHTLC{}, &MsgClaimHTLC{}, &MsgRefundHTLC{})
}
