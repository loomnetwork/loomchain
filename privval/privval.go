
package privval

import (
	"fmt"
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
func GenPrivVal(filePath string, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmConfig.HsmEnabled {
		return hsmpv.GenHsmPV(hsmConfig, filePath)
	}

	return GenFilePV(filePath)
}

// load priv validator
func LoadPrivVal(filePath string, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmConfig.HsmEnabled {
		return hsmpv.LoadHsmPV(hsmConfig, filePath)
	}

	return LoadFilePV(filePath)
}

func NewEd25519Signer(pv PrivValidator) auth.Signer {
	switch v := pv.(type) {
	case *hsmpv.YubiHsmPV:
		return hsmpv.NewYubiHsmSigner(v)
	case *FilePV:
		privKey := [64]byte(v.GetPrivKey())
		return auth.NewEd25519Signer(privKey[:])
	default:
		panic(fmt.Errorf("Unknown PrivValidator implementation %T", v))
	}
}
