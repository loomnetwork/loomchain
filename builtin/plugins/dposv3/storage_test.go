package dposv3

import (
	"sort"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/go-loom/types"
	"github.com/stretchr/testify/assert"
)

var chainID = "default"
var (
	pub1  = []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k=")
	addr1 = loom.MustParseAddress("default:0x27690aE5F91C620B13266dA9044b8c0F35CeFbdC")
	pub2  = []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dddSlxe6Hd30ZuuYWgps")
	addr2 = loom.MustParseAddress("default:0x7278ec96B05B44c643c07005577e9F060dAef4EF")
	pub3  = []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU=")
	addr3 = loom.MustParseAddress("default:0x8364d3A808D586b908b6B18Fc726ff134408fbfE")
	pub4  = []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A=")
	addr4 = loom.MustParseAddress("default:0x9c285B0CE29E29C557a06Ca3a27cf1F550a96f38")
)

func TestAddAndSortCandidateList(t *testing.T) {
	var cl CandidateList
	cl.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: addr2.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: addr3.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: addr1.Local},
	})

	assert.Equal(t, 3, len(cl))

	// add duplicated entry
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: addr3.Local},
	})
	assert.Equal(t, 3, len(cl))

	sort.Sort(byAddress(cl))
	if !sort.IsSorted(byAddress(cl)) {
		t.Fatal("candidate list is not sorted")
	}

	// add another entry
	cl.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: addr4.Local},
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

func TestGetSetCandidateList(t *testing.T) {
	var cands CandidateList
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: addr1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: addr2.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: addr3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// duplicate
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: addr3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// get
	cand1 := cands.Get(loom.Address{ChainID: chainID, Local: addr1.Local})
	assert.NotNil(t, cand1)
	assert.Equal(t, cand1.PubKey, pub1)
	assert.Equal(t, 0, cand1.Address.Local.Compare(addr1.Local))

	cand4 := cands.Get(loom.Address{ChainID: chainID, Local: addr4.Local})
	assert.Nil(t, cand4)

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: addr4.Local},
	})
	assert.Equal(t, 4, len(cands))
}

func TestSortCandidateList(t *testing.T) {
	var cands CandidateList

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{ChainId: chainID, Local: addr4.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{ChainId: chainID, Local: addr1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{ChainId: chainID, Local: addr3.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: addr2.Local},
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
		Address: &types.Address{ChainId: chainID, Local: addr1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{ChainId: chainID, Local: addr2.Local},
	})

	assert.Equal(t, 2, len(cands))

	cand1 := cands.Get(loom.Address{ChainID: chainID, Local: addr1.Local})
	assert.NotNil(t, cand1)

	cand2 := cands.Get(loom.Address{ChainID: chainID, Local: addr2.Local})
	assert.NotNil(t, cand2)

	cands.Delete(addr1)

	assert.Equal(t, 1, len(cands))

	cand1 = cands.Get(loom.Address{ChainID: chainID, Local: addr1.Local})
	assert.Nil(t, cand1)

	cand2 = cands.Get(loom.Address{ChainID: chainID, Local: addr2.Local})
	assert.NotNil(t, cand2)
}

func TestGetSetDistributions(t *testing.T) {
	address1 := delegatorAddress1
	address2 := delegatorAddress2

	pctx := plugin.CreateFakeContext(address1, address1)
	ctx := contractpb.WrapPluginContext(pctx)

	distribution := Distribution{
		Address: address1.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	}

	err := SetDistribution(ctx, &distribution)
	assert.Nil(t, err)

	d, err := GetDistribution(ctx, *address1.MarshalPB())
	assert.NotNil(t, d)
	assert.Nil(t, err)

	assert.Equal(t, 0, d.Address.Local.Compare(address1.Local))
	assert.Equal(t, 0, d.Amount.Value.Cmp(loom.NewBigUIntFromInt(1)))

	distribution2 := Distribution{
		Address: address2.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	}

	// Creating new distribution for address2
	err = SetDistribution(ctx, &distribution2)
	assert.Nil(t, err)

	// Updating address1's distribution
	distribution.Amount = &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}
	err = SetDistribution(ctx, &distribution)
	assert.Nil(t, err)

	d, err = GetDistribution(ctx, *address2.MarshalPB())
	assert.NotNil(t, d)
	assert.Nil(t, err)

	assert.Equal(t, 0, d.Address.Local.Compare(address2.Local))
	assert.Equal(t, 0, d.Amount.Value.Cmp(loom.NewBigUIntFromInt(10)))

	// Checking that address1's distribution was properly updated
	d, err = GetDistribution(ctx, *address1.MarshalPB())
	assert.NotNil(t, d)
	assert.Nil(t, err)

	assert.Equal(t, 0, d.Address.Local.Compare(address1.Local))
	assert.Equal(t, 0, d.Amount.Value.Cmp(loom.NewBigUIntFromInt(5)))
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
	address1 := &types.Address{ChainId: chainID, Local: addr1.Local}
	address2 := &types.Address{ChainId: chainID, Local: addr2.Local}
	address3 := &types.Address{ChainId: chainID, Local: addr3.Local}
	address4 := &types.Address{ChainId: chainID, Local: addr2.Local}
	pctx := plugin.CreateFakeContext(loom.UnmarshalAddressPB(address1), loom.UnmarshalAddressPB(address1))
	ctx := contractpb.WrapPluginContext(pctx)

	dl.SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address2,
		Height:    10,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	})
	dl.SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address3,
		Height:    10,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
	})
	dl.SetDelegation(ctx, &Delegation{
		Validator: address1,
		Delegator: address4,
		Height:    10,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	})
	assert.Equal(t, 3, len(dl))

	// Test getting first set entry
	delegation0, err := GetDelegation(ctx, *address2, *address2)
	assert.Nil(t, err)
	assert.NotNil(t, delegation0)
	assert.Equal(t, delegation0.Validator.Local.Compare(address2.Local), 0)
	assert.Equal(t, delegation0.Delegator.Local.Compare(address2.Local), 0)
	// should contain updated value, not original value
	assert.Equal(t, delegation0.Amount, &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)})
	assert.Equal(t, delegation0.Height, uint64(10))

	// add updated entry
	dl.SetDelegation(ctx, &Delegation{
		Validator: address2,
		Delegator: address2,
		Height:    10,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)},
	})
	assert.Equal(t, 3, len(dl))

	// Test getting first set entry
	delegation1, err := GetDelegation(ctx, *address2, *address2)
	assert.Nil(t, err)
	assert.NotNil(t, delegation1)
	assert.Equal(t, delegation1.Validator.Local.Compare(address2.Local), 0)
	assert.Equal(t, delegation1.Delegator.Local.Compare(address2.Local), 0)
	// should contain updated value, not original value
	assert.Equal(t, delegation1.Amount, &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)})
	assert.Equal(t, delegation1.Height, uint64(10))

	sort.Sort(byValidatorAndDelegator(dl))
	if !sort.IsSorted(byValidatorAndDelegator(dl)) {
		t.Fatal("delegation list is not sorted")
	}

	// add another entry
	dl.SetDelegation(ctx, &Delegation{
		Validator: address3,
		Delegator: address3,
		Height:    10,
		Amount:    &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	})

	assert.Equal(t, 4, len(dl))

	sort.Sort(byValidatorAndDelegator(dl))
	assert.True(t, sort.IsSorted(byValidatorAndDelegator(dl)))
}
