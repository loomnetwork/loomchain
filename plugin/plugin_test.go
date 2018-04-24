package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFileName(t *testing.T) {
	info, err := parseFileName("hello.so.1.2.3")
	require.Nil(t, err)

	assert.Equal(t, "hello", info.Base)
	assert.Equal(t, ".so", info.Ext)
	assert.Equal(t, "1.2.3", info.Version)

	info, err = parseFileName("hello.1.2.3")
	require.Nil(t, err)

	assert.Equal(t, "hello", info.Base)
	assert.Equal(t, "", info.Ext)
	assert.Equal(t, "1.2.3", info.Version)

	_, err = parseFileName("hello.py")
	require.NotNil(t, err)
}
