// +build secp256

//Loom has a single private key type that is defined at compile time, by changing a build tag

package auth

import (
	"errors"

	"github.com/loomnetwork/go-loom/auth"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	ABCIPubKeyType = tmtypes.ABCIPubKeyTypeSecp256k1
	PubKeySize     = secp256k1.PubKeySecp256k1Size
)

type PubKeyType = secp256k1.PubKeySecp256k1

func VerifyBytes(pubKey []byte, msg []byte, sig []byte) error {
	if len(tx.PublicKey) != secp256k1.PubKeySecp256k1Size {
		return errors.New("invalid public key length")
	}

	secp256k1PubKey := secp256k1.PubKeySecp256k1{}
	copy(secp256k1PubKey[:], tx.PublicKey[:])
	secp256k1Signature := secp256k1.SignatureSecp256k1FromBytes(tx.Signature)
	if !secp256k1PubKey.VerifyBytes(tx.Inner, secp256k1Signature) {
		return errors.New("invalid signature")
	}

	return nil
}

func NewSigner(privKey []byte) Signer {
	var err error
	if privKey == nil {
		privKey = secp256k1.GenPrivKey().Bytes()
	}

	return auth.NewSigner(privKey)
}

func NewAuthKey() ([]byte, []byte, error) {
	privKey := secp256k1.GenPrivKey()
	return privKey.PubKey().Bytes(), privKey.Bytes(), nil
}
