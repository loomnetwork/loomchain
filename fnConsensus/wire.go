package fnConsensus

import (
	amino "github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
)

var cdc = amino.NewCodec()

func init() {
	RegisterAmino(cdc)
}

func RegisterAmino(cdc *amino.Codec) {
	cryptoAmino.RegisterAmino(cdc)
	RegisterFnConsensusTypes()
}
