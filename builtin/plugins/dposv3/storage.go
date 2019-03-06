package dposv3

import (
	"bytes"
	"math/big"
	"sort"
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
)

var (
	stateKey         = []byte("state")
	candidatesKey    = []byte("candidates")
	delegationsKey   = []byte("delegation")
	distributionsKey = []byte("distribution")
	statisticsKey    = []byte("statistic")

	requestBatchTallyKey = []byte("request_batch_tally")
)

func sortValidators(validators []*Validator) []*Validator {
	sort.Sort(byPubkey(validators))
	return validators
}

func sortCandidates(cands CandidateList) CandidateList {
	sort.Sort(byAddress(cands))
	return cands
}

func sortDelegations(delegations DelegationList) DelegationList {
	sort.Sort(byValidatorAndDelegator(delegations))
	return delegations
}

type byPubkey []*Validator

func (s byPubkey) Len() int {
	return len(s)
}

func (s byPubkey) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPubkey) Less(i, j int) bool {
	return bytes.Compare(s[i].PubKey, s[j].PubKey) < 0
}

type DelegationList []*DelegationIndex

func computeDelegationsKey(index uint64, validator, delegator types.Address) ([]byte, error) {
	indexBytes := []byte(fmt.Sprintf("%d", index))
	validatorAddressBytes, err := validator.Local.Marshal()
	if err != nil {
		return nil, err
	}
	delegatorAddressBytes, err := delegator.Local.Marshal()
	if err != nil {
		return nil, err
	}

	delegationKey := append(append(indexBytes, validatorAddressBytes...), delegatorAddressBytes...)
	return delegationKey, nil
}

func GetDelegation(ctx contract.StaticContext, index uint64, validator types.Address, delegator types.Address) (*Delegation, error) {
	delegationKey, err := computeDelegationsKey(index, validator, delegator)
	if err != nil {
		return nil, err
	}

	var delegation Delegation
	err = ctx.Get(append(delegationsKey, delegationKey...), &delegation)
	if err != nil {
		return nil, err
	}

	return &delegation, nil
}

// Iterates over non-rewards delegaton indices to find the next available slot
// for a new delegation entry
func GetNextDelegationIndex(ctx contract.StaticContext, validator types.Address, delegator types.Address) (uint64, error) {
	var index uint64 = 1
	for {
		delegation, err := GetDelegation(ctx, index, validator, delegator)
		if err != nil && err != contract.ErrNotFound {
			return 0, err
		}

		if delegation == nil {
			break
		}
		index++
	}
	return index, nil
}

func DelegationsCount(ctx contract.StaticContext) int {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return 0
	}

	return len(delegations)
}

func SetDelegation(ctx contract.Context, delegation *Delegation) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	delegationIndex := &DelegationIndex{
		Validator: delegation.Validator,
		Delegator: delegation.Delegator,
		Index: delegation.Index,
	}

	pastvalue, _ := GetDelegation(ctx, delegation.Index, *delegation.Validator, *delegation.Delegator)
	if pastvalue == nil {
		delegations = append(delegations, delegationIndex)
		if err := saveDelegationList(ctx, delegations); err != nil {
			return err
		}
	}

	delegationKey, err := computeDelegationsKey(delegationIndex.Index, *delegation.Validator, *delegation.Delegator)
	if err != nil {
		return err
	}

	return ctx.Set(append(delegationsKey, delegationKey...), delegation)
}

func DeleteDelegation(ctx contract.Context, delegation *Delegation) error {
	delegations, err := loadDelegationList(ctx)
	if err != nil {
		return err
	}

	for i, d := range delegations {
		if delegation.Validator.Local.Compare(d.Validator.Local) == 0 && delegation.Delegator.Local.Compare(d.Delegator.Local) == 0 {
			copy(delegations[i:], delegations[i+1:])
			delegations = delegations[:len(delegations)-1]
			break
		}
	}
	if err := saveDelegationList(ctx, delegations); err != nil {
		return err
	}

	delegationKey, err := computeDelegationsKey(delegation.Index, *delegation.Validator, *delegation.Delegator)
	if err != nil {
		return err
	}

	ctx.Delete(append(delegationsKey, delegationKey...))

	return nil
}

func saveDelegationList(ctx contract.Context, dl DelegationList) error {
	sorted := sortDelegations(dl)
	return ctx.Set(delegationsKey, &dtypes.DelegationList{Delegations: sorted})
}

func loadDelegationList(ctx contract.StaticContext) (DelegationList, error) {
	var pbcl dtypes.DelegationList
	err := ctx.Get(delegationsKey, &pbcl)
	if err == contract.ErrNotFound {
		return DelegationList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Delegations, nil
}

type byValidatorAndDelegator DelegationList

func (s byValidatorAndDelegator) Len() int {
	return len(s)
}

func (s byValidatorAndDelegator) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byValidatorAndDelegator) Less(i, j int) bool {
	vAddr1 := loom.UnmarshalAddressPB(s[i].Validator)
	vAddr2 := loom.UnmarshalAddressPB(s[j].Validator)
	diff := vAddr1.Compare(vAddr2)

	if diff == 0 {
		dAddr1 := loom.UnmarshalAddressPB(s[i].Delegator)
		dAddr2 := loom.UnmarshalAddressPB(s[j].Delegator)
		diff = dAddr1.Compare(dAddr2)
	}

	if diff == 0 {
		return s[i].Index < s[j].Index
	}

	return diff < 0
}

func GetStatistic(ctx contract.StaticContext, address loom.Address) (*ValidatorStatistic, error) {
	addressBytes, err := address.Local.Marshal()
	if err != nil {
		return nil, err
	}
	return GetStatisticByAddressBytes(ctx, addressBytes)
}

func GetStatisticByAddressBytes(ctx contract.StaticContext, addressBytes []byte) (*ValidatorStatistic, error) {
	var statistic ValidatorStatistic
	err := ctx.Get(append(statisticsKey, addressBytes...), &statistic)
	if err != nil {
		return nil, err
	}

	return &statistic, nil
}

func SetStatistic(ctx contract.Context, statistic *ValidatorStatistic) error {
	addressBytes, err := statistic.Address.Local.Marshal()
	if err != nil {
		return err
	}

	return ctx.Set(append(statisticsKey, addressBytes...), statistic)
}

func GetDistribution(ctx contract.StaticContext, delegator types.Address) (*Distribution, error) {
	addressBytes, err := delegator.Local.Marshal()
	if err != nil {
		return nil, err
	}

	var distribution Distribution
	err = ctx.Get(append(distributionsKey, addressBytes...), &distribution)
	if err != nil {
		return nil, err
	}

	return &distribution, nil
}

func SetDistribution(ctx contract.Context, distribution *Distribution) error {
	addressBytes, err := distribution.Address.Local.Marshal()
	if err != nil {
		return err
	}

	return ctx.Set(append(distributionsKey, addressBytes...), distribution)
}

func IncreaseDistribution(ctx contract.Context, delegator types.Address, increase loom.BigUInt) error {
	distribution, err := GetDistribution(ctx, delegator)
	if err == nil {
		updatedAmount := loom.BigUInt{big.NewInt(0)}
		updatedAmount.Add(&distribution.Amount.Value, &increase)
		distribution.Amount = &types.BigUInt{Value: updatedAmount}
		return SetDistribution(ctx, distribution)
	} else if err == contract.ErrNotFound {
		return SetDistribution(ctx, &Distribution{Address: &delegator, Amount: &types.BigUInt{Value: increase}})
	} else {
		return err
	}
}

func ResetDistributionTotal(ctx contract.Context, delegator types.Address) error {
	distribution, err := GetDistribution(ctx, delegator)
	if err != nil {
		return err
	}

	if distribution == nil {
		return errDistributionNotFound
	} else {
		distribution.Amount = &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}}
	}
	return SetDistribution(ctx, distribution)
}

type CandidateList []*Candidate

func (c CandidateList) Get(addr loom.Address) *Candidate {
	for _, cand := range c {
		if cand.Address.Local.Compare(addr.Local) == 0 {
			return cand
		}
	}
	return nil
}

func GetCandidate(ctx contract.StaticContext, addr loom.Address) *Candidate {
	c, err := loadCandidateList(ctx)
	if err != nil {
		return nil
	}

	for _, cand := range c {
		if cand.Address.Local.Compare(addr.Local) == 0 {
			return cand
		}
	}
	return nil
}

func GetCandidateByPubKey(ctx contract.StaticContext, pubkey []byte) *Candidate {
	c, err := loadCandidateList(ctx)
	if err != nil {
		return nil
	}

	for _, cand := range c {
		if bytes.Compare(cand.PubKey, pubkey) == 0 {
			return cand
		}
	}
	return nil
}

func (c *CandidateList) Set(cand *Candidate) {
	found := false
	candAddr := loom.UnmarshalAddressPB(cand.Address)
	for _, candidate := range *c {
		addr := loom.UnmarshalAddressPB(candidate.Address)
		if candAddr.Local.Compare(addr.Local) == 0 {
			candidate = cand
			found = true
			break
		}
	}
	if !found {
		*c = append(*c, cand)
	}
}

func (c *CandidateList) Delete(addr loom.Address) {
	newcl := *c
	for i, cand := range newcl {
		candAddr := loom.UnmarshalAddressPB(cand.Address)
		if candAddr.Local.Compare(addr.Local) == 0 {
			copy(newcl[i:], newcl[i+1:])
			newcl = newcl[:len(newcl)-1]
			break
		}
	}
	*c = newcl
}

func updateCandidateFeeDelays(ctx contract.Context) error {
	candidates, err := loadCandidateList(ctx)
	if err != nil {
		return err
	}

	// Update each candidate's fee
	for _, c := range candidates {
		if c.Fee != c.NewFee {
			c.FeeDelayCounter += 1
			if c.FeeDelayCounter == FEE_CHANGE_DELAY {
				c.Fee = c.NewFee
			}
		}
	}

	if err = saveCandidateList(ctx, candidates); err != nil {
		return err
	}

	return nil
}

type byAddress CandidateList

func (s byAddress) Len() int {
	return len(s)
}

func (s byAddress) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byAddress) Less(i, j int) bool {
	vaddr1 := loom.UnmarshalAddressPB(s[i].Address)
	vaddr2 := loom.UnmarshalAddressPB(s[j].Address)
	diff := vaddr1.Local.Compare(vaddr2.Local)
	return diff < 0
}

func saveCandidateList(ctx contract.Context, cl CandidateList) error {
	sorted := sortCandidates(cl)
	return ctx.Set(candidatesKey, &dtypes.CandidateList{Candidates: sorted})
}

func loadCandidateList(ctx contract.StaticContext) (CandidateList, error) {
	var pbcl dtypes.CandidateList
	err := ctx.Get(candidatesKey, &pbcl)
	if err == contract.ErrNotFound {
		return CandidateList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Candidates, nil
}

func saveState(ctx contract.Context, state *State) error {
	state.Validators = sortValidators(state.Validators)
	return ctx.Set(stateKey, state)
}

func loadState(ctx contract.StaticContext) (*State, error) {
	var state State
	err := ctx.Get(stateKey, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

type DelegationResult struct {
	ValidatorAddress loom.Address
	DelegationTotal  loom.BigUInt
}

type byDelegationTotal []*DelegationResult

func (s byDelegationTotal) Len() int {
	return len(s)
}

func (s byDelegationTotal) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDelegationTotal) Less(i, j int) bool {
	diff := int64(s[i].DelegationTotal.Cmp(&s[j].DelegationTotal))
	if diff == 0 {
		// make sure output is deterministic if power is equal
		diff = int64(s[i].ValidatorAddress.Compare(s[j].ValidatorAddress))
	}

	return diff > 0
}

// Returns the elements of `former` which are not included in `current`
// `former` and `current` are always assumed to be sorted since validator lists
// are only stored as sorted arrays in `Params.State`
func MissingValidators(former, current []*Validator) []*Validator {
	var validators []*Validator

	var i, j int
	for j < len(former) {
		if i >= len(current) {
			validators = append(validators, former[j:]...)
			break
		}

		switch bytes.Compare(former[j].PubKey, current[i].PubKey) {
		case -1:
			validators = append(validators, former[j])
			j++
		case 0:
			i++
			j++
		case 1:
			i++
		}
	}

	return validators
}

// BATCH REQUESTS

func loadRequestBatchTally(ctx contract.StaticContext) (*RequestBatchTally, error) {
	tally := RequestBatchTally{}

	if err := ctx.Get(requestBatchTallyKey, &tally); err != nil {
		if err == contract.ErrNotFound {
			return &tally, nil
		}
		return nil, err
	}

	return &tally, nil
}

func saveRequestBatchTally(ctx contract.Context, tally *RequestBatchTally) error {
	return ctx.Set(requestBatchTallyKey, tally)
}
