package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilename        = "testDefault"
	testExampleFilename = "testExample"
	exampleLoomYaml     = "loom.example"
)

func TestConfig(t *testing.T) {
	_ = os.Remove(testFilename + ".yaml")
	_ = os.Remove(testExampleFilename + ".yaml")

	confDef := DefaultConfig()
	require.NoError(t, confDef.WriteToFile(testFilename+".yaml"))
	confRead, err := ParseConfigFrom(testFilename)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(confDef, confRead))

	exampleRead, err := ParseConfigFrom(exampleLoomYaml)
	require.NoError(t, err)
	require.NoError(t, exampleRead.WriteToFile(testExampleFilename+".yaml"))
	confRead, err = ParseConfigFrom(testExampleFilename)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(exampleRead, confRead))
}
