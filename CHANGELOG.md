Pending Release
----------


```

* Leveldb WriteBuffer

This used to be 4 megabytes, now defaults to 500 megabytes

loom.yaml
```yaml
DBBackendConfig:
    WriteBufferMegs: 500
```

## Build 963 - Apr 16th

### Changes
* `gateway` CLI commands no longer accept key strings, a path to the key file must be provided
  instead, this avoids leaving private keys in the terminal history.
* Add ability to disable certain contract loaders.
* Fix a couple of issues in DPOS v2, will be enabled in DPOS v2.1

## Build 934 - Apr 10th

### Changes
* Add Deployer Whitelist contract & middleware to allow selected third party devs to deploy contracts.
* Add Prometheus push metrics, for collecting metrics from nodes behind sentries.
  To enable pushing of metrics add the following to `loom.yaml`:
  ```yaml
  PrometheusPushGateway:
    # Enable publishing via a Prometheus Pushgateway
    Enabled: true
    # host:port or ip:port of the Pushgateway
    PushGateWayUrl: "http://gatewayurl.com"
    # Frequency with which to push metrics to Pushgateway
    PushRateInSeconds: 10
    JobName: "plasmachainmetrics"
  ```
* Add additional validation checks to Transfer Gateway.
* Upgrade to DPOS v2.1, which fixes a couple of issues with lock times & rewards.
* Add a feature flag to move EVM tx receipts storage from app.db to a separate DB.
* Add MigrationTx, will be used to migrate to DPOS v3.
* Make `loom gateway map-accounts --interactive` more user friendly.

## Build 895 - Mar 26th

### Changes
* Add feature flags for chain wide configuration changes, to enable hard forks.
* Add tools for debugging issues with the Dashboard UI & Transfer Gateway.
* Add support for txs signed with Ethereum (secp256k1) keys, and a framework for supporting txs
  signed with keys from other chains.
  - Users will be able to use private keys from Ledger, Trezor, and Metamask to sign DAppChain txs.
  - A hardfork will be required to enable this feature, see below for details.
* New CLI tools for managing validator rewards, with support for offline signing.
* Unsafe RPC endpoints can now be served on a separate interface. To enable this feature add the
  following to `loom.yml`:
  ```yml
  UnsafeRPCEnabled: true # false by default
  UnsafeRPCBindAddress: "tcp://127.0.0.1:26680" # this is the default host:port
  ```
* Add GoLevelDB stats to Prometheus metrics. The new metrics are collected by default, but can be
  disabled in `loom.yml`:
  ```yml
  Metrics:
    Database: false
  ```

### Upcoming hard fork

PlasmChain will need to hard fork to enable support for txs signed with Ethereum keys, to ensure
that all your nodes are prepared please add the following settings to `loom.yml` before upgrading to
build 895:

```yml
ChainConfig:
  ContractEnabled: true
Auth:
  Chains:
    default:
      TxType: "loom"
    eth:
      TxType: "eth"
      AccountType: 1
```

We'll provide additional instructions for initiating the hard fork once everyone has had time to
upgrade to build 895.


## Validator Only Build 833 - Mar 7th

Breaking changes (hard fork):

* Order the validators slightly differently during dpos and this can lead to a inconsistency if a cluster is not upgraded together.


Changes:
* HSM Serialization issues for issue #783
* Fixes for mempool expirations in Tendermint 
* Event indexes for dashboard UI staking, to debug users
* Staking command line tools
* Work towards DPoSV3
* Added method to dump mempool


Config options:
* Mempool evictions

Config.toml
```yaml
[mempool]
tx_life_window = 50 # how many blocks back before a transaction is removed

```

* Leveldb caching

This used to be 8 megabytes, now defaults to 2 gigs 

loom.yaml
```yaml
DBBackendConfig:
    CacheSizeMegs: 2042 #2 Gigabytes
```

* Snapshoted leveldb


loom.yaml
```yaml
# 1 - single mutex NodeDB, 2 - multi-mutex NodeDB
NodeDBVersion: 1
# Number of IAVL tree nodes to cache
NodeCacheSize: 10000
# Snapshot type to use, only supported by MultiReaderIAVL store
# (1 - DB, 2 - DB/IAVL tree, 3 - IAVL tree)
SnapshotVersion: 1
```


## Loom 2.0 Release Build 789 - Feb 12th

Major update, major updates for Performance, Caching and DPoS. It is recommended to upgrade your testnets immediately. 

* DPosV2 now supports Delegates staking and full reward cycle
* Nonce incrementing mid block, loom-js and unity-sdk updated to support
* Work has started on DPoSV3 to have shorter reward cycles and other improvements
* Karma updates
* CheckTx performance boosts
* Updated P2P protocol 
* Go contracts now support event indexing - [Go Events Docs](go-events.html)
* EVM Queries has improved caching layer
* Bug for EVM contracts writing more then 11,800 keys is fixed
* Support for non ETH block chains via TG has started
* More metrics exposed
* YubiHSM Fixes
* Many new config options - [Loom Yaml Configs](loom-yaml.html)



## Loom 2.0 Initial Release Build 651 - Dec 14th

* Loom SDK 2.0
* Updated P2P Protocols
* Blockexplorer now supports EVM
* Loom Native token support for on chain Staking
* Support For delegation rewards in protocol
* Fixes for GetEvmBlockByNumber/Hash
* Range function Prefix fixes for Go Contracts
* HSM local key signing  - [HSM Docs](hsm.html)
* Unity SDK Updated with better network management 

Note Loom SDK 2.0 is not protocol compatible with 1.0 chains. If you have a production chain using 1.0 please contact support@loomx.io for migration help. 

## Release 575 - Nov 16th

* HSM Bug fixes for Yubico Devices  - [HSM Docs](hsm.html)
* HSM now can create new private keys on demand

## Release 575 - Nov 13th

* HSM support for Yubico Devices - [HSM Docs](hsm.html)
* DPoS Version2 beta - for Plasmachain external validators
* Plasma cash massive improvements, see [Plasma Cli](https://github.com/loomnetwork/plasma-cli)
* Pruning for the Datastore, if chain gets to large, see [Config](loom-yaml.html)
* New EVM receipts data store, see [Config](loom-yaml.html)


## Release 478 - Oct 8th

* Updated Transfer Gateway utilities
* Transfer Gateway [tutorial plasma testnet](extdev-transfer-gateway.html)

## Release 458 - Sept 27th

**It is recommended that all users move up to this release**

* Plasmachain Testnets available for Devs - [Docs](testnet-plasma.html)
* Karma Faucet for Testnets - [Faucet](http://faucet.dappchains.com)
* Karma contracts (Sybil resistance) Loom SDK - [Karma](karma.html)
* Massive performance increasements for high load environments
* Initial info about running a Validator - [Validator](validator.html)

## Release 404 - Aug 24th

** Features
* ERC20 support upgraded in TransferGateway
* Example ERC20 in Gateway - [Example](https://github.com/loomnetwork/token-gateway-example)
* Loom SDK Doc site in [Korean](https://loomx.io/developers/ko/)
* EVM now has limited support for Payable functions, more coming next week 
* Loom-JS updates for ERC20/ETH transfers with TransferGateway. 
* Loom-JS integration to DPoS contracts 
* Experimental Support for [Plasma Debits](https://github.com/loomnetwork/plasma-cash/pull/115)

** Breaking changes
* Deprecrating QueryServerHost, and consolidating all functions to the RPCServer.
* New Config option: RPCBindAddress: "tcp://0.0.0.0:46658"
* Upgrades of PBFT engine, this may cause incompatibility issues on upgrades, please try in staging environments 


## Release 375 - Aug 10th

*NOTE* This is a feature test release, with minor compatibility changes, please verify in staging environements before upgrading your production environment.   

* Fixes for Eventing on EVM Contracts
* Beta release of the [Transfer Gateway](transfer-gateway.html)
* [Demo of Transfer Gateway](https://github.com/loomnetwork/cards-gateway-example)
* Memory leak fixes
* Minor api breakages, please upgrade go-loom/loom-js. Unity updates coming soon
* Go-loom is upgraded for api breakages
* loom-js is upgraded for this release. 

## Release 330 - July 30th

* Fix for consensus problems on EVM 
* Updates for Zombiechain TestNet
* Added more telemetry to measure performance
* Range queries on Go Contracts
* Added ChainID to loom.yaml

* [EVM indexed filter](https://loomx.io/developers/docs/en/web3js-event-filters.html)
* EVM filter pool fixes and event system
* Loom-JS EVM updates for indexed filters
* Loom-JS EVM fixes for getting block by hash

## Release 288 - July 17th

* [EVM indexed filter](https://loomx.io/developers/docs/en/web3js-event-filters.html)
* EVM filter pool fixes and event system
* Loom-JS EVM updates for indexed filters
* Loom-JS EVM fixes for getting block by hash

## Release 276 - July 13th

* [New Block Explorer](block-explorer-tutorial.html)
* Multinode EVM fixes 
* Loom-JS updates for Plasma cash
* Zombiechain testnet fixes
* DPoS Updates

## Release 209 - June 20th

Major release
* Plasma Cash initial integration - Demos coming next week
* Multinode fixes and performance increases
* Ansible updates for multinode
* Querying / Filtering on EVM supports more types 
* Unity SDK updates for EVM 
* Loom-JS updates for EVM

## Release 186 - June 19th

* [EVM Unity Example app](https://loomx.io/developers/docs/en/unity-sample-tiles-chain-evm.html)
* Unity SDK support for Solidity Apps
* Many fixes for Filtering/Querying Ethereum Events

## Release 163 - June 11th

* Support for latest Build of Truffle
* [Updated truffle example](https://github.com/loomnetwork/loom-truffle-provider)
* Initial Implementation of Sybil resistance framework
* Websocket events now support topics 
* Loom-JS 1.8.0 Release with updated websocket topic support

## Release 161 - June 7th

* Tons of Truffle Fixes
* Tons of web3.js fixes for Loom-Js provider

## Release 155 - June 6th

* [Cocos SDK is Live](cocos-sdk-quickstart.html)
* [Truffle Support available](truffle-deploy.html)
* Static calls to EVM now allow caller
* EVM Fixes for a lot of scenarios 


## Release 143 - June 1st

* [BluePrint Docker Images Available](docker-blueprint.html)
* [Japanese Hackathon Results](https://medium.com/loom-network/highlights-from-the-first-loom-unity-sdk-hackathon-tokyo-edition-6ed723747c19)
* [Docker Images for some of Loom SDK Projects](https://hub.docker.com/r/loomnetwork/)
* Evm TX Reciepts fixes 


## Release 137 - May 30th

* Go Clients can Access EVM Contracts
* Numerous bug fixes for EVM
* [Social Network Example App - Solidity](simple-social-network-example.html)


## Release 136 - May 28th

* Initial Solidity Alpha test build, you can now deploy solidity contracts
* Websocket eventing support for solidity 
* [Example Project for solidity Events](phaser-sdk-demo-web3-websocket.html)
* [Multinode deployment guide](multi-node-deployment.html)

## Release 133 - May 24th

* [Etherboy Demo released](https://loomx.io/developers/docs/en/etherboy-game.html)
* [Japanese Docs released](https://loomx.io/developers/ja)
* Updated Homepage for [docs site](https://loomx.io/developers/en/) 

## Release 132 - May 23rd 

* Websocket performance fixes
* New Websocket Demo App - TilesChain - [Github](https://github.com/loomnetwork/tiles-chain) 

## Release 129 - May 22rd 

* Websocket support for eventing
* Updates to indexing layer for solidty contracts
* Phaser Game Dame - [Github](https://github.com/loomnetwork/phaser-sdk-demo)

## Release 128 - May 21th

* Lots of bug fixes for Etherboy

## Release 128 - May 19th

* Stable Beta Release
* Updating logging to default to multiple files 
* Moving all RPC to a single interface
* Updated External Process interface

