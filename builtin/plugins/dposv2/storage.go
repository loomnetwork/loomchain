package dposv2

import (
	"bytes"
	"math/big"
	"sort"
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/types"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

var (
	stateKey         = []byte("state")
	candidatesKey    = []byte("candidates")
	delegationsKey   = []byte("delegation")
	distributionsKey = []byte("distribution")
	statisticsKey    = []byte("statistic")
)

func addrKey(addr loom.Address) string {
	return string(addr.Bytes())
}

func sortValidators(validators []*DposValidator) []*DposValidator {
	sort.Sort(byPubkey(validators))
	return validators
}

func sortCandidates(cands []*Candidate) []*Candidate {
	sort.Sort(byAddress(cands))
	return cands
}

func sortDelegations(delegations []*Delegation) []*Delegation {
	sort.Sort(byValidatorAndDelegator(delegations))
	return delegations
}

func sortDistributions(distributions DistributionList) DistributionList {
	sort.Sort(byAddressAndAmount(distributions))
	return distributions
}

func sortStatistics(statistics ValidatorStatisticList) ValidatorStatisticList {
	sort.Sort(byValidatorAddress(statistics))
	return statistics
}

type byPubkey []*DposValidator

func (s byPubkey) Len() int {
	return len(s)
}

func (s byPubkey) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPubkey) Less(i, j int) bool {
	return bytes.Compare(s[i].PubKey, s[j].PubKey) < 0
}

type DelegationList []*Delegation

func (dl DelegationList) Get(validator types.Address, delegator types.Address) *Delegation {
	for _, delegation := range dl {
		// TODO shouldn't I just convert to loom.Address and use its compare?
		if delegation.Validator.Local.Compare(validator.Local) == 0 && delegation.Delegator.Local.Compare(delegator.Local) == 0 {
			return delegation
		}
	}
	return nil
}

func (dl *DelegationList) Set(delegation *Delegation) {
	pastvalue := dl.Get(*delegation.Validator, *delegation.Delegator)
	if pastvalue == nil {
		*dl = append(*dl, delegation)
	} else {
		pastvalue.Amount = delegation.Amount
		pastvalue.Height = delegation.Height
	}
}

// func (c *DelegationList) Delete(validator loom.Address, delegator loom.Address) {

func saveDelegationList(ctx contract.Context, dl DelegationList) error {
	sorted := sortDelegations(dl)
	return ctx.Set(delegationsKey, &dtypes.DelegationListV2{Delegations: sorted})
}

func loadDelegationList(ctx contract.StaticContext) (DelegationList, error) {
	var pbcl dtypes.DelegationListV2
	err := ctx.Get(delegationsKey, &pbcl)
	if err == contract.ErrNotFound {
		return DelegationList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Delegations, nil
}

type ValidatorStatisticList []*ValidatorStatistic

func (sl ValidatorStatisticList) Get(validatorAddress types.Address) *ValidatorStatistic {
	for _, stat := range sl {
		if stat.Address.Local.Compare(validatorAddress.Local) == 0 {
			return stat
		}
	}
	return nil
}

func (sl *ValidatorStatisticList) IncreaseValidatorReward(validatorAddress types.Address, reward loom.BigUInt) error {
	pastvalue := sl.Get(validatorAddress)
	if pastvalue == nil {
		return errValidatorNotFound
	} else {
		fmt.Println("pastvalue: %s", pastvalue)
		updatedAmount := loom.BigUInt{big.NewInt(0)}
		updatedAmount.Add(&pastvalue.DistributionTotal.Value, &reward)
		pastvalue.DistributionTotal = &types.BigUInt{updatedAmount}
	}
	return nil
}

func saveValidatorStatisticList(ctx contract.Context, sl ValidatorStatisticList) error {
	sorted := sortStatistics(sl)
	return ctx.Set(statisticsKey, &dtypes.ValidatorStatisticListV2{Statistics: sorted})
}

func loadValidatorStatisticList(ctx contract.StaticContext) (ValidatorStatisticList, error) {
	var pbcl dtypes.ValidatorStatisticListV2
	err := ctx.Get(statisticsKey, &pbcl)
	if err == contract.ErrNotFound {
		return ValidatorStatisticList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Statistics, nil
}

type byValidatorAddress ValidatorStatisticList

func (s byValidatorAddress) Len() int {
	return len(s)
}

func (s byValidatorAddress) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byValidatorAddress) Less(i, j int) bool {
	vaddr1 := loom.UnmarshalAddressPB(s[i].Address)
	vaddr2 := loom.UnmarshalAddressPB(s[j].Address)
	diff := vaddr1.Local.Compare(vaddr2.Local)
	return diff < 0
}

type DistributionList []*Distribution

func (dl DistributionList) Get(delegator types.Address) *Distribution {
	for _, distribution := range dl {
		// TODO shouldn't I just convert to loom.Address and use its compare?
		if distribution.Address.Local.Compare(delegator.Local) == 0 {
			return distribution
		}
	}
	return nil
}

func (dl *DistributionList) IncreaseDistribution(delegator types.Address, increase loom.BigUInt) error {
	distribution := dl.Get(delegator)
	if distribution == nil {
		*dl = append(*dl, &Distribution{Address: &delegator, Amount: &types.BigUInt{increase}})
	} else {
		updatedAmount := loom.BigUInt{big.NewInt(0)}
		updatedAmount.Add(&distribution.Amount.Value, &increase)
		distribution.Amount = &types.BigUInt{updatedAmount}
	}
	return nil
}

// func (dl *DistributionList) DeleteDistribution(delegator types.Address, increase loom.BigUInt) error {

func saveDistributionList(ctx contract.Context, dl DistributionList) error {
	sorted := sortDistributions(dl)
	return ctx.Set(distributionsKey, &dtypes.DistributionListV2{Distributions: sorted})
}

func loadDistributionList(ctx contract.StaticContext) (DistributionList, error) {
	var pbcl dtypes.DistributionListV2
	err := ctx.Get(distributionsKey, &pbcl)
	if err == contract.ErrNotFound {
		return DistributionList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return pbcl.Distributions, nil
}

type byValidatorAndDelegator []*Delegation

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

	return diff < 0
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

func (c CandidateList) GetByPubKey(pubkey []byte) *Candidate {
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
	var newcl CandidateList
	for _, cand := range *c {
		candAddr := loom.UnmarshalAddressPB(cand.Address)
		addr := loom.UnmarshalAddressPB(cand.Address)
		if candAddr.Local.Compare(addr.Local) != 0 {
			newcl = append(newcl, cand)
		}
	}
	*c = newcl
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
	return ctx.Set(candidatesKey, &dtypes.CandidateListV2{Candidates: sorted})
}

func loadCandidateList(ctx contract.StaticContext) (CandidateList, error) {
	var pbcl dtypes.CandidateListV2
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

type byAddressAndAmount DistributionList

func (s byAddressAndAmount) Len() int {
	return len(s)
}

func (s byAddressAndAmount) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byAddressAndAmount) Less(i, j int) bool {
	diff := bytes.Compare(s[i].Address.Local, s[j].Address.Local)
	if diff == 0 {
		// make sure output is deterministic if for some reason multiple records
		// for the same address exist in the list
		diff = s[i].Amount.Value.Cmp(&s[j].Amount.Value)
	}

	return diff > 0
}

// frac is expressed in basis points
func calculateDistributionShare(frac loom.BigUInt, total loom.BigUInt) loom.BigUInt {
	updatedAmount := loom.BigUInt{big.NewInt(0)}
	updatedAmount.Mul(&total, &frac)
	updatedAmount.Div(&updatedAmount, &loom.BigUInt{big.NewInt(10000)})
	return updatedAmount
}

func calculateShare(delegation loom.BigUInt, total loom.BigUInt, rewards loom.BigUInt) loom.BigUInt {
	frac := loom.BigUInt{big.NewInt(0)}
	// TODO make this a constant
	frac.Mul(&delegation, &loom.BigUInt{big.NewInt(10000)})
	frac.Div(&frac, &total)
	return calculateDistributionShare(frac, rewards)
}
