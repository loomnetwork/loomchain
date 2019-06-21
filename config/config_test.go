package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

const (
	testFilename        = "testDefault"
	testExampleFilename = "testExample"
	exampleLoomYaml     = "loom.example"
	exampleLoomHsm      = "loom_hsm"
	exampleLoom         = "loom"
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
	assert.Equal(t, "static", exampleRead.ContractLoaders[0], "Test order of Loader 1")
	assert.Equal(t, "dynamic", exampleRead.ContractLoaders[1], "Test order of Loader 2")
	assert.Equal(t, "external", exampleRead.ContractLoaders[2], "Test order of Loader 3")
	require.NoError(t, exampleRead.WriteToFile(testExampleFilename+".yaml"))
	confRead, err = ParseConfigFrom(testExampleFilename)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(exampleRead, confRead))
}

func TestParseConfigWithHSMfile(t *testing.T) {
	conf := DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	confHsm := DefaultConfig()
	confHsm.HsmConfig.HsmSignKeyID = 1010
	confHsm.WriteToHsmFile(exampleLoomHsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, confHsm, actual)
	_ = os.Remove(exampleLoomHsm + ".yaml")
	_ = os.Remove(exampleLoom + ".yaml")
}

func TestParseConfigWithoutHSMfile(t *testing.T) {
	conf := DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(exampleLoom + ".yaml")
}

func TestParseConfigWithOtherKeyInHSMfile(t *testing.T) {
	conf := DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	confHsm := DefaultConfig()
	confHsm.DPOSVersion = 100
	confHsm.DeployEnabled = false
	confHsm.WriteToHsmFile(exampleLoomHsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(exampleLoomHsm + ".yaml")
	_ = os.Remove(exampleLoom + ".yaml")
}
