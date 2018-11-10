
package hsmpv

import (
	"errors"
	"sync"
	"io/ioutil"

	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/types"

	cmn "github.com/tendermint/tendermint/libs/common"

	"github.com/certusone/yubihsm-go"
	"github.com/certusone/yubihsm-go/commands"
	"github.com/certusone/yubihsm-go/connector"
)

// fixme: these constants should be set by configuration
const (
	YUBIHSM_CONN_URL      = "localhost:1234"
	YUBIHSM_AUTH_KEYID    = 1
	YUBIHSM_PASSWORD      = "loomchain"

	YUBIHSM_SIGNKEY_LABEL = "loomchain-hsm-pv"
)

// YubiHSM structure
type YubiHsmPV struct {
	sessionMgr      *yubihsm.SessionManager

	hsmURL          string
	authKeyID       uint16
	password        string

	LastHeight      int64             `json:"last_height"`
	LastRound       int               `json:"last_round"`
	LastStep        int8              `json:"last_step"`

	Address         types.Address     `json:"address"`
	SignKeyID       uint16            `json:"key_id"`

	PubKey          crypto.PubKey     `json:"pub_key"`

	filePath        string
	mtx             sync.Mutex
}

// create a new instance of YubiHSM priv validator
func NewYubiHsmPV(connURL string, authKeyID uint16, password string) *YubiHsmPV {
	return &YubiHsmPV {
		hsmURL:      connURL,
		authKeyID:   authKeyID,
		password:    password,
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
	err = pv.genEd25519KeyPair()
	if err != nil {
		pv.Destroy()
		return err
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

	pv.filePath = filePath
	return nil
}

// initialize YubiHsm priv validator
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

	signBytes, err := pv.signBytes(vote.SignBytes(chainID))
	if err != nil {
		return err
	}

	vote.Signature = ed25519.SignatureEd25519FromBytes(signBytes)
	return nil
}

// sign proposal
func (pv *YubiHsmPV) SignProposal(chainID string, proposal *types.Proposal) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	signBytes, err := pv.signBytes(proposal.SignBytes(chainID))
	if err != nil {
		return err
	}

	proposal.Signature = ed25519.SignatureEd25519FromBytes(signBytes)
	return nil
}

// sign heartbeat
func (pv *YubiHsmPV) SignHeartbeat(chainID string, heartbeat *types.Heartbeat) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	signBytes, err := pv.signBytes(heartbeat.SignBytes(chainID))
	if err != nil {
		return err
	}

	heartbeat.Signature = ed25519.SignatureEd25519FromBytes(signBytes)
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
	publicKey := new(ed25519.PubKeyEd25519)
	copy(publicKey[:], parsedResp.KeyData[:])

	// Cache publicKey
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
