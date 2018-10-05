package bloom

import (
	`github.com/loomnetwork/go-loom/plugin/types`
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	BitsPerKey = 8
)

func NewBloomFilter() filter.Filter {
	return filter.NewBloomFilter(BitsPerKey)
}

func GenBloomFilter(msgs []*types.EventData) []byte {
	if len(msgs) == 0 {
		return []byte{}
	} else {
		bloomFilter := filter.NewBloomFilter(BitsPerKey)
		generator := bloomFilter.NewGenerator()

		for _, msg := range msgs {
			for _, topic := range msg.Topics {
				generator.Add([]byte(topic))
			}
			generator.Add(msg.Address.Local)
		}
		buff := &util.Buffer{}
		generator.Generate(buff)
		return buff.Bytes()
	}
}
