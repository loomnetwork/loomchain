package plasma_cash

import (
	"errors"

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
	Pending                      = pctypes.Pending
)

type PlasmaCash struct {
}

var (
	blockHeightKey = []byte("pcash_height")
	pendingTXsKey  = []byte("pcash_pending")
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

	var leaves map[int64][]byte
	smt, err := mamamerkle.NewSparseMerkleTree(64, leaves)
	if err != nil {
		return err
	}
	pb := &PlasmaBlock{
		Proof:      []byte("123"),
		MerkleHash: smt.CreateMerkleProof(0), //TODO root?
	}
	ctx.Set(blockKey(pbk.CurrentHeight.Value), pb)

	//TODO EMIT merkle hash
	//        merkle_hash = w3.toHex(block.merklize_transaction_set())
	//hashed_transaction_dict = {tx.uid: tx.hash
	//	for tx in self.transaction_set}
	//self.merkle = SparseMerkleTree(64, hashed_transaction_dict)
	//return self.merkle.root

	//    @property
	//def merkle_hash(self):
	//return w3.sha3(rlp.encode(self))

	return nil
}

func (c *PlasmaCash) PlasmaTxRequest(ctx contract.Context, req *PlasmaTxRequest) error {
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
	return &GetBlockResponse{Block: &PlasmaBlock{
		Proof: []byte("123"),
	}}, nil
}

func (c *PlasmaCash) GetProofRequest(ctx contract.StaticContext, req *GetProofRequest) (*GetProofResponse, error) {
	return &GetProofResponse{}, nil
}

var Contract plugin.Contract = contract.MakePluginContract(&PlasmaCash{})
