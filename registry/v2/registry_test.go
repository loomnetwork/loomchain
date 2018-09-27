package registry

import (
	"context"
	"testing"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/loomnetwork/loomchain/registry"
)

func TestValidateName(t *testing.T) {
	assert.Nil(t, validateName("foo123"))
	assert.Nil(t, validateName("foo.bar"))

	assert.NotNil(t, validateName("foo@bar"))
}

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

type MockState struct {
	store *store.MemStore
}

func (s *MockState) Range(prefix []byte) plugin.RangeData {
	return s.store.Range(prefix)
}

func (s *MockState) Get(key []byte) []byte {
	return s.store.Get(key)
}

func (s *MockState) Has(key []byte) bool {
	return s.store.Has(key)
}

func (s *MockState) Validators() []*loom.Validator {
	return []*loom.Validator{}
}

func (s *MockState) SetValidatorPower(pubKey []byte, power int64) {
	// Do nothing
}

func (s *MockState) Set(key, value []byte) {
	s.store.Set(key, value)
}

func (s *MockState) Delete(key []byte) {
	s.store.Delete(key)
}

func (s *MockState) Block() types.BlockHeader {
	return types.BlockHeader{}
}

func (s *MockState) Context() context.Context {
	return nil
}

func (s *MockState) WithContext(ctx context.Context) loomchain.State {
	return nil
}

func TestContractAddressForSameName(t *testing.T) {
	mockState := MockState{store: store.NewMemStore()}

	reg := StateRegistry{State: &mockState}

	// Backward compatiblity test
	err := reg.Register("contract2", common.DefaultContractVersion, addr2, addr2)
	require.NoError(t, err)

	err = reg.Register("contract2", common.DefaultContractVersion, addr2, addr2)
	require.Error(t, err)

	// Sentinel version tag check
	addr, err := reg.Resolve("contract2", common.SentinelVersion)
	require.Error(t, err)

	err = reg.Register("contract1", "0.0.1", addr1, addr1)
	require.NoError(t, err)

	err = reg.Register("contract1", "0.0.1", addr1, addr1)
	require.Error(t, err)

	err = reg.Register("contract1", "0.0.2", addr2, addr2)
	require.NoError(t, err)

	// Sentinel version tag check
	addr, err = reg.Resolve("contract1", common.SentinelVersion)
	require.NoError(t, err)

	// Need to give address registered initially
	addr, err = reg.Resolve("contract1", common.DefaultContractVersion)
	require.NoError(t, err)
	assert.Equal(t, addr1.Compare(addr), 0)

	// Contract address is same for diff versions
	record, err := reg.GetRecord(addr1)
	require.NoError(t, err)
	assert.Equal(t, addr1.Compare(loom.UnmarshalAddressPB(record.Address)), 0)
	assert.Equal(t, addr1.Compare(loom.UnmarshalAddressPB(record.Owner)), 0)
}
