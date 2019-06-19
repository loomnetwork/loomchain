package common

import (
	"os"
	"testing"

	"github.com/loomnetwork/loomchain/config"
	"github.com/stretchr/testify/assert"
)

const (
	example_loom_hsm = "loom_hsm"
	example_loom     = "loom"
)

func TestParseConfigWith_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(example_loom + ".yaml")
	conf_hsm := config.DefaultConfig()
	conf_hsm.HsmConfig.HsmSignKeyID = 1010
	conf_hsm.WriteToFile(example_loom_hsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf_hsm, actual)
	_ = os.Remove(example_loom_hsm + ".yaml")
	_ = os.Remove(example_loom + ".yaml")
}

func TestParseConfigWithout_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(example_loom + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(example_loom + ".yaml")
}

func TestParseConfigWith_OtherKey_In_HSMfile(t *testing.T) {
	conf := config.DefaultConfig()
	conf.WriteToFile(example_loom + ".yaml")
	conf_hsm := config.DefaultConfig()
	conf_hsm.DPOSVersion = 100
	conf_hsm.WriteToFile(example_loom_hsm + ".yaml")
	actual, err := ParseConfig()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, conf, actual)
	_ = os.Remove(example_loom_hsm + ".yaml")
	_ = os.Remove(example_loom + ".yaml")
}
