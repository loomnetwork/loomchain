package dposv2

import (
	"sort"
	"testing"

	"github.com/loomnetwork/go-loom"

	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv2"
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

func TestAddAndSortDelegationList(t *testing.T) {
	var dl DelegationList
	address1 := &types.Address{ChainId: chainID, Local: addr1.Local}
	address2 := &types.Address{ChainId: chainID, Local: addr2.Local}
	address3 := &types.Address{ChainId: chainID, Local: addr3.Local}
	address4 := &types.Address{ChainId: chainID, Local: addr2.Local}

	dl.Set(&dtypes.DelegationV2{
		Validator: address2,
		Delegator: address2,
		Height: 10,
		Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	})
	dl.Set(&dtypes.DelegationV2{
		Validator: address2,
		Delegator: address3,
		Height: 10,
		Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
	})
	dl.Set(&dtypes.DelegationV2{
		Validator: address1,
		Delegator: address4,
		Height: 10,
		Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	})
	assert.Equal(t, 3, len(dl))

	// add updated entry
	dl.Set(&dtypes.DelegationV2{
		Validator: address2,
		Delegator: address2,
		Height: 10,
		Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)},
	})
	assert.Equal(t, 3, len(dl))

	// Test getting first set entry
	delegation1 := dl.Get(*address2, *address2)
	assert.NotNil(t, delegation1)
	assert.Equal(t, delegation1.Validator, address2)
	assert.Equal(t, delegation1.Delegator, address2)
	// should contain updated value, not original value
	assert.Equal(t, delegation1.Amount, &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)})
	assert.Equal(t, delegation1.Height, uint64(10))

	sort.Sort(byValidatorAndDelegator(dl))
	if !sort.IsSorted(byValidatorAndDelegator(dl)) {
		t.Fatal("delegation list is not sorted")
	}

	// add another entry
	dl.Set(&dtypes.DelegationV2{
		Validator:&types.Address{ChainId: chainID, Local: addr3.Local},
		Delegator:&types.Address{ChainId: chainID, Local: addr3.Local},
		Height: 10,
		Amount: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1)},
	})

	assert.Equal(t, 4, len(dl))

	sort.Sort(byValidatorAndDelegator(dl))
	assert.True(t, sort.IsSorted(byValidatorAndDelegator(dl)))
}

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
	validators := []*DposValidator{
		&DposValidator{
			PubKey: []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A="),
		},
		&DposValidator{
			PubKey: []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU="),
		},
		&DposValidator{
			PubKey: []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k="),
		},
		&DposValidator{
			PubKey: []byte("bOZnGz5QzPh7xFHKlqyFQqMeEsidI8XmWClLlWuS5dw=+k="),
		},
		&DposValidator{
			PubKey: []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dSlxe6Hd30ZuuYWgps"),
		},
	}

	sortedValidatores := sortValidators(validators)
	assert.True(t, sort.IsSorted(byPubkey(sortedValidatores)))

	sortedValidatores = append(sortedValidatores, &DposValidator{
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
