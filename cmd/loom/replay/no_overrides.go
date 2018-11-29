// +build !plasmachain

package replay

import (
	"github.com/loomnetwork/loomchain/plugin"
)

func ContractOverrides() plugin.ContractOverrideMap {
	return nil
}
