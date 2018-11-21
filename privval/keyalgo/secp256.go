// +build secp256

//Loom has a single private key type that is defined at compile time, by changing a build tag

package keyalgo

import (
	"github.com/tendermint/tendermint/crypto/secp256k1"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	ABCIPubKeyType = tmtypes.ABCIPubKeyTypeSecp256k1
	PubKeySize     = secp256k1.PubKeySecp256k1Size
)

type PubKeyType = secp256k1.PubKeySecp256k1
