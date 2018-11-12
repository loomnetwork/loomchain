
package privval

import (
	"io/ioutil"

	fpv "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

// file-based priv validator
type FilePV struct {
	*fpv.FilePV
}

// get priv validator
func GenFilePV(filePath string) (*FilePV, error) {
	pv := fpv.GenFilePV(filePath)
	return &FilePV{pv}, nil
}

// load priv validator
func LoadFilePV(filePath string) (*FilePV, error) {
	_, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	pv := fpv.LoadFilePV(filePath)
	return &FilePV{pv}, nil
}

// reset priv validator
func (pv *FilePV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

// get private key
func (pv *FilePV) GetPrivKey() ed25519.PrivKeyEd25519 {
	return pv.PrivKey.(ed25519.PrivKeyEd25519)
}
