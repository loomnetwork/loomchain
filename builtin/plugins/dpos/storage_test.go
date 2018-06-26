package dpos

import (
	"bytes"
	"sort"
	"testing"

	"github.com/loomnetwork/go-loom"

	"github.com/loomnetwork/go-loom/types"
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
