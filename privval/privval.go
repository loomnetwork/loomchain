package privval

import (
	"fmt"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/tendermint/tendermint/types"

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
		return auth.NewSigner(auth.SignerTypeYubiHsm, v.PrivateKey)
	case *FilePV:
		privKey := [64]byte(v.GetPrivKey())
		return auth.NewSigner(auth.SignerTypeEd25519, privKey[:])
	default:
		panic(fmt.Errorf("Unknown PrivValidator implementation %T", v))
	}
}
