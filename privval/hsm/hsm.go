package hsmpv

import (
	"errors"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/types"

	amino "github.com/tendermint/go-amino"
)

// amino route
const (
	Ed25519PrivKeyAminoRoute   = "tendermint/PrivKeyEd25519"
	Ed25519PubKeyAminoRoute    = "tendermint/PubKeyEd25519"
	Ed25519SignatureAminoRoute = "tendermint/SignatureEd25519"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		Ed25519PubKeyAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		Ed25519PrivKeyAminoRoute, nil)
}

// HsmPrivVal interface
type HsmPrivVal interface {
	types.PrivValidator

	GenPrivVal(filePath string) error
	LoadPrivVal(filePath string) error
	Save()
	Reset(height int64)
	Destroy()
}

// GenHsmPV generates priv validator with ed25519 keypair
func GenHsmPV(hsmConfig *HsmConfig, filePath string) (HsmPrivVal, error) {
	var pv HsmPrivVal
	var err error

	// load configuration
	if hsmConfig.HsmDevType == HsmDevTypeYubi {
		pv = NewYubiHsmPV(hsmConfig.HsmConnURL, hsmConfig.HsmAuthKeyID, hsmConfig.HsmDevLoginCred, hsmConfig.HsmSignKeyID)
	} else {
		return nil, errors.New("Unsupported HSM type")
	}

	if err = pv.GenPrivVal(filePath); err != nil {
		return nil, err
	}

	return pv, nil
}

// LoadHsmPV loads parameters from priv validator file
func LoadHsmPV(hsmConfig *HsmConfig, filePath string) (HsmPrivVal, error) {
	var pv HsmPrivVal

	// load configuration
	if hsmConfig.HsmDevType == HsmDevTypeYubi {
		pv = NewYubiHsmPV(hsmConfig.HsmConnURL, hsmConfig.HsmAuthKeyID, hsmConfig.HsmDevLoginCred, hsmConfig.HsmSignKeyID)
	} else {
		return nil, errors.New("Unsupported HSM type")
	}

	if err := pv.LoadPrivVal(filePath); err != nil {
		return nil, err
	}

	return pv, nil
}
