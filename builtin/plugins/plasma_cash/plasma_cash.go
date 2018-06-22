package plasma_cash

import (
	"errors"
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/common"
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
	GetProofRequest              = pctypes.GetProofRequest
	GetProofResponse             = pctypes.GetProofResponse
	PlasmaTx                     = pctypes.PlasmaTx
	GetCurrentBlockResponse      = pctypes.GetCurrentBlockResponse
	GetCurrentBlockRequest       = pctypes.GetCurrentBlockRequest
	PlasmaBookKeeping            = pctypes.PlasmaBookKeeping
	PlasmaBlock                  = pctypes.PlasmaBlock
	PendingSMT 					 = pctypes.PendingSMT
)

type PlasmaCash struct {
}

var (
	blockHeightKey = []byte("pcash_height")
	pendingSMTKey = []byte("pcash_pending_smt")
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

func (c *PlasmaCash) SubmitBlockToMainnet(ctx contract.Context, req *SubmitBlockToMainnetRequest) error {
	//this will largely happen in an external oracle
	
	//TODO lets make sure we don't allow it to happen twice

	//if we have a half open block we should flush it
	//Raise blockheight
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)
	pbk.CurrentHeight.Value = *pbk.CurrentHeight.Value.Add(loom.NewBigUIntFromInt(1), &pbk.CurrentHeight.Value)
	ctx.Set(blockHeightKey, pbk)

	pb := &PlasmaBlock{
		Proof: []byte("123"),
	}
	ctx.Set(blockKey(pbk.CurrentHeight.Value), pb)

	return nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
	var err error
	var smt *mamamerkle.SparseMerkleTree
	pendingSMT := &PendingSMT{}
	ctx.Get(pendingSMTKey, pendingSMT)
	data := pendingSMT.GetData()

	if len(data) == 0 {
		//Lets see if we have an open sparse merkle tree, if not lets start a new one
		var leaves = make(map[int64][]byte)
		smt, err = mamamerkle.NewSparseMerkleTree(64, leaves)
		fmt.Printf("weeee-%v\n", smt)
	} else {
		smt, err = mamamerkle.LoadSparseMerkleTree(data)
	}
	if err != nil {
		return err
	}


	//Lets serialize and store it
	res,err := smt.Serialize() //right now we store as a byte array

	if err != nil {
		return err
	}

	resPendingSMT := &PendingSMT{}
	resPendingSMT.Data = res
	resPendingSMT.Version = 1 //track the internal format of the smt

	ctx.Set(pendingSMTKey, resPendingSMT)

	return err
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
	return &GetBlockResponse{Block: &PlasmaBlock{
		Proof: []byte("123"),
	}}, nil
}

func (c *PlasmaCash) GetProofRequest(ctx contract.StaticContext, req *GetProofRequest) (*GetProofResponse, error) {
	return &GetProofResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
