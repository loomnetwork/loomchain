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

func TestParseConfigWith_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	confHsm := config.DefaultConfig()
	confHsm.HsmConfig.HsmSignKeyID = 1010
	confHsm.WriteToFile(exampleLoomHsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, confHsm, actual)
	_ = os.Remove(exampleLoomHsm + ".yaml")
	_ = os.Remove(exampleLoom + ".yaml")
}

func TestParseConfigWithout_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(exampleLoom + ".yaml")
}

func TestParseConfigWith_OtherKey_In_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(exampleLoom + ".yaml")
	conf_hsm := config.DefaultConfig()
	conf_hsm.DPOSVersion = 100
	conf_hsm.WriteToFile(exampleLoomHsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(exampleLoomHsm + ".yaml")
	_ = os.Remove(exampleLoom + ".yaml")
}
