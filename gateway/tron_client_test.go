// +build evm

package gateway

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTronAddress(t *testing.T) {
	hex1 := "0x9570a09e55f932ccba8cc2e23739d2a88ef8cae1"
	hex2 := "419570a09e55f932ccba8cc2e23739d2a88ef8cae1"
	base58Addr := "TPbNc8baoTdFYYcLW73m38uBxuWZduDj6r"

	require.Equal(t, base58Addr, AddressHexToBase58(hex1))
	require.Equal(t, base58Addr, AddressHexToBase58(hex2))
	result, err := AddressBase58ToHex(base58Addr)
	require.NoError(t, err)
	require.Equal(t, hex2, result)

	hex3 := "416b201fb7b9f2b97bbdaf5e0920191229767c30ee"
	base58Addr2 := "TKjdnbJxP4yHeLTHZ86DGnFFY6QhTjuBv2"
	require.Equal(t, base58Addr2, AddressHexToBase58(hex3))
	result, err = AddressBase58ToHex(base58Addr2)
	require.NoError(t, err)
	require.Equal(t, hex3, result)

	priv1 := "a4ec684ac1b266438653f52ac010ac28b7248a4557dd18813fa38e3f03305d46"
	base58Addr3 := "TV9Q9MYgKJ89J7bDaqu51neUvcChCSq3XF"
	require.Equal(t, base58Addr3, AddressBase58FromPrivKey(priv1))
}
