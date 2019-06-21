package common

import (
	"os"
	"testing"

	"github.com/loomnetwork/loomchain/config"
	"github.com/stretchr/testify/assert"
)

const (
	exampleLoomHsm = "loom_hsm"
	exampleLoom    = "loom"
)

func TestParseConfigWithHSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	confHsm := config.DefaultConfig()
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
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(exampleLoom + ".yaml")
}

func TestParseConfigWithOtherKeyInHSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	confHsm := config.DefaultConfig()
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
