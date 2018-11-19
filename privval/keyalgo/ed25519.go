// +build !secp256

//Loom has a single private key type that is defined at compile time, by changing a build tag

package keyalgo

import (
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	ABCIPubKeyType = tmtypes.ABCIPubKeyTypeEd25519
	PubKeySize     = ed25519.PubKeyEd25519Size
)

type PubKeyType = ed25519.PubKeyEd25519
