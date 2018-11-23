// +build secp256

package filepv

import (
	"io/ioutil"

	"github.com/loomnetwork/loomchain/privval/auth"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	fpv "github.com/tendermint/tendermint/privval"
)

type FilePV struct {
	*fpv.FilePV
}

func GenFilePV(filePath string) (*FilePV, error) {
	epv := fpv.GenFilePV(filePath)

	privKey := secp256k1.GenPrivKey()
	epv.Address = privKey.PubKey().Address()
	epv.PubKey = privKey.PubKey()
	epv.PrivKey = privKey

	return &FilePV{epv}, nil
}

func LoadFilePV(filePath string) (*FilePV, error) {
	_, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	pv := fpv.LoadFilePV(filePath)
	return &FilePV{pv}, nil
}

func NewFilePVSigner(pv *FilePV) auth.Signer {
	privKey := [32]byte(pv.GetPrivKey())
	return auth.NewSigner(privKey[:])
}

func (pv *FilePV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

func (pv *FilePV) GetPrivKey() secp256k1.PrivKeySecp256k1 {
	return pv.PrivKey.(secp256k1.PrivKeySecp256k1)
}

func (pv *FilePV) GetPubKeyBytes(pubKey crypto.PubKey) []byte {
	pub := pubKey.(secp256k1.PubKeySecp256k1)
	return pub[:]
}
