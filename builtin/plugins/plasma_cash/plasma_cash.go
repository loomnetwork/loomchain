package plasma_cash

import (
	"errors"
	"fmt"

	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
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
)

type PlasmaCash struct {
}

func (c *PlasmaCash) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "plasmacash",
		Version: "1.0.0",
	}, nil
}

func (c *PlasmaCash) Init(ctx contract.Context, req *InitRequest) error {
	//params := req.Params
	return nil
}

func (c *PlasmaCash) SubmitBlockToMainnet(ctx contract.Context, req *SubmitBlockToMainnetRequest) error {
	return nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {

	//TODO we are going to close a block on each TX for now
	//then later we are going to need to make the cron interface do it
	var leaves = make(map[int64][]byte)
	smt, err := mamamerkle.NewSparseMerkleTree(3, leaves)
	fmt.Printf("weeee-%v\n", smt)

	return err
}

func (c *PlasmaCash) DepositRequest(ctx contract.Context, req *DepositRequest) error {
	return nil
}

func (c *PlasmaCash) GetBlockRequest(ctx contract.StaticContext, req *GetBlockRequest) (*GetBlockResponse, error) {
	return &GetBlockResponse{}, nil
}

func (c *PlasmaCash) GetProofRequest(ctx contract.StaticContext, req *GetProofRequest) (*GetProofResponse, error) {
	return &GetProofResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
