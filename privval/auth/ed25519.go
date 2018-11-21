// +build !secp256

//Loom has a single private key type that is defined at compile time, by changing a build tag

package auth

import (
	"errors"

	"github.com/loomnetwork/go-loom/auth"
	tmtypes "github.com/tendermint/tendermint/types"
	"golang.org/x/crypto/ed25519"
)

const (
	ABCIPubKeyType = tmtypes.ABCIPubKeyTypeEd25519
	PubKeySize     = ed25519.PublicKeySize
)

func NewSigner(privKey []byte) Signer {
	return auth.NewSigner(auth.SignerTypeEd25519, privKey)
}

func NewAuthKey() ([]byte, []byte, error) {
	return ed25519.GenerateKey(nil)
}

func VerifyBytes(pubKey []byte, msg []byte, sig []byte) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return errors.New("invalid public key length")
	}

	if len(sig) != ed25519.SignatureSize {
		return errors.New("invalid signature length")
	}

	if !ed25519.Verify(pubKey, msg, sig) {
		return errors.New("invalid signature")
	}

	return nil
}
