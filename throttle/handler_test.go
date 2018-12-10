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
	handler := NewOriginValidator(period, allowedDeplolyers, true, true)
	nonce2:=uint64(1)
	nonce3:=uint64(1)
	nonce4:=uint64(1)
	nonce5:=uint64(1)



	require.NoError(t, handler.validateDeployer(addr2))
	require.NoError(t, handler.validateDeployer(addr3))
	require.Error(t, handler.validateDeployer(addr5))
	require.NoError(t, handler.validateDeployer(addr2))
	require.NoError(t, handler.validateDeployer(addr3))

	require.NoError(t, handler.validateCaller(addr2,nonce2))
	require.NoError(t, handler.validateCaller(addr2,nonce2))
	nonce2++
	require.NoError(t, handler.validateCaller(addr3,nonce3))
	require.NoError(t, handler.validateCaller(addr3,nonce3))
	nonce3++
	require.NoError(t, handler.validateCaller(addr4,nonce4))
	require.NoError(t, handler.validateCaller(addr4,nonce4))
	nonce4++
	require.Error(t, handler.validateCaller(addr2,nonce2))
	nonce2++
	require.Error(t, handler.validateCaller(addr3,nonce3))
	nonce3++
	require.Error(t, handler.validateCaller(addr4,nonce4))
	nonce4++

	handler.Reset(5)

	require.NoError(t, handler.validateDeployer(addr2))
	require.NoError(t, handler.validateDeployer(addr3))
	require.Error(t, handler.validateDeployer(addr4))
	require.Error(t, handler.validateDeployer(addr5))
	require.NoError(t, handler.validateDeployer(addr2))
	require.NoError(t, handler.validateDeployer(addr3))

	require.Error(t, handler.validateCaller(addr2,nonce2))
	nonce2++
	require.Error(t, handler.validateCaller(addr3,nonce3))
	nonce3++
	require.Error(t, handler.validateCaller(addr4,nonce4))
	nonce4++
	require.NoError(t, handler.validateCaller(addr5,nonce5))
	nonce5++

	require.Error(t, handler.validateCaller(addr2,nonce2))
	nonce2++
	require.Error(t, handler.validateCaller(addr3,nonce3))
	nonce3++
	require.Error(t, handler.validateCaller(addr4,nonce4))
	nonce4++
	require.Error(t, handler.validateCaller(addr5,nonce5))
	nonce5++

	handler.Reset(123*period)

	require.NoError(t, handler.validateCaller(addr2,nonce2))
	require.NoError(t, handler.validateCaller(addr2,nonce2))
	nonce2++
	require.NoError(t, handler.validateCaller(addr3,nonce3))
	require.NoError(t, handler.validateCaller(addr3,nonce3))
	nonce3++
	require.NoError(t, handler.validateCaller(addr4,nonce4))
	require.NoError(t, handler.validateCaller(addr4,nonce4))
	nonce4++
}