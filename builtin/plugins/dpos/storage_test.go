package dpos

import (
	"bytes"
	"sort"
	"testing"

	"github.com/loomnetwork/go-loom"

	dtypes "github.com/loomnetwork/go-loom/builtin/types/dpos"
	"github.com/loomnetwork/go-loom/types"
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
		Address: &types.Address{
			ChainId: "default",
			Local:   loom.LocalAddressFromPublicKey([]byte("bykX4FC6tEU0vL6jUxuUzOpnwZsZrROQQSEUD0/vbY0=")),
		},
		PubKey: []byte("bykX4FC6tEU0vL6jUxuUzOpnwZsZrROQQSEUD0/vbY0="),
	})
	cl.Set(&Candidate{
		Address: &types.Address{
			ChainId: "default",
			Local:   loom.LocalAddressFromPublicKey([]byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU=")),
		},
		PubKey: []byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU="),
	})
	cl.Set(&Candidate{
		Address: &types.Address{
			ChainId: "default",
			Local:   loom.LocalAddressFromPublicKey([]byte("yobKYwqgZ1ADufBfPFobry6Gakuw0yrtJtUu90kl0lU=")),
		},
		PubKey: []byte("yobKYwqgZ1ADufBfPFobry6Gakuw0yrtJtUu90kl0lU="),
	})
	if len(cl) != 3 {
		t.Errorf("want candidate list len 3, got %d", len(cl))
	}

	// add duplicated entry
	cl.Set(&Candidate{
		Address: &types.Address{
			ChainId: "default",
			Local:   loom.LocalAddressFromPublicKey([]byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU=")),
		},
		PubKey: []byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU="),
	})
	if len(cl) != 3 {
		t.Errorf("want candidate list len 3, got %d", len(cl))
	}

	sort.Sort(byAddress(cl))

	if !bytes.Equal(cl[0].PubKey, []byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU=")) {
		t.Errorf("wrong sorted list")
	}

	if !bytes.Equal(cl[1].PubKey, []byte("yobKYwqgZ1ADufBfPFobry6Gakuw0yrtJtUu90kl0lU=")) {
		t.Errorf("wrong sorted list")
	}

	if !bytes.Equal(cl[2].PubKey, []byte("bykX4FC6tEU0vL6jUxuUzOpnwZsZrROQQSEUD0/vbY0=")) {
		t.Errorf("wrong sorted list")
	}

	// add another entry
	cl.Set(&Candidate{
		Address: &types.Address{
			ChainId: "default",
			Local:   loom.LocalAddressFromPublicKey([]byte("O/2v/slTum2wnmATyyJk40MnZ97oG7+PYEoR0dZgvqQ=")),
		},
		PubKey: []byte("O/2v/slTum2wnmATyyJk40MnZ97oG7+PYEoR0dZgvqQ="),
	})
	if len(cl) != 4 {
		t.Errorf("want candidate list len 4, got %d", len(cl))
	}

	sort.Sort(byAddress(cl))

	if !bytes.Equal(cl[0].PubKey, []byte("3NratSDT1L4R7XXJ5rYEirt8zFNCXXyMn3OyFOHFymU=")) {
		t.Errorf("wrong sorted list")
	}

	if !bytes.Equal(cl[1].PubKey, []byte("O/2v/slTum2wnmATyyJk40MnZ97oG7+PYEoR0dZgvqQ=")) {
		t.Errorf("wrong sorted list")
	}

	if !bytes.Equal(cl[2].PubKey, []byte("yobKYwqgZ1ADufBfPFobry6Gakuw0yrtJtUu90kl0lU=")) {
		t.Errorf("wrong sorted list")
	}

	if !bytes.Equal(cl[3].PubKey, []byte("bykX4FC6tEU0vL6jUxuUzOpnwZsZrROQQSEUD0/vbY0=")) {
		t.Errorf("wrong sorted list")
	}
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

	sort.Sort(byPubkey(witnesses))

	// check order
	if !bytes.Equal(witnesses[0].PubKey, []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dSlxe6Hd30ZuuYWgps")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[1].PubKey, []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[2].PubKey, []byte("bOZnGz5QzPh7xFHKlqyFQqMeEsidI8XmWClLlWuS5dw=+k=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[3].PubKey, []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[4].PubKey, []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU=")) {
		t.Errorf("wrong sorted list")
	}

	witnesses = append(witnesses, &Witness{
		PubKey: []byte("2AUfclH6vC7G2jkf7RxOTzhTYHVdE/2Qp5WSsK8m/tQ="),
	})
	if !bytes.Equal(witnesses[5].PubKey, []byte("2AUfclH6vC7G2jkf7RxOTzhTYHVdE/2Qp5WSsK8m/tQ=")) {
		t.Errorf("wrong sorted list")
	}

	sort.Sort(byPubkey(witnesses))
	if !bytes.Equal(witnesses[0].PubKey, []byte("2AUfclH6vC7G2jkf7RxOTzhTYHVdE/2Qp5WSsK8m/tQ=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[1].PubKey, []byte("5wYR5atUGpnpZ+oerOZ8hi3B4dSlxe6Hd30ZuuYWgps")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[2].PubKey, []byte("ZkBHnAw9XgBLMRxbFwH4ZEKoSNIpSeCZw0L0suu98+k=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[3].PubKey, []byte("bOZnGz5QzPh7xFHKlqyFQqMeEsidI8XmWClLlWuS5dw=+k=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[4].PubKey, []byte("emvRy1THBgGbNw/j1m5hqpXaVIZLHVz/GHQ58mxyc3A=")) {
		t.Errorf("wrong sorted list")
	}
	if !bytes.Equal(witnesses[5].PubKey, []byte("oTFzT+lt+ztuUQd9yuQbPAdZPmezuoOtOFCUULSqgmU=")) {
		t.Errorf("wrong sorted list")
	}
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
	if len(cands) != 3 {
		t.Fatalf("want candidate list len 3, got %d", len(cands))
	}
	// duplicate
	cands.Set(&Candidate{
		PubKey:  pub3,
		Address: &types.Address{chainID, addr3.Local},
	})
	if len(cands) != 3 {
		t.Fatalf("want candidate list len 3, got %d", len(cands))
	}

	// get
	cand1 := cands.Get(loom.Address{chainID, addr1.Local})
	if cand1 == nil {
		t.Fatalf("want candidate addr %s, got nil", addr1)
	}
	if bytes.Compare(cand1.PubKey, pub1) != 0 {
		t.Errorf("want same pub key")
	}
	if cand1.Address.Local.Compare(addr1.Local) != 0 {
		t.Errorf("want same address")
	}

	cand4 := cands.Get(loom.Address{chainID, addr4.Local})
	if cand4 != nil {
		t.Errorf("want nil candidate, got %v", cand4)
	}

	cands.Set(&Candidate{
		PubKey:  pub4,
		Address: &types.Address{chainID, addr4.Local},
	})
	if len(cands) != 4 {
		t.Fatalf("want candidate list len 4, got %d", len(cands))
	}
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

	// before sort
	if cands[0].Address.Local.Compare(addr4.Local) != 0 {
		t.Errorf("candidate should be unsorted")
	}
	if cands[1].Address.Local.Compare(addr1.Local) != 0 {
		t.Errorf("candidate should be unsorted")
	}
	if cands[2].Address.Local.Compare(addr3.Local) != 0 {
		t.Errorf("candidate should be unsorted")
	}
	if cands[3].Address.Local.Compare(addr2.Local) != 0 {
		t.Errorf("candidate should be unsorted")
	}

	sort.Sort(byAddress(cands))
	// sorted by address asc
	if cands[0].Address.Local.Compare(addr1.Local) != 0 {
		t.Errorf("wrong sorting order")
	}
	if cands[1].Address.Local.Compare(addr2.Local) != 0 {
		t.Errorf("wrong sorting order")
	}
	if cands[2].Address.Local.Compare(addr3.Local) != 0 {
		t.Errorf("wrong sorting order")
	}
	if cands[3].Address.Local.Compare(addr4.Local) != 0 {
		t.Errorf("wrong sorting order")
	}
}

func TestGetSetVoteList(t *testing.T) {
	var votes VoteList
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr1.Local},
	})
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr2.Local},
	})
	if len(votes) != 2 {
		t.Fatalf("want votes len 2")
	}
	vote1 := votes.Get(addr1)
	if vote1 == nil {
		t.Fatal("vote 1 should not be nil")
	}
	vote2 := votes.Get(addr2)
	if vote2 == nil {
		t.Fatal("vote 2 should not be nil")
	}
	vote3 := votes.Get(addr3)
	if vote3 != nil {
		t.Fatal("vote 3 should be nil")
	}
	votes.Set(&dtypes.Vote{
		VoterAddress: &types.Address{chainID, addr3.Local},
	})
	vote3 = votes.Get(addr3)
	if vote3 == nil {
		t.Fatal("vote 3 should not be nil")
	}
	if len(votes) != 3 {
		t.Fatalf("want votes len 3")
	}

	vote3.Amount = uint64(21)
	votes.Set(vote3)

	nvote3 := votes.Get(addr3)
	if nvote3.Amount != vote3.Amount {
		t.Fatalf("want vote amount %d, got %d", vote3.Amount, nvote3.Amount)
	}
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

	// before sort
	if votes[0].VoterAddress.Local.Compare(addr4.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[1].VoterAddress.Local.Compare(addr3.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[2].VoterAddress.Local.Compare(addr1.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[3].VoterAddress.Local.Compare(addr2.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}

	// after sort
	sort.Sort(byAddressAndAmount(votes))
	if votes[0].VoterAddress.Local.Compare(addr1.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[1].VoterAddress.Local.Compare(addr2.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[2].VoterAddress.Local.Compare(addr3.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
	if votes[3].VoterAddress.Local.Compare(addr4.Local) != 0 {
		t.Errorf("votes should be unsorted")
	}
}
