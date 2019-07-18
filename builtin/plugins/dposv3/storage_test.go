package dposv3

import (
	"bytes"
	"sort"
	"testing"

	"github.com/loomnetwork/loomchain"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/go-loom/types"
	"github.com/stretchr/testify/assert"
)

var (
	pub1     = []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k=")
	address1 = loom.MustParseAddress("default:0x27690aE5F91C620B13266dA9044b8c0F35CeFbdC")
	pub2     = []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dddSlxe6Hd30ZuuYWgps")
	address2 = loom.MustParseAddress("default:0x7278ec96B05B44c643c07005577e9F060dAef4EF")
	pub3     = []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU=")
	address3 = loom.MustParseAddress("default:0x8364d3A808D586b908b6B18Fc726ff134408fbfE")
	pub4     = []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A=")
	address4 = loom.MustParseAddress("default:0x9c285B0CE29E29C557a06Ca3a27cf1F550a96f38")
)

func TestAddAndSortCandidateList(t *testing.T) {
	var cl CandidateList
	cl.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: address2.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: address3.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: address1.Local},
	})

	assert.Equal(t, 3, len(cl))

	// add duplicated entry
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: address3.Local},
	})
	assert.Equal(t, 3, len(cl))

	sort.Sort(byAddress(cl))
	if !sort.IsSorted(byAddress(cl)) {
		t.Fatal("candidate list is not sorted")
	}

	// add another entry
	cl.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: address4.Local},
	})
	assert.Equal(t, 4, len(cl))

	sort.Sort(byAddress(cl))
	assert.True(t, sort.IsSorted(byAddress(cl)))
}

func TestSortValidatorList(t *testing.T) {
	validators := []*Validator{
		&Validator{
			PubKey: []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A="),
		},
		&Validator{
			PubKey: []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU="),
		},
		&Validator{
			PubKey: []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k="),
		},
		&Validator{
			PubKey: []byte("bOZnGz5QzPh7xFHKlqyFQqMeEsidI8XmWClLlWuS5dw=+k="),
		},
		&Validator{
			PubKey: []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dSlxe6Hd30ZuuYWgps"),
		},
	}

	sortedValidatores := sortValidators(validators)
	assert.True(t, sort.IsSorted(byPubkey(sortedValidatores)))

	sortedValidatores = append(sortedValidatores, &Validator{
		PubKey: []byte("2AUfclH6vC7G2jkf7RxOTzhTYHVdE/2Qp5WSsK8m/tQ="),
	})

	sortedValidatores = sortValidators(validators)
	assert.True(t, sort.IsSorted(byPubkey(sortedValidatores)))
}

func TestMissingValidators(t *testing.T) {
	validatorsA := []*Validator{
		&Validator{
			PubKey: []byte("aaaaaa"),
		},
		&Validator{
			PubKey: []byte("bbbbbb"),
		},
		&Validator{
			PubKey: []byte("cccccc"),
		},
		&Validator{
			PubKey: []byte("uuuuuu"),
		},
		&Validator{
			PubKey: []byte("rrrrrr"),
		},
	}
	validatorsA = sortValidators(validatorsA)

	validatorsB := append(append(make([]*Validator, 0, len(validatorsA)+1), validatorsA...),
		&Validator{
			PubKey: []byte("ddddd"),
		})
	validatorsB = sortValidators(validatorsB)

	// B - A should return ["ddddd"]
	bMinusA := MissingValidators(validatorsB, validatorsA)
	assert.Equal(t, 1, len(bMinusA))
	assert.Equal(t, 0, bytes.Compare(bMinusA[0].PubKey, []byte("ddddd")))

	// A - B should return []
	aMinusB := MissingValidators(validatorsA, validatorsB)
	assert.Equal(t, 0, len(aMinusB))

	// A - [] should return A
	var empty = make([]*Validator, 0)
	assert.Equal(t, len(validatorsA), len(MissingValidators(validatorsA, empty)))

	// [] - A should return []
	assert.Equal(t, len(empty), len(MissingValidators(empty, validatorsA)))

	validatorsC := append(append(make([]*Validator, 0, len(validatorsB)+1), validatorsB...),
		&Validator{
			PubKey: []byte("zzzzz"),
		})
	validatorsC = sortValidators(validatorsC)

	// C - A should return ["ddddd"], ["zzzzz"]
	cMinusA := MissingValidators(validatorsC, validatorsA)
	assert.Equal(t, 2, len(cMinusA))
	assert.Equal(t, 0, bytes.Compare(cMinusA[0].PubKey, []byte("ddddd")))
	assert.Equal(t, 0, bytes.Compare(cMinusA[1].PubKey, []byte("zzzzz")))

	// A - C should return []
	aMinusC := MissingValidators(validatorsA, validatorsC)
	assert.Equal(t, 0, len(aMinusC))
}

func TestGetSetCandidateList(t *testing.T) {
	var cands CandidateList
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: address1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: address2.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: address3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// duplicate
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: address3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// get
	cand1 := cands.Get(loom.Address{ChainID: chainID, Local: address1.Local})
	assert.NotNil(t, cand1)
	assert.Equal(t, cand1.PubKey, pub1)
	assert.Equal(t, 0, cand1.Address.Local.Compare(address1.Local))

	cand4 := cands.Get(loom.Address{ChainID: chainID, Local: address4.Local})
	assert.Nil(t, cand4)

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: address4.Local},
	})
	assert.Equal(t, 4, len(cands))
}

func TestSortCandidateList(t *testing.T) {
	var cands CandidateList

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: address4.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: address1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: address3.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: address2.Local},
	})

	sortedCands := sortCandidates(cands)
	assert.True(t, sort.IsSorted(byAddress(sortedCands)))
}

func TestCalcFraction(t *testing.T) {
	amount := *loom.NewBigUIntFromInt(125000000)
	assert.Equal(t, CalculateFraction(TierBonusMap[TierMap[0]], amount), amount)

	newAmount := *loom.NewBigUIntFromInt(187500000)
	assert.Equal(t, CalculateFraction(TierBonusMap[TierMap[1]], amount), newAmount)

	newAmount = *loom.NewBigUIntFromInt(250000000)
	assert.Equal(t, CalculateFraction(TierBonusMap[TierMap[2]], amount), newAmount)

	newAmount = *loom.NewBigUIntFromInt(500000000)
	assert.Equal(t, CalculateFraction(TierBonusMap[TierMap[3]], amount), newAmount)

}

func TestCandidateDelete(t *testing.T) {
	var cands CandidateList

	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: address1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: address2.Local},
	})

	assert.Equal(t, 2, len(cands))

	cand1 := cands.Get(loom.Address{ChainID: chainID, Local: address1.Local})
	assert.NotNil(t, cand1)

	cand2 := cands.Get(loom.Address{ChainID: chainID, Local: address2.Local})
	assert.NotNil(t, cand2)

	cands.Delete(address1)

	assert.Equal(t, 1, len(cands))

	cand1 = cands.Get(loom.Address{ChainID: chainID, Local: address1.Local})
	assert.Nil(t, cand1)

	cand2 = cands.Get(loom.Address{ChainID: chainID, Local: address2.Local})
	assert.NotNil(t, cand2)
}

func TestGetSetStatistics(t *testing.T) {
	address1 := delegatorAddress1
	address2 := delegatorAddress2

	pctx := plugin.CreateFakeContext(address1, address1)
	ctx := contractpb.WrapPluginContext(pctx)

	statistic := ValidatorStatistic{
		Address:         address1.MarshalPB(),
		WhitelistAmount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	}

	err := SetStatistic(ctx, &statistic)
	assert.Nil(t, err)

	s, err := GetStatistic(ctx, address1)
	assert.NotNil(t, s)
	assert.Nil(t, err)

	assert.Equal(t, 0, s.Address.Local.Compare(address1.Local))
	assert.Equal(t, 0, s.WhitelistAmount.Value.Cmp(loom.NewBigUIntFromInt(1)))

	statistic2 := ValidatorStatistic{
		Address:         address2.MarshalPB(),
		WhitelistAmount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	}

	// Creating new distribution for address2
	err = SetStatistic(ctx, &statistic2)
	assert.Nil(t, err)

	// Updating address1's distribution
	statistic.WhitelistAmount = &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}
	err = SetStatistic(ctx, &statistic)
	assert.Nil(t, err)

	s, err = GetStatistic(ctx, address2)
	assert.NotNil(t, s)
	assert.Nil(t, err)

	assert.Equal(t, 0, s.Address.Local.Compare(address2.Local))
	assert.Equal(t, 0, s.WhitelistAmount.Value.Cmp(loom.NewBigUIntFromInt(10)))

	// Checking that address1's distribution was properly updated
	s, err = GetStatistic(ctx, address1)
	assert.NotNil(t, s)
	assert.Nil(t, err)

	assert.Equal(t, 0, s.Address.Local.Compare(address1.Local))
	assert.Equal(t, 0, s.WhitelistAmount.Value.Cmp(loom.NewBigUIntFromInt(5)))
}

func TestAddAndSortDelegationList(t *testing.T) {
	var dl DelegationList
	address1 := &types.Address{ChainId: chainID, Local: address1.Local}
	address2 := &types.Address{ChainId: chainID, Local: address2.Local}
	address3 := &types.Address{ChainId: chainID, Local: address3.Local}
	address4 := &types.Address{ChainId: chainID, Local: address2.Local}
	pctx := plugin.CreateFakeContext(loom.UnmarshalAddressPB(address1), loom.UnmarshalAddressPB(address1))
	ctx := contractpb.WrapPluginContext(pctx)

	SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address2,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	})
	SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address3,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
	})
	SetDelegation(ctx, &Delegation{
		Validator: address1,
		Delegator: address4,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	})

	// Test getting first set entry
	delegation0, err := GetDelegation(ctx, 0, *address2, *address2)
	assert.Nil(t, err)
	assert.NotNil(t, delegation0)
	assert.Equal(t, delegation0.Validator.Local.Compare(address2.Local), 0)
	assert.Equal(t, delegation0.Delegator.Local.Compare(address2.Local), 0)
	// should contain updated value, not original value
	assert.Equal(t, delegation0.Amount, &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)})

	// add updated entry
	SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address2,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)},
	})

	// Test getting first set entry
	delegation1, err := GetDelegation(ctx, 0, *address2, *address2)
	assert.Nil(t, err)
	assert.NotNil(t, delegation1)
	assert.Equal(t, delegation1.Validator.Local.Compare(address2.Local), 0)
	assert.Equal(t, delegation1.Delegator.Local.Compare(address2.Local), 0)
	// should contain updated value, not original value
	assert.Equal(t, delegation1.Amount, &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)})

	sort.Sort(byValidatorAndDelegator(dl))
	if !sort.IsSorted(byValidatorAndDelegator(dl)) {
		t.Fatal("delegation list is not sorted")
	}

	// add another entry with same (validator, delegator) pair as first set
	// delegation
	highIndexDelegation := &Delegation{
		Validator: address2,
		Delegator: address2,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
		Index:     3,
	}
	SetDelegation(ctx, highIndexDelegation)

	sort.Sort(byValidatorAndDelegator(dl))
	assert.True(t, sort.IsSorted(byValidatorAndDelegator(dl)))

	DeleteDelegation(ctx, highIndexDelegation)

	delegationResult, err := GetDelegation(ctx, 0, *address2, *address2)
	assert.NotNil(t, delegationResult)
	delegationResult, err = GetDelegation(ctx, 3, *address2, *address2)
	assert.Nil(t, delegationResult)
}

func TestGetSetReferrer(t *testing.T) {
	pctx := plugin.CreateFakeContext(address1, address1)
	ctx := contractpb.WrapPluginContext(pctx)

	err := SetReferrer(ctx, "hi", address1.MarshalPB())
	assert.Nil(t, err)
	address := GetReferrer(ctx, "hi")
	assert.NotNil(t, address)
	assert.True(t, address.Local.Compare(address1.Local) == 0)

	//Enable feature dpos:v3.5
	pctx.SetFeature(loomchain.DPOSVersion3_5, true)
	err = SetReferrer(ctx, "hi-3.5", address1.MarshalPB())
	assert.Nil(t, err)
	addr := GetReferrer(ctx, "hi-3.5")
	assert.NotNil(t, addr)
	assert.True(t, addr.Local.Compare(address1.Local) == 0)

	address = GetReferrer(ctx, "bye")
	assert.Nil(t, address)
}
