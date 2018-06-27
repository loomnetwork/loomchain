package plasma_cash

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	loom "github.com/loomnetwork/go-loom"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/mamamerkle"
)

var (
	decimals                  int64 = 18
	errCandidateNotRegistered       = errors.New("candidate is not registered")
)

type (
	InitRequest                  = pctypes.PlasmaCashInitRequest
	SubmitBlockToMainnetRequest  = pctypes.SubmitBlockToMainnetRequest
	SubmitBlockToMainnetResponse = pctypes.SubmitBlockToMainnetResponse
	GetBlockRequest              = pctypes.GetBlockRequest
	GetBlockResponse             = pctypes.GetBlockResponse
	PlasmaTxRequest              = pctypes.PlasmaTxRequest
	PlasmaTxResponse             = pctypes.PlasmaTxResponse
	DepositRequest               = pctypes.DepositRequest
	PlasmaTx                     = pctypes.PlasmaTx
	GetCurrentBlockResponse      = pctypes.GetCurrentBlockResponse
	GetCurrentBlockRequest       = pctypes.GetCurrentBlockRequest
	PlasmaBookKeeping            = pctypes.PlasmaBookKeeping
	PlasmaBlock                  = pctypes.PlasmaBlock
	Pending                      = pctypes.Pending
)

type PlasmaCash struct {
}

var (
	blockHeightKey    = []byte("pcash_height")
	pendingTXsKey     = []byte("pcash_pending")
	plasmaMerkleTopic = "pcash_mainnet_merkle"
)

func blockKey(height common.BigUInt) []byte {
	return util.PrefixKey([]byte("pcash_block_"), []byte(height.String()))
}

func (c *PlasmaCash) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "plasmacash",
		Version: "1.0.0",
	}, nil
}

func (c *PlasmaCash) Init(ctx contract.Context, req *InitRequest) error {
	//params := req.Params
	ctx.Set(blockHeightKey, &PlasmaBookKeeping{CurrentHeight: &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(0),
	}})

	return nil
}

func round(num, near int64) int64 {
	if num == 0 {
		return near
	}
	if near == num {
		return num + near
	}
	return ((num + (near - 1)) / near) * near
}

func (c *PlasmaCash) SubmitBlockToMainnet(ctx contract.Context, req *SubmitBlockToMainnetRequest) (*SubmitBlockToMainnetResponse, error) {
	//if we have a half open block we should flush it
	//Raise blockheight
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	//TODO do this rounding in a bigint safe way
	// round to nearest 1000
	roundedInt := round(pbk.CurrentHeight.Value.Int64(), 1000)
	pbk.CurrentHeight.Value = *loom.NewBigUIntFromInt(roundedInt)
	ctx.Set(blockHeightKey, pbk)

	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	leaves := make(map[int64][]byte)

	if len(pending.Transactions) == 0 {
		//TODO maybe allow empty blocks after so many minutes?
		return nil, fmt.Errorf("No transactions in block. Refusing to create block")
	}

	for _, v := range pending.Transactions {

		if v.PreviousBlock == nil || v.PreviousBlock.Value.Int64() == int64(0) {
			hash, err := soliditySha3(v.Slot)
			if err != nil {
				return nil, err
			}
			v.MerkleHash = hash
		} else {
			hash, err := rlpEncodeWithSha3(v)
			if err != nil {
				return nil, err
			}
			v.MerkleHash = hash
		}

		leaves[int64(v.Slot)] = v.MerkleHash //TODO change mamamerkle to use uint64
	}

	smt, err := mamamerkle.NewSparseMerkleTree(64, leaves)
	if err != nil {
		return nil, err
	}

	for _, v := range pending.Transactions {
		v.Proof = smt.CreateMerkleProof(int64(v.Slot))
	}

	merkleHash := smt.Root()

	pb := &PlasmaBlock{
		MerkleHash:   merkleHash,
		Transactions: pending.Transactions,
	}
	ctx.Set(blockKey(pbk.CurrentHeight.Value), pb)

	ctx.EmitTopics(merkleHash, plasmaMerkleTopic)

	return &SubmitBlockToMainnetResponse{MerkleHash: merkleHash}, nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
	fmt.Printf("PlasmaTxRequest\n")
	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	pending.Transactions = append(pending.Transactions, req.Plasmatx)

	return ctx.Set(pendingTXsKey, pending)
}

func (c *PlasmaCash) DepositRequest(ctx contract.Context, req *DepositRequest) error {
	return nil
}

func (c *PlasmaCash) GetCurrentBlockRequest(ctx contract.StaticContext, req *GetCurrentBlockRequest) (*GetCurrentBlockResponse, error) {
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)
	return &GetCurrentBlockResponse{pbk.CurrentHeight}, nil
}

func (c *PlasmaCash) GetBlockRequest(ctx contract.StaticContext, req *GetBlockRequest) (*GetBlockResponse, error) {
	pb := &PlasmaBlock{}

	err := ctx.Get(blockKey(req.BlockHeight.Value), pb)
	if err != nil {
		return nil, err
	}

	return &GetBlockResponse{Block: pb}, nil
}

func soliditySha3(data uint64) ([]byte, error) {
	pairs := []*evmcompat.Pair{&evmcompat.Pair{"uint64", strconv.FormatUint(data, 10)}}
	hash, err := evmcompat.SoliditySHA3(pairs)
	if err != nil {
		return []byte{}, err
	}
	return hash, err
}

func rlpEncodeWithSha3(pb *PlasmaTx) ([]byte, error) {
	hash, err := rlpEncode(pb)
	if err != nil {
		return []byte{}, err
	}
	d := sha3.NewKeccak256()
	d.Write(hash)
	return d.Sum(nil), nil
}

func rlpEncode(pb *PlasmaTx) ([]byte, error) {
	return rlp.EncodeToBytes([]interface{}{
		uint64(pb.Slot),
		pb.PreviousBlock.Value.Bytes(),
		uint32(pb.Denomination.Value.Int64()),
		pb.GetNewOwner().Local,
	})
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
