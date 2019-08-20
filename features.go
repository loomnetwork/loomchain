package loomchain

// List of feature flags
const (
	// Enables deduping of Mainnet events in the Gateway contract by tx hash.
	TGCheckTxHashFeature = "tg:check-txhash"
	// Enables hot wallet (users can submit Ethereum deposit tx hashes).
	TGHotWalletFeature = "tg:hot-wallet"
	// Enables prevention of zero amount token withdrawals in the Gateway contract
	TGCheckZeroAmount = "tg:check-zamt"
	// Enables workaround for handling of ERC721 deposits in the Gateway contract
	TGFixERC721Feature = "tg:fix-erc721"
	// Enables support for Binance contract mappings in the Binance Gateway contract
	TGBinanceContractMappingFeature = "tg:binance-cm"
	// Store Mainnet Gateway address in Gateway Go contract
	TGVersion1_1 = "tg:v1.1"

	// Enables support for mapping DAppChain accounts to Binance accounts
	AddressMapperVersion1_1 = "addrmapper:v1.1"

	// Enables processing of txs via MultiChainSignatureTxMiddleware, there's a feature flag per
	// allowed chain ID, e.g. auth:sigtx:default, auth:sigtx:eth
	AuthSigTxFeaturePrefix = "auth:sigtx:"

	// Enables stricter chain-specific signature verification in MultiChainSignatureTxMiddleware
	MultiChainSigTxMiddlewareVersion1_1 = "mw:mulcsigtx:v1.1"

	// Enables DPOS v3
	// NOTE: The DPOS v3 contract must be loaded & deployed first!
	DPOSVersion3Feature = "dpos:v3"
	// Enables precise rewards calculations in DPOSv3
	DPOSVersion3_1 = "dpos:v3.1"
	// Enables slashing metrics
	DPOSVersion3_2 = "dpos:v3.2"
	// Enables jailing offline validators
	DPOSVersion3_3 = "dpos:v3.3"
	// Enables both downtime slashing and a parameter flag to toggle jailing offline validators on/off
	DPOSVersion3_4 = "dpos:v3.4"
	// Fixes prefixing of referrer keys so that ListReferrers method works
	DPOSVersion3_5 = "dpos:v3.5"

	// Enables rewards to be distributed even when a delegator owns less than 0.01% of the validator's stake
	// Also makes whitelists give bonuses correctly if whitelist locktime tier is set to be 0-3 (else defaults to 5%)
	DPOSVersion2_1 = "dpos:v2.1"

	// Enables EVM tx receipts storage in separate DB.
	EvmTxReceiptsVersion2Feature = "receipts:v2"

	// Enables deployer whitelist middleware that only allows whitelisted accounts to
	// deploy contracts & run migrations.
	DeployerWhitelistFeature = "mw:deploy-wl"

	// Enables post commit middleware for user-deployer-whitelist
	UserDeployerWhitelistFeature = "mw:userdeploy-wl"

	// Enables block range & max txs fields in tier info stored in User Deployer Whitelist contract
	UserDeployerWhitelistVersion1_1Feature = "userdeploy-wl:v1.1"

	// Makes UserDeployerWhitelist.RemoveUserDeployer mark deployer accounts as inactive instead of
	// deleting them.
	UserDeployerWhitelistVersion1_2Feature = "userdeploy-wl:v1.2"

	// Enables processing of MigrationTx.
	MigrationTxFeature = "tx:migration"

	// Enables specific migrations, each migration has an ID that's prefixed by this string.
	MigrationFeaturePrefix = "migration:"

	// Enables usage of ctx.Validators() in ChainConfig contract.
	ChainCfgVersion1_1 = "chaincfg:v1.1"

	// Enables validator build number tracking via the ChainConfig contract.
	ChainCfgVersion1_2 = "chaincfg:v1.2"

	// Enables config setting in the ChainConfig contract.
	ChainCfgVersion1_3 = "chaincfg:v1.3"

	// Forces the MultiWriterAppStore to write EVM state only to evm.db, otherwise it'll write EVM
	// state to both evm.db & app.db.
	EvmDBFeature = "db:evm"

	// Enables Coin v1.1 contract (also applies to ETHCoin)
	CoinVersion1_1Feature = "coin:v1.1"

	// Enables Coin v1.2 to validate fields in request of Coin and ETH Coin contract
	CoinVersion1_2Feature = "coin:v1.2"

	// Force ReceiptHandler to write BloomFilter and EVM TxHash only to receipts_db, otherwise it'll
	// write BloomFilter and EVM TxHash to both receipts_db & app.db.
	// This feature has been deprecated along with legacy code.
	AuxEvmDBFeature = "db:auxevm"
	// Force MultiWriterAppStore to write EVM root to app.db only if the root changes
	AppStoreVersion3_1 = "appstore:v3.1"

	// Enable option to allow checking the registry error
	DeployTxVersion1_1Feature = "deploytx:v1.1"

	// Restrict the value of call & deploy txs to non-negative amounts
	CheckTxValueFeature = "tx:check-value"

	// Enables Constantinople hard fork in EVM interpreter
	EvmConstantinopleFeature = "evm:constantinople"
)
