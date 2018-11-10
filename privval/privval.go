
package privval

import (
	"github.com/tendermint/tendermint/types"
	"github.com/loomnetwork/go-loom/auth"

	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
)

type PrivValidator interface {
	types.PrivValidator
	Save()
	Reset(height int64)
}

// generate priv validator while generating ed25519 keypair
func GenPrivVal(filePath string, hsmEnabled bool, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmEnabled {
		return hsmpv.GenHsmPV(hsmConfig, filePath)
	}

	return GenFilePV(filePath)
}

// load priv validator
func LoadPrivVal(filePath string, hsmEnabled bool, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmEnabled {
		return hsmpv.LoadHsmPV(hsmConfig, filePath)
	}

	return LoadFilePV(filePath)
}

func NewEd25519Signer(pv PrivValidator, hsmEnabled bool) auth.Signer {
	if hsmEnabled {
		return hsmpv.NewYubiHsmSigner(pv.(*hsmpv.YubiHsmPV))
	}

	privKey := [64]byte(pv.(*FilePV).GetPrivKey())
	return auth.NewEd25519Signer(privKey[:])
}
