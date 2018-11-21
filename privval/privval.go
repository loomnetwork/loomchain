package privval

import (
	"fmt"

	"github.com/tendermint/tendermint/crypto"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/tendermint/tendermint/types"

	filepv "github.com/loomnetwork/loomchain/privval/file"
	hsmpv "github.com/loomnetwork/loomchain/privval/hsm"
)

type PrivValidator interface {
	types.PrivValidator
	Save()
	Reset(height int64)
	GetPubKeyBytes(pubKey crypto.PubKey) []byte
}

func GenPrivVal(filePath string, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmConfig.HsmEnabled {
		return hsmpv.GenHsmPV(hsmConfig, filePath)
	}

	return filepv.GenFilePV(filePath)
}

func LoadPrivVal(filePath string, hsmConfig *hsmpv.HsmConfig) (PrivValidator, error) {
	if hsmConfig.HsmEnabled {
		return hsmpv.LoadHsmPV(hsmConfig, filePath)
	}

	return filepv.LoadFilePV(filePath)
}

func NewPrivValSigner(pv PrivValidator) auth.Signer {
	switch v := pv.(type) {
	case *hsmpv.YubiHsmPV:
		return hsmpv.NewYubiHsmSigner(v)
	case *filepv.FilePV:
		return filepv.NewFilePVSigner(v)
	default:
		panic(fmt.Errorf("Unknown PrivValidator implementation %T", v))
	}

	return nil
}
