package htlc

import (
    "encoding/json"

    "github.com/cosmos/cosmos-sdk/client"
    "github.com/cosmos/cosmos-sdk/codec"
    "github.com/cosmos/cosmos-sdk/codec/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/types/module"

    "github.com/interchainx/x/htlc/keeper"
    "github.com/interchainx/x/htlc/types"
)

// AppModuleBasic defines the basic application module used by the htlc module.
type AppModuleBasic struct{}

// Name returns the htlc module's name.
func (AppModuleBasic) Name() string {
    return types.ModuleName
}

// RegisterCodec registers the module's types for the amino codec.
func (AppModuleBasic) RegisterCodec(cdc *codec.LegacyAmino) {
    types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (AppModuleBasic) RegisterInterfaces(registry types.InterfaceRegistry) {
    types.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the htlc module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
    return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the htlc module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
    var data types.GenesisState
    if err := cdc.UnmarshalJSON(bz, &data); err != nil {
        return err
    }
    return types.ValidateGenesis(data)
}

// AppModule implements an application module for the htlc module.
type AppModule struct {
    AppModuleBasic
    keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(k keeper.Keeper) AppModule {
    return AppModule{
        AppModuleBasic: AppModuleBasic{},
        keeper:         k,
    }
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
    types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
    // Register other services like query server here if needed
}

// BeginBlocker processes expired HTLCs at the beginning of each block
func (am AppModule) BeginBlocker(ctx sdk.Context) {
    am.keeper.ExpireHTLCs(ctx)
}
