// +build !secp256

package filepv

import (
	"io/ioutil"

	"github.com/loomnetwork/go-loom/auth"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	fpv "github.com/tendermint/tendermint/privval"
)

type FilePV struct {
	*fpv.FilePV
}

func GenFilePV(filePath string) (*FilePV, error) {
	pv := fpv.GenFilePV(filePath)
	return &FilePV{pv}, nil
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
	privKey := [64]byte(pv.GetPrivKey())
	return auth.NewSigner(privKey[:])
}

func (pv *FilePV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

func (pv *FilePV) GetPrivKey() ed25519.PrivKeyEd25519 {
	return pv.PrivKey.(ed25519.PrivKeyEd25519)
}

func (pv *FilePV) GetPubKeyBytes(pubKey crypto.PubKey) []byte {
	pub := pubKey.(ed25519.PubKeyEd25519)
	return pub[:]
}
