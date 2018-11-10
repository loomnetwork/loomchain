
package privval

import (
	"io/ioutil"

	fpv "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

// file-based priv validator
type FilePV struct {
	pv *fpv.FilePV
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
	pv.pv.LastHeight = height
	pv.pv.LastRound = 0
	pv.pv.LastStep = 0
}

// get private key
func (pv *FilePV) GetPrivKey() ed25519.PrivKeyEd25519 {
	return pv.pv.PrivKey.(ed25519.PrivKeyEd25519)
}

// get address
func (pv *FilePV) GetAddress () types.Address {
	return pv.pv.GetAddress()
}

// get public key
func (pv *FilePV) GetPubKey() crypto.PubKey {
	return pv.pv.GetPubKey()
}

// save priv validator
func (pv *FilePV) Save() {
	pv.pv.Save()
}

// sign vote
func (pv *FilePV) SignVote(chainID string, vote *types.Vote) error {
	return pv.pv.SignVote(chainID, vote)
}

// sign proposal
func (pv *FilePV) SignProposal(chainID string, proposal *types.Proposal) error {
	return pv.pv.SignProposal(chainID, proposal)
}

// sign heartbeat
func (pv *FilePV) SignHeartbeat(chainID string, heartbeat *types.Heartbeat) error {
	return pv.pv.SignHeartbeat(chainID, heartbeat)
}
