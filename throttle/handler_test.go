package throttle

import (
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/stretchr/testify/require"
)

const (
	period = 7
)

var (
	addr2 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr4 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr5 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c6")
	allowedDeplolyers = []loom.Address{addr2, addr3}
)

func TestDeployValidator(t *testing.T) {
	handler := NewOrginHandler(period, allowedDeplolyers, true, true)

	require.NoError(t, handler.ValidateDeployer(addr2))
	require.NoError(t, handler.ValidateDeployer(addr3))
	require.Error(t, handler.ValidateDeployer(addr5))
	require.NoError(t, handler.ValidateDeployer(addr2))
	require.NoError(t, handler.ValidateDeployer(addr3))

	require.NoError(t, handler.ValidateCaller(addr2))
	require.NoError(t, handler.ValidateCaller(addr3))
	require.NoError(t, handler.ValidateCaller(addr4))

	require.Error(t, handler.ValidateCaller(addr2))
	require.Error(t, handler.ValidateCaller(addr3))
	require.Error(t, handler.ValidateCaller(addr4))


	handler.Reset(5)

	require.NoError(t, handler.ValidateDeployer(addr2))
	require.NoError(t, handler.ValidateDeployer(addr3))
	require.Error(t, handler.ValidateDeployer(addr4))
	require.Error(t, handler.ValidateDeployer(addr5))
	require.NoError(t, handler.ValidateDeployer(addr2))
	require.NoError(t, handler.ValidateDeployer(addr3))

	require.Error(t, handler.ValidateCaller(addr2))
	require.Error(t, handler.ValidateCaller(addr3))
	require.Error(t, handler.ValidateCaller(addr4))
	require.NoError(t, handler.ValidateCaller(addr5))


	require.Error(t, handler.ValidateCaller(addr2))
	require.Error(t, handler.ValidateCaller(addr3))
	require.Error(t, handler.ValidateCaller(addr4))
	require.Error(t, handler.ValidateCaller(addr5))

	handler.Reset(123*period)

	require.NoError(t, handler.ValidateCaller(addr2))
	require.NoError(t, handler.ValidateCaller(addr3))
	require.NoError(t, handler.ValidateCaller(addr4))

	require.Error(t, handler.ValidateCaller(addr2))
	require.Error(t, handler.ValidateCaller(addr3))
	require.Error(t, handler.ValidateCaller(addr4))
}