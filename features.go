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

	// Enables rewards to be distributed even when a delegator owns less than 0.01% of the validator's stake
	// Also makes whitelists give bonuses correctly if whitelist locktime tier is set to be 0-3 (else defaults to 5%)
	DPOSVersion2_1 = "dpos:v2.1"

	// Enables EVM tx receipts storage in separate DB.
	EvmTxReceiptsVersion2Feature = "receipts:v2"

	// Enables deployer whitelist middleware that only allows whitelisted accounts to
	// deploy contracts & run migrations.
	DeployerWhitelistFeature = "mw:deploy-wl"

	// Enables processing of MigrationTx.
	MigrationTxFeature = "tx:migration"

	// Enables specific migrations, each migration has an ID that's prefixed by this string.
	MigrationFeaturePrefix = "migration:"

	// Enables usage of ctx.Validators() in ChainConfig contract.
	ChainCfgVersion1_1 = "chaincfg:v1.1"
)
