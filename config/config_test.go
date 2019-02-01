package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilename = "testloom.yaml"
)

func TestConfig(t *testing.T) {
	confDef := DefaultConfig()
	err := confDef.WriteToFile(testFilename)
	require.NoError(t, err)
	confRead, err := ParseConfigFrom("testloom")
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(confDef, confRead))
}
