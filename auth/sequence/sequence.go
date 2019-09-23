package sequence

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

func (s *Sequence) Value(state store.KVReader) uint64 {
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

func (s *Sequence) Next(state store.KVStore) uint64 {
	seq := s.Value(state) + 1

	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, seq)
	if err != nil {
		panic(err)
	}

	state.Set(s.Key, buf.Bytes())
	return seq
}
