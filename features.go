package loomchain

// List of feature flags
const (
	// Enables deduping of Mainnet events in the Gateway contract by tx hash.
	TGCheckTxHashFeature = "tg:check-txhash"

	// Enables processing of txs via MultiChainSignatureTxMiddleware, there's a feature flag per
	// allowed chain ID, e.g. auth:sigtx:default, auth:sigtx:eth
	AuthSigTxFeaturePrefix = "auth:sigtx:"

	// Enables DPOS v3
	// NOTE: The DPOS v3 contract must be loaded & deployed first!
	DPOSVersion3Feature = "dpos:v3"

	// Enables deployer whitelist middleware
	// enable deployer whitelist middleware to allow only whitelisted deployers to deploy or migrate contract
	DeployerWhitelistFeature = "mw:deploy-wl"

	// Enables MigrationTx
	// Enables processing of MigrationTx
	MigrationTxFeature = "handler:migration-tx"

	// Enables migration function feature
	// enables processing of migration function
	MigrationFeturePrefix = "migration:"
)
