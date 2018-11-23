// +build secp256

//Loom has a single private key type that is defined at compile time, by changing a build tag

package auth

import (
	"errors"

	"github.com/loomnetwork/go-loom/auth"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	ABCIPubKeyType = tmtypes.ABCIPubKeyTypeSecp256k1
	PubKeySize     = auth.Secp256k1PubKeyBytes
)

func NewSigner(privKey []byte) Signer {
	return auth.NewSigner(auth.SignerTypeSecp256k1, privKey)
}

func NewAuthKey() ([]byte, []byte, error) {
	pubKey, privKey := auth.GenSecp256k1Key()
	return pubKey, privKey, nil
}

func VerifyBytes(pubKey []byte, msg []byte, sig []byte) error {
	if len(pubKey) != auth.Secp256k1PubKeyBytes {
		return errors.New("invalid public key length")
	}

	if len(sig) != auth.Secp256k1SigBytes {
		return errors.New("invalid signature length")
	}

	if !auth.VerifyBytes(pubKey, msg, sig) {
		return errors.New("invalid signature")
	}

	return nil
}
