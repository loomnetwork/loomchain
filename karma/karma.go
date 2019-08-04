//nolint
package karma

import (
	"encoding/binary"

	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
)

var (
	// TODO: eliminate
	lastKarmaUpkeepKey = []byte("last:upkeep:karma")
)

func NewKarmaHandler(karmaContractCtx contractpb.Context) loomchain.KarmaHandler {
	return &karmaHandler{
		karmaContractCtx: karmaContractCtx,
	}
}

type karmaHandler struct {
	karmaContractCtx contractpb.Context
}

func (kh *karmaHandler) Upkeep() error {
	return karma.Upkeep(kh.karmaContractCtx)
}

func UintToBytesBigEndian(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.BigEndian.PutUint64(heightB, height)
	return heightB
}
