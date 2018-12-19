# delegated Proof of Stake Contract (v2)

dPoS layers economic incentives on top of an updated PBFT-style consensus
engine underlying tendermint. dappchain nodes register as validator candidates

## Staking

In delegated Proof-of-Stake, voting power in PBFT-style consensus is determined
by Stake. Stake is equal to a validator's `DelegationTotal` which consists of
all the tokens delegated to the validator, i.e. tokens deposited to the dPoS
contract and assigned to a validator by arbitrary parties called delegators or
by the validator itself.

### Candidate Registration

In order for a would-be validator to receive delegations it must first register
as a `Candidate` and tell potential delegators the following important pieces of
information:

`Candidate address`: the address which the `Candidate` is uniquly identified

`Fee`: the commission, experessed as a percentage in basis points, that the
`Candidate` will take of the rewards he recieves for participating in consensus.

`Name`: the `Candidate`'s human-readable name, a secondary identifier

`Description`: Short piece of information about the `Candidate`

`Website`: URL where additional information about the `Candidate` can be found

#### Registration Parameters

`registrationRequirement`: Quantity in nominal tokens which a would-be validator
must deposit (self-delegate) to the dPoS contract in order to become a canidate
which participates in Elections.

### Delegation

A delegation is a 5-tuple of `(Delegator, Validator, Amount, UpdateAmount, State)`.

Delegations can exist in three distinct states:

`BONDED`: A token delegation has been made from `Delegator` to `Validator`; the
tokens have been transfered to the dPoS contract and the token amount counts
towards `Validator`'s `DelegationTotal` and thus earns rewards for the validator
and all of his delegators. The token delegation is liable to be slashed in case
of faulty behavior from `Validator`. Only when a delegation is the `BONDED`
state can `Delegator` increase his `Delegation` to `Validator` by calling
`Delegate` or decrease his delegation by called `Unbond`.

`BONDING`: New delegated tokens have been received by the dPoS contract but they
will not count toward the validator's `DelegationTotal` until the next validator
election, nor with this new delegation amount earn rewards for the delegator.
Any delegation amount that was previously `BONDED` by the delegator to the
validator continues to earn rewards. The newly delegated tokens are not at risk
of slashing.

`UNBONDING`: A request to withdraw tokens has been submitted by a delegator but
the tokens have not yet been released. The tokens continue to earn rewards for
the delegator and are liable to be slashed until the next valdiator election
when they are automatically transfered to an address which the delegator specifies.

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

## Slashing

### Slashing Parameters

`doubleSignSlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits
a double-sign (equivocation) fault.

`inactivitySlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits an
inactivity (crash) fualt.

## Rewards

Besides disincentivizing deviations from the consensus protocol using slashing,
validator participation is incentivized with rewards.

### Rewards Parameters

`blockRewardPercentage`: Percentage expressed in basis points which a honest
validator should expect his `DelegationTotal` to grow by over the course of
a year.
