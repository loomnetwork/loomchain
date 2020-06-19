# delegated Proof of Stake Contract (v2)

dPoS layers economic incentives on top of an updated PBFT-style consensus
engine underlying Tendermint. Nodes register as validator candidates.

## Staking

In delegated Proof-of-Stake, voting power in PBFT-style consensus is determined
by stake. The stake is equal to a validator's `DelegationTotal` which consists of
all the tokens delegated to the validator, i.e. tokens deposited to the dPoS
contract and assigned to a validator by arbitrary parties called delegators or
by the validator itself.

### Candidate Registration

For a would-be validator to receive delegations, it must first register
as a `Candidate` and tell potential delegators the following important pieces of
information:

* `Candidate address`: the address which uniquely identifes a `Candidate`
* `Fee`: the commission, expressed as a percentage in basis points, that the `Candidate` will take of the rewards he receives for participating in consensus.
* `Name`: the `Candidate`'s human-readable name, a secondary identifier
* `Description`: Short piece of information about the `Candidate`
* `Website`: URL where additional information about the `Candidate` can be found

#### Registration Parameters

`registrationRequirement`: Quantity in nominal tokens which a would-be validator
must deposit (self-delegate) to the dPoS contract to become a candidate
which participates in Elections.

### Delegation

A delegation from a delegator to a validator happens in-protocol so delegators do
not have to trust that a validator will return their tokens. The delegator can
unbond a delegation at any time without asking for the validator's consent.

A delegation is a 5-tuple of `(Delegator, Validator, Amount, UpdateAmount, State)`.

Delegations can exist in four distinct states:

1. `BONDED`: A token delegation has been made from `Delegator` to `Validator`; the
tokens have been transferred to the dPoS contract and the token amount counts
towards the `Validator`'s `DelegationTotal` and thus earns rewards for the validator
and all of his delegators. The token delegation is liable to be slashed in case
of faulty behavior from `Validator`. Only when a delegation is the `BONDED`
state can a `Delegator` increase his `Delegation` to a `Validator` by calling
`Delegate` or decrease his delegation by calling `Unbond`.

2. `BONDING`: New delegated tokens have been received by the dPoS contract but they
will not count toward the validator's `DelegationTotal` until the next validator
election, nor with this new delegation amount earn rewards for the delegator.
Any delegation amount that was previously `BONDED` by the delegator to the
validator continues to earn rewards. The newly delegated tokens are not at risk
of slashing.

3. `UNBONDING`: A request to withdraw tokens has been submitted by a delegator but
the tokens have not yet been released. The tokens continue to earn rewards for
the delegator and are liable to be slashed until the next validator election,
when they are automatically transferred to an address which the delegator specifies.

4. `REDELEGATING`: A redelegation request has been made within the last election
period. During the next election, the `delegation.Validator`  value will be set
to the `delegation.UpdateValidator`.

## Election

Loom's dPoS implementation relies on a dynamic set of Validators that
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
the chain has the opportunity to submit an array of ValidatorUpdates as an
`EndBlockResponse`. Epochs are much longer than a block time, so the
`LastElectionTime` is tracked in app state and compared to the timestamps
included in every block header. When `ElectionCycleLEngth` has passed, an election is run based on the instantaneous staking state of the Validators and
Delegators of the chain, and the top `n = ValidatorCount` candidates by
Delegation total is selected to be validators for the next Epoch.

## Slashing

To disincentivize dishonest behavior, a validator's `DelegationTotal`
is liable for penalties or "slashes" if the validator commits a fault. There are
two general categories of fault:

1. `Crash Fault`: Failing to send or receive messages from other nodes, usually due
to being offline

2. `Byzantine Fault`: Arbitrary deviation from the consensus protocol including
signing different blocks at the same block height

### Slashing Parameters

`doubleSignSlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits
a double-sign (equivocation) fault.

`inactivitySlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits an
inactivity (crash) fault.

In any given election period, a validator will not be slashed more than
`inactivitySlashPercentage + doubleSignSlashPercentage`, and this will only
occur if, first, the validator commits an inactivity fault and later commits
a double sign fault whose penalty is greater than that of an inactivity fault.

### Slashing Implementation

Slashing calculations are carried out in `plugin/validators_manager.go` and not
in the dPoS contract itself.

## Rewards

Besides disincentivizing deviations from the consensus protocol using slashing,
validator participation is incentivized with rewards. As long as a validator
does not commit any faults, i.e. participates in consensus properly, the
validator is rewarded.

### Rewards Parameters

`blockRewardPercentage`: Percentage expressed in basis points which an honest
validator should expect his `DelegationTotal` to grow by over the course of
a year.

`maxYearlyRewards`: No election can result in the distribution of more than
(max_yearly_rewards * (election_cycle / year)). This value is set manually by
the oracle

### Delegator Rewards Distribution

After a Validator's fee has been removed from the total rewards and the
validator distribution is created, the rest of the rewards are distributed to
the delegators based on what fraction of a validator's `DelegationTotal`
a delegator's `Delegation` represents.

The rewards distributions are calculated during every election. Delegators and
Validators both claim their rewards identically, by calling the
`ClaimDistribution` function. A validator cannot withhold rewards from delegators
because distribution happens in-protocol.

## The role of `plugin/validators_manager.go`

For any dPoS contract functionality which must be triggered automatically by
Tendermint events, a `ValidatorManager` is used to call dPoS functions based on
the content of Tendermint `EndBlockRequest`s and `BeginBlockRequest`s.
