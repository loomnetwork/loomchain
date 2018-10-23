package dposv2

import (
	"encoding/hex"
//	"testing"

	loom "github.com/loomnetwork/go-loom"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
)

func mustAddrFromPubKey(s string) loom.Address {
	pubKey, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
}
//TODO make a delegation election unit test
