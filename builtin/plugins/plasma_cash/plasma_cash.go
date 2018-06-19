package plasma_cash

import (
	"errors"

	loom "github.com/loomnetwork/go-loom"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
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
)

type PlasmaCash struct {
}

var (
	blockHeightKey = []byte("pcash_height")
)

func accountKey(addr loom.Address) []byte {
	return util.PrefixKey([]byte("account"), addr.Bytes())
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
	return nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
	var err error

	//TODO we are going to close a block on each TX for now
	//then later we are going to need to make the cron interface do it
	//var leaves = make(map[int64][]byte)
	//	smt, err := mamamerkle.NewSparseMerkleTree(3, leaves)
	//	fmt.Printf("weeee-%v\n", smt)

	//Raise blockheight
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)
	pbk.CurrentHeight.Value = *pbk.CurrentHeight.Value.Add(loom.NewBigUIntFromInt(1), &pbk.CurrentHeight.Value)
	ctx.Set(blockHeightKey, pbk)

	return err
}

func (c *PlasmaCash) DepositRequest(ctx contract.Context, req *DepositRequest) error {
	return nil
}

func (c *PlasmaCash) GetCurrentBlockRequest(ctx contract.StaticContext, req *GetCurrentBlockRequest) (*GetCurrentBlockResponse, error) {
	return &GetCurrentBlockResponse{}, nil
}

func (c *PlasmaCash) GetBlockRequest(ctx contract.StaticContext, req *GetBlockRequest) (*GetBlockResponse, error) {
	return &GetBlockResponse{}, nil
}

func (c *PlasmaCash) GetProofRequest(ctx contract.StaticContext, req *GetProofRequest) (*GetProofResponse, error) {
	return &GetProofResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
