package loomchain

import (
	"bytes"
	"encoding/binary"

	"github.com/loomnetwork/loomchain/store"
)

type Sequence struct {
	Key []byte
}

func NewSequence(key []byte) *Sequence {
	return &Sequence{Key: key}
}

func (s *Sequence) Value(state ReadOnlyState) uint64 {
	var seq uint64
	data := state.Get(s.Key)
	if len(data) > 0 {
		err := binary.Read(bytes.NewReader(data), binary.BigEndian, &seq)
		if err != nil {
			panic(err)
		}
	}

	return seq
}

func (s *Sequence) Next(state State, kvstore store.KVStore, isCheckTx bool) uint64 {
	seq := s.Value(state) + 1

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, seq)
	if err != nil {
		panic(err)
	}

	state.Set(s.Key, buf.Bytes())

	if state.FeatureEnabled(IncrementNonceFailedTxFeature, false) && !isCheckTx {
		kvstore.Set(s.Key, buf.Bytes())
	}

	return seq
}
