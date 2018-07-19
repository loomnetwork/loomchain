// +build evm

package evm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// This test only verifies running a sort twice gives same result
func TestSortKeys(t *testing.T) {
	test1 := []kvPair{
		{[]byte("prefixFred"), []byte("data1")},
		{[]byte("noPrefixMary"), []byte("data2")},
		{[]byte("noPrefixJohn"), []byte("data3")},
		{[]byte("prefixSally"), []byte("data4")},
		{[]byte("noPrefixBob"), []byte("data5")},
		{[]byte("prefixAnne"), []byte("data6")},
	}
	test1 = sortKeys([]byte("prefix"), test1)

	test2 := []kvPair{
		{[]byte("prefixSally"), []byte("data4")},
		{[]byte("noPrefixMary"), []byte("data2")},
		{[]byte("noPrefixJohn"), []byte("data3")},
		{[]byte("prefixAnne"), []byte("data6")},
		{[]byte("noPrefixBob"), []byte("data5")},
		{[]byte("prefixFred"), []byte("data1")},
	}

	test2 = sortKeys([]byte("prefix"), test2)
	for i := 0; i < len(test1); i++ {
		require.Equal(t, 0, bytes.Compare(test1[i].key, test2[i].key))
	}
}

// This test verifies that prefixed items are sorted by ascending order
func TestSortKeys2(t *testing.T) {
	test1 := []kvPair{
		{[]byte("prefixSally"), []byte("data4")},
		{[]byte("prefixFred"), []byte("data1")},
		{[]byte("noPrefixMary"), []byte("data2")},
		{[]byte("noPrefixJohn"), []byte("data3")},
		{[]byte("noPrefixBob"), []byte("data5")},
		{[]byte("prefixAnne"), []byte("data6")},
	}
	test1 = sortKeys([]byte("prefix"), test1)

	test2 := []kvPair{
		{[]byte("prefixAnne"), []byte("data6")},
		{[]byte("prefixFred"), []byte("data1")},
		{[]byte("noPrefixMary"), []byte("data2")},
		{[]byte("noPrefixJohn"), []byte("data3")},
		{[]byte("noPrefixBob"), []byte("data5")},
		{[]byte("prefixSally"), []byte("data4")},
	}

	for i := 0; i < len(test1); i++ {
		require.Equal(t, string(test2[i].key), string(test1[i].key))
	}
}
