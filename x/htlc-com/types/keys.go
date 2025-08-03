package types

const (
	ModuleName = "htlc"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
	QuerierRoute = ModuleName

	// KeyPrefixHTLC is the prefix for storing HTLCs
	KeyPrefixHTLC = "htlc/"

	// KeyNextHTLCId is the key for storing the next HTLC ID
	KeyNextHTLCId = "next_htlc_id"
)
