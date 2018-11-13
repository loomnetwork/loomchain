package hsmpv

import (
	"errors"
)

// YubiHsmSigner implements the Signer interface using YubiHSM.
type YubiHsmSigner struct {
	pv *YubiHsmPV
}

// NewYubiHsmSigner creates new instance of YubiHsmSigner
func NewYubiHsmSigner(pv *YubiHsmPV) *YubiHsmSigner {
	return &YubiHsmSigner{pv}
}

// Sign message
func (s *YubiHsmSigner) Sign(msg []byte) []byte {
	if s.pv.SignKeyID == 0 {
		panic(errors.New("SignKeyID isn't set"))
	}

	signBytes, err := s.pv.signBytes(msg)
	if err != nil {
		panic(err)
	}
	return signBytes.Bytes()
}

// PublicKey gets public key as byte array
func (s *YubiHsmSigner) PublicKey() []byte {
	return s.pv.PubKey.Bytes()
}
