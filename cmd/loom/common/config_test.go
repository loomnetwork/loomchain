package common

import (
	"os"
	"testing"

	"github.com/loomnetwork/loomchain/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	exampleHsm = "loom_hsm_test"
)

func TestParseConfigFileName(t *testing.T) {
	conf_hsm := config.DefaultConfig()
	conf_hsm.WriteToFile(exampleHsm + ".yaml")
	v_actual := viper.New()
	PasreConfigFileName(exampleHsm, v_actual)
	assert.Equal(t, conf_hsm.HsmConfig.HsmDevType, v_actual.GetString("HsmConfig.HsmDevType"))
	_ = os.Remove(exampleHsm + ".yaml")
}
