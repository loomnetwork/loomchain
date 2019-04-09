package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilename        = "testDefault"
	testExampleFilename = "testExample"
	exampleLoomYaml     = "loom.example"
	contractLoadConfigYaml  = "contractLoad"
)

func TestConfig(t *testing.T) {
	_ = os.Remove(testFilename + ".yaml")
	_ = os.Remove(testExampleFilename + ".yaml")

	confDef := DefaultConfig()
	require.NoError(t, confDef.WriteToFile(testFilename+".yaml"))
	confRead, err := ParseConfigFrom(testFilename)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(confDef, confRead))
	confRead, err = ParseConfigFrom(contractLoadConfigYaml)
	require.NoError(t, err)
	assert.Equal(t,"static",confRead.ContractLoaders[0],"Test order of Loader 1")
	assert.Equal(t,"external",confRead.ContractLoaders[1],"Test orderof Loader 2")
	assert.Equal(t,"dynamic",confRead.ContractLoaders[2],"Test order of Loader 3")
	exampleRead, err := ParseConfigFrom(exampleLoomYaml)
	require.NoError(t, err)
	require.NoError(t, exampleRead.WriteToFile(testExampleFilename+".yaml"))
	confRead, err = ParseConfigFrom(testExampleFilename)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(exampleRead, confRead))
}
