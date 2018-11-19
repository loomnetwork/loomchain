package privval

import (
	"fmt"
	"os"

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

	if GetSecp256k1Enabled() {
		return GenECFilePV(filePath)
	}

	return GenFilePV(filePath)
}

// load priv validator
func LoadPrivVal(filePath string, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmConfig.HsmEnabled {
		return hsmpv.LoadHsmPV(hsmConfig, filePath)
	}

	if GetSecp256k1Enabled() {
		return LoadECFilePV(filePath)
	}

	return LoadFilePV(filePath)
}

func NewPrivValSigner(pv PrivValidator) auth.Signer {
	switch v := pv.(type) {
	case *hsmpv.YubiHsmPV:
		return hsmpv.NewYubiHsmSigner(v)
	case *FilePV:
		privKey := [64]byte(v.GetPrivKey())
		return auth.NewEd25519Signer(privKey[:])
	case *ECFilePV:
		privKey := [32]byte(v.GetPrivKey())
		return auth.NewSecp256k1Signer(privKey[:])
	default:
		panic(fmt.Errorf("Unknown PrivValidator implementation %T", v))
	}
}

func GetSecp256k1Enabled() bool {
	if os.Getenv(auth.EnableSecp256k1EnvVarName) == "1" {
		return true
	}

	return false
}
