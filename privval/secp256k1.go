package privval

import (
	"io/ioutil"

	"github.com/tendermint/tendermint/crypto/secp256k1"
	fpv "github.com/tendermint/tendermint/privval"
)

// ECFilePV is priv validator with secp256k1
type ECFilePV struct {
	*fpv.FilePV
}

// GenECFilePV generates priv validator with secp256k1
func GenECFilePV(filePath string) (*ECFilePV, error) {
	epv := fpv.GenFilePV(filePath)

	privKey := secp256k1.GenPrivKey()
	epv.Address = privKey.PubKey().Address()
	epv.PubKey = privKey.PubKey()
	epv.PrivKey = privKey

	return &ECFilePV{epv}, nil
}

// LoadECFilePV loads priv validator with secp256k1
func LoadECFilePV(filePath string) (*ECFilePV, error) {
	_, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	pv := fpv.LoadFilePV(filePath)
	return &ECFilePV{pv}, nil
}

// Reset priv validator with given height
func (pv *ECFilePV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

// GetPrivKey gets private key
func (pv *ECFilePV) GetPrivKey() secp256k1.PrivKeySecp256k1 {
	return pv.PrivKey.(secp256k1.PrivKeySecp256k1)
}

// Ecdsa signer implements the Signer interface using ecdsa keys
type Secp256k1Signer struct {
	privateKey secp256k1.PrivKeySecp256k1
}

func NewSecp256k1Signer(privateKey []byte) *Secp256k1Signer {
	secp256k1Signer := &Secp256k1Signer{}

	copy(secp256k1Signer.privateKey[:], privateKey)
	return secp256k1Signer
}

func (s *Secp256k1Signer) Sign(msg []byte) []byte {
	sig, err := s.privateKey.Sign(msg)
	if err != nil {
		return nil
	}
	return sig.Bytes()
}

func (s *Secp256k1Signer) PublicKey() []byte {
	return s.privateKey.PubKey().Bytes()
}
