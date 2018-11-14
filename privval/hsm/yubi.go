package hsmpv

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/certusone/yubihsm-go"
	"github.com/certusone/yubihsm-go/commands"
	"github.com/certusone/yubihsm-go/connector"
)

// fixme: these constants should be set by configuration
const (
	YUBIHSM_SIGNKEY_LABEL = "loomchain-hsm-pv"
)

// YubiHSM structure
type YubiHsmPV struct {
	sessionMgr *yubihsm.SessionManager

	hsmURL    string
	authKeyID uint16
	password  string

	LastHeight int64 `json:"last_height"`
	LastRound  int   `json:"last_round"`
	LastStep   int8  `json:"last_step"`

	LastSignature []byte       `json:"last_signature,omitempty"`
	LastSignBytes cmn.HexBytes `json:"last_signbytes,omitempty"`

	Address   types.Address `json:"address"`
	SignKeyID uint16        `json:"key_id"`

	PubKey crypto.PubKey `json:"pub_key"`

	filePath string
	mtx      sync.Mutex
}

// TODO: type ?
const (
	stepNone      int8 = 0 // Used to distinguish the initial state
	stepPropose   int8 = 1
	stepPrevote   int8 = 2
	stepPrecommit int8 = 3
)

func voteToStep(vote *types.Vote) int8 {
	switch vote.Type {
	case types.PrevoteType:
		return stepPrevote
	case types.PrecommitType:
		return stepPrecommit
	default:
		cmn.PanicSanity("Unknown vote type")
		return 0
	}
}

// create a new instance of YubiHSM priv validator
func NewYubiHsmPV(connURL string, authKeyID uint16, password string, signKeyId uint16) *YubiHsmPV {
	return &YubiHsmPV{
		hsmURL:    connURL,
		authKeyID: authKeyID,
		password:  password,
		SignKeyID: signKeyId,
	}
}

// generate YubiHSM priv validator
func (pv *YubiHsmPV) GenPrivVal(filePath string) error {
	var err error

	// init yubi HSM pv
	err = pv.Init()
	if err != nil {
		return err
	}

	// generate keypair
	if pv.SignKeyID == 0 {
		err = pv.genEd25519KeyPair()
		if err != nil {
			pv.Destroy()
			return err
		}
	}

	// export public key
	err = pv.exportEd25519PubKey()
	if err != nil {
		pv.Destroy()
		return err
	}

	pv.filePath = filePath
	return nil
}

// load YubiHSM priv validator from file
func (pv *YubiHsmPV) LoadPrivVal(filePath string) error {
	// parse priv validator file
	pvJsonBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = cdc.UnmarshalJSON(pvJsonBytes, &pv)
	if err != nil {
		return err
	}

	// init YubiHSM priv validator
	err = pv.Init()
	if err != nil {
		return err
	}

	// export pubkey
	err = pv.exportEd25519PubKey()
	if err != nil {
		pv.Destroy()
		return err
	}

	pv.filePath = filePath
	return nil
}

// init YubiHsm priv validator
func (pv *YubiHsmPV) Init() error {
	var err error

	httpConnector := connector.NewHTTPConnector(pv.hsmURL)
	pv.sessionMgr, err = yubihsm.NewSessionManager(httpConnector, pv.authKeyID, pv.password)
	if err != nil {
		return err
	}

	return nil
}

// destroy YubiHsm priv validator
func (pv *YubiHsmPV) Destroy() {
	if pv.sessionMgr == nil {
		return
	}

	pv.sessionMgr.Destroy()
}

// reset parameters
func (pv *YubiHsmPV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

// save YubiHsm priv validator info
func (pv *YubiHsmPV) Save() {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	pv.save()
}

func (pv *YubiHsmPV) save() {
	outFile := pv.filePath
	if outFile == "" {
		panic("Cannot save YubiHSM PrivValidator: filePath not set")
	}

	jsonBytes, err := cdc.MarshalJSONIndent(pv, "", "  ")
	if err != nil {
		panic(err)
	}

	err = cmn.WriteFileAtomic(outFile, jsonBytes, 0600)
	if err != nil {
		panic(err)
	}
}

// get public key
func (pv *YubiHsmPV) GetPubKey() crypto.PubKey {
	return pv.PubKey
}

// get address
func (pv *YubiHsmPV) GetAddress() types.Address {
	return pv.PubKey.Address()
}

// sign vote
func (pv *YubiHsmPV) SignVote(chainID string, vote *types.Vote) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	if err := pv.signVote(chainID, vote); err != nil {
		return fmt.Errorf("Error signing vote: %v", err)
	}
	return nil
}

// sign proposal
func (pv *YubiHsmPV) SignProposal(chainID string, proposal *types.Proposal) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	if err := pv.signProposal(chainID, proposal); err != nil {
		return fmt.Errorf("Error signing proposal: %v", err)
	}
	return nil
}

// sign heartbeat
func (pv *YubiHsmPV) SignHeartbeat(chainID string, heartbeat *types.Heartbeat) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	sig, err := pv.signBytes(heartbeat.SignBytes(chainID))
	if err != nil {
		return err
	}
	heartbeat.Signature = sig
	return nil
}

// generate ed25519 keypair
func (pv *YubiHsmPV) genEd25519KeyPair() error {
	// create command to generate ed25519 keypair
	command, err := commands.CreateGenerateAsymmetricKeyCommand(0x00, []byte(YUBIHSM_SIGNKEY_LABEL),
		commands.Domain1, commands.CapabilityAsymmetricSignEddsa, commands.AlgorighmED25519)
	if err != nil {
		return err
	}

	// send command to YubiHSM
	resp, err := pv.sessionMgr.SendEncryptedCommand(command)
	if err != nil {
		return err
	}
	parsedResp, matched := resp.(*commands.CreateAsymmetricKeyResponse)
	if !matched {
		return errors.New("Invalid response for generating of ed25519 keypair")
	}

	// set sign key ID
	pv.SignKeyID = parsedResp.KeyID

	return nil
}

// export ed25519 public key
func (pv *YubiHsmPV) exportEd25519PubKey() error {
	var publicKey ed25519.PubKeyEd25519

	// create command to export ed25519 public key
	command, err := commands.CreateGetPubKeyCommand(pv.SignKeyID)
	if err != nil {
		return err
	}

	// send command to YubiHSM
	resp, err := pv.sessionMgr.SendEncryptedCommand(command)
	if err != nil {
		return err
	}
	parsedResp, matched := resp.(*commands.GetPubKeyResponse)
	if !matched {
		return errors.New("Invalid response for exporting ed25519 keypair")
	}

	if parsedResp.Algorithm != commands.AlgorighmED25519 {
		return errors.New("Wrong algorithm of public key")
	}

	// set public key data
	if len(parsedResp.KeyData) != ed25519.PubKeyEd25519Size {
		return errors.New("Invalid pubKey size")
	}

	// Convert raw key data to tendermint PubKey type
	copy(publicKey[:], parsedResp.KeyData[:])
	pv.PubKey = publicKey

	return nil
}

// sign bytes using ecdsa
func (pv *YubiHsmPV) signBytes(data []byte) ([]byte, error) {
	// send command to sign data
	command, err := commands.CreateSignDataEddsaCommand(pv.SignKeyID, data)
	if err != nil {
		return nil, err
	}
	resp, err := pv.sessionMgr.SendEncryptedCommand(command)
	if err != nil {
		return nil, err
	}

	// parse response
	parsedResp, matched := resp.(*commands.SignDataEddsaResponse)
	if !matched {
		return nil, errors.New("Invalid response type for sign command")
	}

	// TODO replace with ed25519.SignatureSize once tendermint is upgraded to >=v0.24.0
	if len(parsedResp.Signature) != 64 {
		return nil, errors.New("Invalid signature length")
	}

	return parsedResp.Signature, nil
}

// TODO: Remove this, it's only used in tests, and doesn't need access to internal fields so can
// be reimplemented as a standalone function.
// verify signature
func (pv *YubiHsmPV) verifySig(msg []byte, sig []byte) bool {
	pubKey := pv.PubKey.(ed25519.PubKeyEd25519)
	return pubKey.VerifyBytes(msg, sig)
}

// returns error if HRS regression or no LastSignBytes. returns true if HRS is unchanged
func (pv *YubiHsmPV) checkHRS(height int64, round int, step int8) (bool, error) {
	if pv.LastHeight > height {
		return false, errors.New("Height regression")
	}

	if pv.LastHeight == height {
		if pv.LastRound > round {
			return false, errors.New("Round regression")
		}

		if pv.LastRound == round {
			if pv.LastStep > step {
				return false, errors.New("Step regression")
			} else if pv.LastStep == step {
				if pv.LastSignBytes != nil {
					if pv.LastSignature == nil {
						panic("pv: LastSignature is nil but LastSignBytes is not!")
					}
					return true, nil
				}
				return false, errors.New("No LastSignature found")
			}
		}
	}
	return false, nil
}

// Persist height/round/step and signature
func (pv *YubiHsmPV) saveSigned(height int64, round int, step int8,
	signBytes []byte, sig []byte) {

	pv.LastHeight = height
	pv.LastRound = round
	pv.LastStep = step
	pv.LastSignature = sig
	pv.LastSignBytes = signBytes
	pv.save()
}

// signVote checks if the vote is good to sign and sets the vote signature.
// It may need to set the timestamp as well if the vote is otherwise the same as
// a previously signed vote (ie. we crashed after signing but before the vote hit the WAL).
func (pv *YubiHsmPV) signVote(chainID string, vote *types.Vote) error {
	height, round, step := vote.Height, vote.Round, voteToStep(vote)
	signBytes := vote.SignBytes(chainID)

	sameHRS, err := pv.checkHRS(height, round, step)
	if err != nil {
		return err
	}

	// We might crash before writing to the wal,
	// causing us to try to re-sign for the same HRS.
	// If signbytes are the same, use the last signature.
	// If they only differ by timestamp, use last timestamp and signature
	// Otherwise, return error
	if sameHRS {
		if bytes.Equal(signBytes, pv.LastSignBytes) {
			vote.Signature = pv.LastSignature
		} else if timestamp, ok := checkVotesOnlyDifferByTimestamp(pv.LastSignBytes, signBytes); ok {
			vote.Timestamp = timestamp
			vote.Signature = pv.LastSignature
		} else {
			err = fmt.Errorf("Conflicting data")
		}
		return err
	}

	// It passed the checks. Sign the vote
	sig, err := pv.signBytes(signBytes)
	if err != nil {
		return err
	}
	pv.saveSigned(height, round, step, signBytes, sig)
	vote.Signature = sig
	return nil
}

// signProposal checks if the proposal is good to sign and sets the proposal signature.
// It may need to set the timestamp as well if the proposal is otherwise the same as
// a previously signed proposal ie. we crashed after signing but before the proposal hit the WAL).
func (pv *YubiHsmPV) signProposal(chainID string, proposal *types.Proposal) error {
	height, round, step := proposal.Height, proposal.Round, stepPropose
	signBytes := proposal.SignBytes(chainID)

	sameHRS, err := pv.checkHRS(height, round, step)
	if err != nil {
		return err
	}

	// We might crash before writing to the wal,
	// causing us to try to re-sign for the same HRS.
	// If signbytes are the same, use the last signature.
	// If they only differ by timestamp, use last timestamp and signature
	// Otherwise, return error
	if sameHRS {
		if bytes.Equal(signBytes, pv.LastSignBytes) {
			proposal.Signature = pv.LastSignature
		} else if timestamp, ok := checkProposalsOnlyDifferByTimestamp(pv.LastSignBytes, signBytes); ok {
			proposal.Timestamp = timestamp
			proposal.Signature = pv.LastSignature
		} else {
			err = fmt.Errorf("Conflicting data")
		}
		return err
	}

	// It passed the checks. Sign the proposal
	sig, err := pv.signBytes(signBytes)
	if err != nil {
		return err
	}
	pv.saveSigned(height, round, step, signBytes, sig)
	proposal.Signature = sig
	return nil
}

// returns the timestamp from the lastSignBytes.
// returns true if the only difference in the votes is their timestamp.
func checkVotesOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) (time.Time, bool) {
	var lastVote, newVote types.CanonicalVote
	if err := cdc.UnmarshalJSON(lastSignBytes, &lastVote); err != nil {
		panic(fmt.Sprintf("LastSignBytes cannot be unmarshalled into vote: %v", err))
	}
	if err := cdc.UnmarshalJSON(newSignBytes, &newVote); err != nil {
		panic(fmt.Sprintf("signBytes cannot be unmarshalled into vote: %v", err))
	}

	lastTime := lastVote.Timestamp
	// set the times to the same value and check equality
	now := tmtime.Now()
	lastVote.Timestamp = now
	newVote.Timestamp = now
	lastVoteBytes, _ := cdc.MarshalJSON(lastVote)
	newVoteBytes, _ := cdc.MarshalJSON(newVote)

	return lastTime, bytes.Equal(newVoteBytes, lastVoteBytes)
}

// returns the timestamp from the lastSignBytes.
// returns true if the only difference in the proposals is their timestamp
func checkProposalsOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) (time.Time, bool) {
	var lastProposal, newProposal types.CanonicalProposal
	if err := cdc.UnmarshalJSON(lastSignBytes, &lastProposal); err != nil {
		panic(fmt.Sprintf("LastSignBytes cannot be unmarshalled into proposal: %v", err))
	}
	if err := cdc.UnmarshalJSON(newSignBytes, &newProposal); err != nil {
		panic(fmt.Sprintf("signBytes cannot be unmarshalled into proposal: %v", err))
	}

	lastTime := lastProposal.Timestamp
	// set the times to the same value and check equality
	now := tmtime.Now()
	lastProposal.Timestamp = now
	newProposal.Timestamp = now
	lastProposalBytes, _ := cdc.MarshalJSON(lastProposal)
	newProposalBytes, _ := cdc.MarshalJSON(newProposal)

	return lastTime, bytes.Equal(newProposalBytes, lastProposalBytes)
}
