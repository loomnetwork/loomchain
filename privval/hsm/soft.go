
package hsmpv

import (
	"fmt"
	"errors"
	"sync"
	"time"
	"bytes"

	"io/ioutil"

	"crypto/ecdsa"
	"crypto/elliptic"

	"encoding/base64"
	"encoding/asn1"

	"math/rand"
	"math/big"

	"github.com/miekg/pkcs11"

	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/btcd/btcec"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/types"
)

// Information about an Elliptic Curve
type curveInfo struct {
	// ASN.1 marshaled OID
	oid []byte

	// Curve definition in Go form
	curve elliptic.Curve
}

// error codes
var ErrKeyNotFound = errors.New("Failed to find the key with specified ID")
var ErrUnsupportedEllipticCurve = errors.New("Unsupported elliptic curve")
var ErrMalformedPoint = errors.New("Malformed elliptic curve point")
var ErrMalformedDER = errors.New("Malformed DER format")

// HSM priv validator
type SoftHsmPV struct {
	p11LibPath      string

	p11Ctx          *pkcs11.Ctx
	p11Session      pkcs11.SessionHandle

	bSessionExist   bool
	bLogined        bool

	userID          string
	userPin         string

	Address         types.Address  `json:"address"`

	LastHeight      int64          `json:"last_height"`
	LastRound       int            `json:"last_round"`
	LastStep        int8           `json:"last_step"`

	SignKeyID       string         `json:"key_id"`

	PubKeyBytes     crypto.PubKey  `json:"pub_key"`
	privKeyHandle   pkcs11.ObjectHandle

	filePath        string
	mtx             sync.Mutex
}

// create new Soft HSM priv validator
func NewSoftHsmPV(p11LibPath string, userPin string) *SoftHsmPV {
	return &SoftHsmPV {
		p11LibPath: p11LibPath,
		userPin:    userPin,
	};
}

// init HSM priv validator
func (pv *SoftHsmPV) Init() error {
	var p11Slots []uint
	var err error

	// initialize pkcs11 library
	pv.p11Ctx = pkcs11.New(pv.p11LibPath)
	if pv.p11Ctx == nil {
		return fmt.Errorf("Invalid PKCS#11 library '%s'", pv.p11LibPath)
	}

	if err = pv.p11Ctx.Initialize(); err != nil {
		return err
	}

	// get slot list
	if p11Slots, err = pv.p11Ctx.GetSlotList(true); err != nil {
		pv.Destroy()
		return err
	}

	// open session with HSM
	pv.p11Session, err = pv.p11Ctx.OpenSession(p11Slots[0], pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		pv.Destroy()
		return err
	}
	pv.bSessionExist = true

	// login to token
	err = pv.p11Ctx.Login(pv.p11Session, pkcs11.CKU_USER, pv.userPin)
	if err != nil {
		pv.Destroy()
		return err
	}
	pv.bLogined = true

	return nil
}

// destroy HSM priv validator
func (pv *SoftHsmPV) Destroy() {
	// logout session
	if pv.bLogined {
		pv.p11Ctx.Logout(pv.p11Session)
	}

	// close session
	if pv.bSessionExist {
		pv.p11Ctx.CloseSession(pv.p11Session)
	}

	// destroy and finalize pcks11 context
	pv.p11Ctx.Finalize()
	pv.p11Ctx.Destroy()
}

// generate new HsmPV with generated keypair
func (pv *SoftHsmPV) GenPrivVal(filepath string) error {
	var pubKeyHandle pkcs11.ObjectHandle
	var privKeyHandle pkcs11.ObjectHandle

	var pubKeyData *btcec.PublicKey
	var pubKeyBytes secp256k1.PubKeySecp256k1

	var err error

	// init HsmPV
	err = pv.Init()
	if err != nil {
		return err
	}

	// generate keypair
	if pubKeyHandle, privKeyHandle, err = pv.genEd25519KeyPair(); err != nil {
		pv.Destroy()
		return err
	}
	pv.filePath = filepath

	// export public and private key
	pubKeyData, err = pv.exportECDSAPublicKey(pubKeyHandle)
	if err != nil {
		return err
	}

	// set public and private key to validator
	copy(pubKeyBytes[:], pubKeyData.SerializeCompressed())

	pv.PubKeyBytes = pubKeyBytes
	pv.privKeyHandle = privKeyHandle

	return nil
}

// load HsmPV from file
func (pv *SoftHsmPV) LoadPrivVal(filePath string) error {
	pvJSONBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		cmn.Exit(err.Error())
	}

	err = cdc.UnmarshalJSON(pvJSONBytes, &pv)
	if err != nil {
		return err
	}

	// init HsmPV
	if err := pv.Init(); err != nil {
		return err
	}

	pv.filePath = filePath
	return nil
}

// reset parameters
func (pv *SoftHsmPV) Reset(height int64) {
	pv.LastHeight = height
	pv.LastRound = 0
	pv.LastStep = 0
}

// save persists the HsmPV to disk
func (pv *SoftHsmPV) Save() {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	pv.save()
}

func (pv *SoftHsmPV) save() {
	outFile := pv.filePath
	if outFile == "" {
		panic("Cannot save PrivValidator: filePath not set")
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
func (pv *SoftHsmPV) GetPubKey() crypto.PubKey {
	return pv.PubKeyBytes
}

// get address
func (pv *SoftHsmPV) GetAddress() types.Address {
	return pv.PubKeyBytes.Address()
}

// sign vote
func (pv *SoftHsmPV) SignVote(chainID string, vote *types.Vote) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	signBytes, err := pv.signBytes(vote.SignBytes(chainID))
	if err != nil {
		return err
	}

	vote.Signature = secp256k1.SignatureSecp256k1FromBytes(signBytes)
	return nil
}

// sign proposal
func (pv *SoftHsmPV) SignProposal(chainID string, proposal *types.Proposal) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	signBytes, err := pv.signBytes(proposal.SignBytes(chainID))
	if err != nil {
		return err
	}

	proposal.Signature = secp256k1.SignatureSecp256k1FromBytes(signBytes)
	return nil
}

// sign heartbeat
func (pv *SoftHsmPV) SignHeartbeat(chainID string, heartbeat *types.Heartbeat) error {
	pv.mtx.Lock()
	defer pv.mtx.Unlock()

	signBytes, err := pv.signBytes(heartbeat.SignBytes(chainID))
	if err != nil {
		return err
	}

	heartbeat.Signature = secp256k1.SignatureSecp256k1FromBytes(signBytes)
	return nil
}

// generate random key ID
func genKeyID() (string, error) {
	const idSize = 32

	// generate random seed
	rand.Seed(time.Now().UnixNano())

	// generate random bytes
	rawData := make([]byte, idSize)
	_, err := rand.Read(rawData)
	if err != nil {
		return "", err
	}

	// encoding by base64
	return base64.StdEncoding.EncodeToString(rawData), nil
}

// generate keypair
func (pv *SoftHsmPV) genEd25519KeyPair() (pkcs11.ObjectHandle, pkcs11.ObjectHandle, error) {
	var parameters []byte

	// generate random key ID
	keyID, err := genKeyID()
	if err != nil {
		return 0, 0, err
	}
	pv.SignKeyID = keyID

	// set parameter
	if parameters, err = asn1.Marshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}); err != nil {
		return 0, 0, err
	}

	// set attributes for private and public key
	pubKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),
		pkcs11.NewAttribute(pkcs11.CKA_ID, keyID),
		pkcs11.NewAttribute(pkcs11.CKA_ECDSA_PARAMS, parameters),
	}
	privKeyTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
		pkcs11.NewAttribute(pkcs11.CKA_ID, keyID),
	}

	// generate keypair
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA_KEY_PAIR_GEN, nil)}
	pubHandle, privHandle, err := pv.p11Ctx.GenerateKeyPair(pv.p11Session,
		mech,
		pubKeyTemplate,
		privKeyTemplate)
	if err != nil {
		return 0, 0, err
	}

	return pubHandle, privHandle, nil
}

// Export the public key corresponding to a private ECDSA key.
func (pv *SoftHsmPV) exportECDSAPublicKey(pubHandle pkcs11.ObjectHandle) (*btcec.PublicKey, error) {
	var err error
	var attributes []*pkcs11.Attribute
	var pub ecdsa.PublicKey

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_ECDSA_PARAMS, nil),
		pkcs11.NewAttribute(pkcs11.CKA_EC_POINT, nil),
	}

	if attributes, err = pv.p11Ctx.GetAttributeValue(pv.p11Session, pubHandle, template); err != nil {
		return nil, err
	}

	if pub.Curve, err = unmarshalEcParams(attributes[0].Value); err != nil {
		return nil, err
	}

	if pub.X, pub.Y, err = unmarshalEcPoint(attributes[1].Value, pub.Curve); err != nil {
		return nil, err
	}

	return (*btcec.PublicKey)(&pub), nil
}

// ASN.1 marshal some value and panic on error
func mustMarshal(val interface{}) []byte {
	if b, err := asn1.Marshal(val); err != nil {
		panic(err)
	} else {
		return b
	}
}

// well know curve infos
var wellKnownCurves = map[string]curveInfo{
	"P-192": {
		mustMarshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 1}),
		nil,
	},
	"P-224": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 33}),
		elliptic.P224(),
	},
	"P-256": {
		mustMarshal(asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}),
		elliptic.P256(),
	},
	"P-384": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 34}),
		elliptic.P384(),
	},
	"P-521": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 35}),
		elliptic.P521(),
	},

	"K-163": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 1}),
		nil,
	},
	"K-233": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 26}),
		nil,
	},
	"K-283": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 16}),
		nil,
	},
	"K-409": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 36}),
		nil,
	},
	"K-571": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 38}),
		nil,
	},

	"B-163": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 15}),
		nil,
	},
	"B-233": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 27}),
		nil,
	},
	"B-283": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 17}),
		nil,
	},
	"B-409": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 37}),
		nil,
	},
	"B-571": {
		mustMarshal(asn1.ObjectIdentifier{1, 3, 132, 0, 39}),
		nil,
	},
}

func marshalEcParams(c elliptic.Curve) ([]byte, error) {
	if ci, ok := wellKnownCurves[c.Params().Name]; ok {
		return ci.oid, nil
	}
	// TODO use ANSI X9.62 ECParameters representation instead
	return nil, ErrUnsupportedEllipticCurve
}

func unmarshalEcParams(b []byte) (elliptic.Curve, error) {
	// See if it's a well-known curve
	for _, ci := range wellKnownCurves {
		if bytes.Compare(b, ci.oid) == 0 {
			if ci.curve != nil {
				return ci.curve, nil
			}
			return nil, ErrUnsupportedEllipticCurve
		}
	}
	// TODO try ANSI X9.62 ECParameters representation
	return nil, ErrUnsupportedEllipticCurve
}

func unmarshalEcPoint(b []byte, c elliptic.Curve) (x *big.Int, y *big.Int, err error) {
	// Decoding an octet string in isolation seems to be too hard
	// with encoding.asn1, so we do it manually. Look away now.
	if b[0] != 4 {
		return nil, nil, ErrMalformedDER
	}
	var l, r int
	if b[1] < 128 {
		l = int(b[1])
		r = 2
	} else {
		ll := int(b[1] & 127)
		if ll > 2 { // unreasonably long
			return nil, nil, ErrMalformedDER
		}
		l = 0
		for i := int(0); i < ll; i++ {
			l = 256*l + int(b[2+i])
		}
		r = ll + 2
	}
	if r+l > len(b) {
		return nil, nil, ErrMalformedDER
	}
	pointBytes := b[r:]
	x, y = elliptic.Unmarshal(c, pointBytes)
	if x == nil || y == nil {
		err = ErrMalformedPoint
	}
	return
}

// get handle of private or public key
func (pv *SoftHsmPV) getECKeyHandle(isPubKey bool) (pkcs11.ObjectHandle, error) {
	var keyClass uint
	var template []*pkcs11.Attribute

	// set key class
	if isPubKey {
		keyClass = pkcs11.CKO_PUBLIC_KEY
	} else {
		keyClass = pkcs11.CKO_PRIVATE_KEY
	}

	// build attributes template
	template = append(template, pkcs11.NewAttribute(pkcs11.CKA_CLASS, keyClass))
	template = append(template, pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_ECDSA))
	template = append(template, pkcs11.NewAttribute(pkcs11.CKA_ID, pv.SignKeyID))

	// find object
	if err := pv.p11Ctx.FindObjectsInit(pv.p11Session, template); err != nil {
		return 0, err
	}
	defer pv.p11Ctx.FindObjectsFinal(pv.p11Session)

	keyHandles, _, err := pv.p11Ctx.FindObjects(pv.p11Session, 1)
	if err != nil {
		return 0, err
	}

	if len(keyHandles) == 0 {
		return 0, ErrKeyNotFound
	}

	return keyHandles[0], nil
}


// sign bytes using ecdsa
func (pv *SoftHsmPV) signBytes(data []byte) ([]byte, error) {
	var err error
	var sigBytes []byte

	// check if private object handle was cached
	if (pv.privKeyHandle == 0) {
		pv.privKeyHandle, err = pv.getECKeyHandle(false)
		if err != nil {
			return nil, err
		}
	}

	// sign digest
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}
	if err = pv.p11Ctx.SignInit(pv.p11Session, mech, pv.privKeyHandle); err != nil {
		return nil, err
	}

	sigBytes, err = pv.p11Ctx.Sign(pv.p11Session, crypto.Sha256(data))
	if err != nil {
		return nil, err
	}

	return sigBytes, nil
}

// verify signature
func (pv *SoftHsmPV) verifySign(plainData []byte, signature []byte) error {
	var pubKeyHandle pkcs11.ObjectHandle
	var err error

	// get public key handle
	pubKeyHandle, err = pv.getECKeyHandle(true)
	if err != nil {
		return err
	}

	// init verification
	mech := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}
	if err = pv.p11Ctx.VerifyInit(pv.p11Session, mech, pubKeyHandle); err != nil {
		return err
	}

	// verify
	if err = pv.p11Ctx.Verify(pv.p11Session, crypto.Sha256(plainData), signature); err != nil {
		return err
	}

	return nil
}
