// +build evm

package evm

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestSortSecureKeys(t *testing.T) {
	test1 := []kvPair{
		{[]byte("secure-key-q�����;� ��Z���'=��ks֝B"), []byte("data1")},
		{[]byte("secure-key-؀&*>�Y��F8I听Qia���SQ�6��f@"), []byte("data2")},
		{[]byte("secure-key-)\n��T�b��E��8o�K���H@�6/���c"), []byte("data3")},
		{[]byte("h����Ntԇ�ב��E��K]}�ɐW��a7��"), []byte("data4")},
		{[]byte("�牔!��FQ���e�8���M˫����ܤ�S"), []byte("data5")},
		{[]byte("�Ka����ͯ>/�� �\tߕ|���}j���<<�"), []byte("data6")},
		{[]byte("-�F�bt����S	�A������;BT�b�gF"), []byte("data7")},
	}
	test1 = sortKeys([]byte("secure-key-"), test1)

	test2 := []kvPair{
		{[]byte("secure-key-)\n��T�b��E��8o�K���H@�6/���c"), []byte("data3")},
		{[]byte("secure-key-q�����;� ��Z���'=��ks֝B"), []byte("data1")},
		{[]byte("secure-key-؀&*>�Y��F8I听Qia���SQ�6��f@"), []byte("data2")},
		{[]byte("h����Ntԇ�ב��E��K]}�ɐW��a7��"), []byte("data4")},
		{[]byte("�牔!��FQ���e�8���M˫����ܤ�S"), []byte("data5")},
		{[]byte("�Ka����ͯ>/�� �\tߕ|���}j���<<�"), []byte("data6")},
		{[]byte("-�F�bt����S	�A������;BT�b�gF"), []byte("data7")},
	}

	test2 = sortKeys([]byte("secure-key-"), test2)

	for i := 0; i < len(test1); i++ {
		require.Equal(t, 0, bytes.Compare(test1[i].key, test2[i].key))
	}
}
