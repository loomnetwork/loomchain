# delegated Proof of Stake Contract (v2)

dPoS layers economic incentives on top of an updated PBFT-style consensus
engine underlying tendermint. dappchain nodes register as validator candidates

## Staking

### Candidate Registration

### Delegation

## Election

Loom's dPoS implementation relies on a dynamic set of Validators which
participate in Tendermint's PBFT-style consensus. Though the validator set can
change over time, for any given round of PBFT consensus the validator set is
fixed. A period during which the validator set does not change is called an
`Epoch` and each `Epoch` begins with a Validator Election.

### Election Parameters

`ValidatorCount`: How many validators are elected to participate in Tendermint
consensus

`ElectionCycleLEngth`: How many seconds must elapse between Validator Elections

### Validator Set Changes in `EndBlock`

Whenever an `EndBlockRequest` is received from the Tendermint consensus engine,
the dappchain has the opportunity to submit an array of ValidatorUpdates as an
`EndBlockResponse`. Epochs are much longer than a block time, so the
`LastElectionTime` is tracked in app state and compared to the timestamps
included in every block header. When `ElectionCycleLEngth` has passed, an
Election is run based on the instaneous staking state of the Validators and
Delegators of the chain, and the top `n = ValidatorCount` candidates by
Delegation total are selected to be validators for the next Epoch.

## Rewards

### Rewards Parameters

## Slashing

### Slashing Parameters
