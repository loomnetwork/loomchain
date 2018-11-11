
package hsmpv

import (
	"errors"
)

// YubiHsmSigner implements the Signer interface using YubiHSM.
type YubiHsmSigner struct {
	pv  *YubiHsmPV
}

func NewYubiHsmSigner(pv *YubiHsmPV) *YubiHsmSigner {
	return &YubiHsmSigner{pv}
}

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

func (s *YubiHsmSigner) PublicKey() []byte {
	return s.pv.PubKey.Bytes()
}
