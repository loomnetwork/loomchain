package plasma_cash

import (
	"errors"
	"fmt"
	"strconv"

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

func (c *PlasmaCash) SubmitBlockToMainnet(ctx contract.Context, req *SubmitBlockToMainnetRequest) error {
	//this will largely happen in an external oracle

	//TODO lets make sure we don't allow it to happen twice

	//if we have a half open block we should flush it
	//Raise blockheight
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	//TODO do this rounding in a bigint safe way
	// round to nearest 1000
	roundedInt := round(pbk.CurrentHeight.Value.Int64(), 1000)
	//	pbk.CurrentHeight.Value = *pbk.CurrentHeight.Value.Add(loom.NewBigUIntFromInt(1), )
	pbk.CurrentHeight.Value = *loom.NewBigUIntFromInt(roundedInt)
	ctx.Set(blockHeightKey, pbk)

	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	leaves := make(map[int64][]byte)

	for _, v := range pending.Transactions {
		//TODO how does the MerkleHASH get set?
		leaves[int64(v.Slot)] = v.MerkleHash //TODO change mamamerkle to use uint64
	}

	smt, err := mamamerkle.NewSparseMerkleTree(64, leaves)
	if err != nil {
		return err
	}
	fmt.Printf("weee smt-%v\n", smt)

	//TODO convert to web3 hex
	//w3.toHex
	merkleHash := smt.CreateMerkleProof(int64(0))
	pb := &PlasmaBlock{
		MerkleHash: merkleHash,
	}
	ctx.Set(blockKey(pbk.CurrentHeight.Value), pb)

	ctx.EmitTopics(merkleHash, plasmaMerkleTopic)

	return nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)

	blockHeight := 0
	if blockHeight%1000 == 0 {
		//TODO handle plasma blocks
		//ret = w3.sha3(rlp.encode(self, UnsignedTransaction))
	} else {
		// ret = w3.soliditySha3(['uint64'], [self.uid])
		///req.Plasmatx.MerkleHash = soliditySha3("uint64", req.Plasmatx.Slot)
		pairs := []*evmcompat.Pair{&evmcompat.Pair{"uint64", strconv.FormatUint(req.Plasmatx.Slot, 10)}}
		err, hash := evmcompat.SoliditySHA3(pairs)
		if err != nil {
			return err
		}
		req.Plasmatx.MerkleHash = hash
	}

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

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
