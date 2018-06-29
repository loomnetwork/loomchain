package dpos

import (
	"sort"
	"testing"

	"github.com/loomnetwork/go-loom"

	dtypes "github.com/loomnetwork/go-loom/builtin/types/dpos"
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
		Address: &types.Address{chainID, addr2.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	cl.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{chainID, addr1.Local},
	})

	assert.Equal(t, 3, len(cl))

	// add duplicated entry
	cl.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	assert.Equal(t, 3, len(cl))

	sort.Sort(byAddress(cl))
	if !sort.IsSorted(byAddress(cl)) {
		t.Fatal("candidate list is not sorted")
	}

	// add another entry
	cl.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{chainID, addr4.Local},
	})
	assert.Equal(t, 4, len(cl))

	sort.Sort(byAddress(cl))
	assert.True(t, sort.IsSorted(byAddress(cl)))
}

func TestSortWitnessList(t *testing.T) {
	witnesses := []*Witness{
		&Witness{
			PubKey: []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A="),
		},
		&Witness{
			PubKey: []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU="),
		},
		&Witness{
			PubKey: []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k="),
		},
		&Witness{
			PubKey: []byte("bOZnGz5QzPh7xFHKlqyFQqMeEsidI8XmWClLlWuS5dw=+k="),
		},
		&Witness{
			PubKey: []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dSlxe6Hd30ZuuYWgps"),
		},
	}

	sortedWitnesses := sortWitnesses(witnesses)
	assert.True(t, sort.IsSorted(byPubkey(sortedWitnesses)))

	sortedWitnesses = append(sortedWitnesses, &Witness{
		PubKey: []byte("2AUfclH6vC7G2jkf7RxOTzhTYHVdE/2Qp5WSsK8m/tQ="),
	})

	sortedWitnesses = sortWitnesses(witnesses)
	assert.True(t, sort.IsSorted(byPubkey(sortedWitnesses)))
}

func TestGetSetCandidateList(t *testing.T) {
	var cands CandidateList
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{chainID, addr1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{chainID, addr2.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// duplicate
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	assert.Equal(t, 3, len(cands))

	// get
	cand1 := cands.Get(loom.Address{chainID, addr1.Local})
	assert.NotNil(t, cand1)
	assert.Equal(t, cand1.PubKey, pub1)
	assert.Equal(t, 0, cand1.Address.Local.Compare(addr1.Local))

	cand4 := cands.Get(loom.Address{chainID, addr4.Local})
	assert.Nil(t, cand4)

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{chainID, addr4.Local},
	})
	assert.Equal(t, 4, len(cands))
}

func TestSortCandidateList(t *testing.T) {
	var cands CandidateList

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{chainID, addr4.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub1,
		Address: &types.Address{chainID, addr1.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	cands.Set(&Candidate{
		PubKey:  pub2,
		Address: &types.Address{chainID, addr2.Local},
	})

	sortedCands := sortCandidates(cands)
	assert.True(t, sort.IsSorted(byAddress(sortedCands)))
}

func TestGetSetVoteList(t *testing.T) {
	var votes VoteList
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr1.Local},
	})
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr2.Local},
	})
	assert.Equal(t, 2, len(votes))
	vote1 := votes.Get(addr1)
	assert.NotNil(t, vote1)
	vote2 := votes.Get(addr2)
	assert.NotNil(t, vote2)
	vote3 := votes.Get(addr3)
	assert.Nil(t, vote3)

	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr3.Local},
	})
	vote3 = votes.Get(addr3)
	assert.NotNil(t, vote3)

	assert.Equal(t, 3, len(votes))

	vote3.Amount = uint64(21)
	votes.Set(vote3)

	nvote3 := votes.Get(addr3)
	assert.Equal(t, nvote3.Amount, vote3.Amount)
}

func TestSortVoteList(t *testing.T) {
	var votes VoteList
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr4.Local},
	})
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr3.Local},
	})
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr1.Local},
	})
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr2.Local},
	})

	sortedVotes := sortVotes(votes)
	assert.True(t, sort.IsSorted(byAddressAndAmount(sortedVotes)))
}
