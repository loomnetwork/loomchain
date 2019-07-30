# delegated Proof of Stake Contract (v3)

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
`Candidate` will take of the rewards he receives for participating in consensus.

`Name`: the `Candidate`'s human-readable name, a secondary identifier

`Description`: Short piece of information about the `Candidate`

`Website`: URL where additional information about the `Candidate` can be found

`Maximum Referral Fee`: The maximum referral fee the candidate is willing to
accept. The referral fee is a percentage of the validator's fee. Any
delegations made via a referrer with too high a fee will be rejected.

#### Registration Parameters

`registrationRequirement`: Quantity in nominal tokens which a would-be validator
must deposit (self-delegate) to the dPoS contract in order to become a canidate
which participates in Elections.

### Delegation

Delegation from a delegator to a validator happens in-protocol so delegators do
not have to trust that a validator will return their tokens--the delegator can
unbond a delegation at any time without asking for the validator's consent.

A delegation is a 5-tuple of `(Delegator, Validator, Amount, UpdateAmount, State)`.

Delegations can exist in three distinct states:

`BONDED`: A token delegation has been made from `Delegator` to `Validator`; the
tokens have been transferred to the dPoS contract and the token amount counts
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
when they are automatically transferred to an address which the delegator specifies.

`REDELEGATING`: A redelegation request has been made within the last election
period. During the next election, the `delegation.Validator` value will be set
to the `delegation.UpdateValidator`.

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

In order to disincentivize dishonest behavior, a validator's `DelegationTotal`
is liable for penalties or "slashes" if the validator commits a fault. There are
two general categories of fault:

`Crash Fault`: Failing to send or receive messages from other nodes, usually due
to being offline

`Byzantine Fault`: Arbitrary deviation from the consensus protocol including
signing different blocks at the same block height

### Slashing Parameters

`doubleSignSlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits
a double-sign (equivocation) fault.

`inactivitySlashPercentage`: Percentage expressed in basis points which is
deducted from a validator's `DelegationTotal` in case the validator commits an
inactivity (crash) fualt.

In any given election period, a validator will not be slashed more than
`inactivitySlashPercentage + doubleSignSlashPercentage`, and this will only
occur if, first, the validator commits an inactivity fault and later commits
a double sign fault whose penalty is greater that that of an inactivity fault.

`maximumDowntimePercentage`: Of the last four periods, a validator must sign at
least `maximumDowntimePercentage` of blocks in order to avoid slashing. For
example, if `maximumDowntimePercentage` is set at 20%, a validator avoids
slashing if during the last four periods he missed [100%, 19%, 100%, 100%] of
blocks, but not if he missed [21%, 21%, 21%, 21%] of blocks.

### Slashing Implementation

Slashing calculations are carried out in `plugin/validators_manager.go` and not
in the dpos contract itself.

Inactivity is calculated by counting how many blocks in a `downtimePeriod`
(measured in blocks) a validator fails to sign. If a validator fails to sign
more than 20% of blocks during any four consecutive `downtimePeriod`s, the
validator is considered inactive.

Inactivity leads to a loss of `inactivitySlashPercentage * stake` not only for
validator but for delegators bonded to him as well.

## Rewards

Besides disincentivizing deviations from the consensus protocol using slashing,
validator participation is incentivized with rewards. As long as a validator
does not commit any faults, i.e. participates in consensus properly, the
validator is rewarded.

### Delegation lockup & rewards bonuses

According to how long a delegator locks his delegation the delegator receives
a different level of reward. Their are four total lockup tiers with an
associated bonus:

- TIER_ZERO      two week lockup       1x bonus
- TIER_ONE       three month lockup    1.5x bonus
- TIER_TWO       six months lockup     2x bonus
- TIER_THREE     one year lockup       4x bonus

Even after a lockup period expires, a delegator continues to enjoy whatever
lockup tier they chose when they delegated.

### Rewards Parameters

`blockRewardPercentage`: Percentage expressed in basis points which a honest
validator should expect his `DelegationTotal` to grow by over the course of
a year.

`maxYearlyRewards`: No election can result in the distribution of more than
(max_yearly_rewards * (election_cycle / year)). This value is set manually by
the oracle.

#### Reward Cap adjustments

TODO

-------

All rewards begin by being awarded to a Validator who participated in consensus.
This initial bulk reward is calculated in dpos.go's func rewardValidator. We now
assume we have not hit a rewards cap but are generating a fraction of the
Validator's DelegationTotal as rewards.

`reward := CalculateFraction(blockRewardPercentage, statistic.DelegationTotal.Value)`,
where currently, blockRewardPercentatge is 5%. Then we scale the reward to the
appropriate reward period:

```
reward.Mul(&reward, &loom.BigUInt{big.NewInt(cycleSeconds)})
reward.Div(&reward, &secondsInYear)
```

Once this scaling is applied, a delegator is given rewards according to their
share of the Validator's DelegationTotal minus the Validator's fee. If we assume
a 0% fee, we can simply calculate the total rewards a Delegator earns by the
following abbreviated formula:

```
(0.05) * delegationAmount * electionPeriodLength / secondsInYear
```

If we assume an election period of 10s, the formula becomes

```
electionCycleReward = 1.5854895991882296e-8 * delegationAmount
```

So even at this short election period setting, a delegation of 1 token (10^18
fractional units) should generate 1.5854895991882296e10 fractional tokens in
rewards. These calculations are carried out with integer arithmetic only, but
the discrepancy in the calculations should be minimal.

dpos.go's func `distributeDelegatorRewards` is where a Validator's earned
rewards are distributed to his delegators.

### Validator Rewards Distribution

A validator takes a fixed percentage of the rewards earned from a delegation.
If, using the calculation above, the total reward for a delegation is 1000
tokens and a validator chares a 10% fee, he recieves 100 tokens as a reward.

#### Referrer Rewards Distribution

NOTE: **Brand new feature**, certain behaviours aren't well-defined.

Referrals fees are currently all 3% by default. These referral fee percentage
are taken from the validotar fee that is taken from the rewards earned on from
a particular delegation. If a validator earns 100 tokens in an election period
from a particular delegation & the referrer of that delegation charges a 3%
fee, the referrer receives 3 tokens and the validator 97.

Note that any referral made while a MaxReferralPercentage > ReferrerFee is
grandfathered in (i.e. valid) even after a candidate lowers their
MaxReferralPercentage

### Delegator Rewards Distribution

After a Validator's fee has been removed from the total rewards and the
validator distribution is created, the rest of the rewards are distributed to
the delegators based on what fraction of a validator's `DelegationTotal`
a delegator's `Delegation` represents.

The rewards distributions are calculated during every eleciton. Delegators and
Validators both claim their rewards identically, by calling the
`ClaimDistribution` function. A validator cannot withhold rewards from delegators
because distribution happens in-protocol.

## The role of `plugin/validators_manager.go`

For any dPoS contract functionality which must be triggered automatically by
Tendermint events, a `ValidatorManager` is used to call dPoS functions based on
the content of Tendermint `EndBlockRequest`s and `BeginBlockRequest`s.
