// +build evm

package oracle

import (
	"math/big"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/go-loom/client"
	ltypes "github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
)

type DAppChainPlasmaClientConfig struct {
	ChainID  string
	WriteURI string
	ReadURI  string
	// Used to sign txs sent to Loom Protocol
	Signer auth.Signer
	// name of plasma cash contract on Protocol
	ContractName string
}

type DAppChainPlasmaClient interface {
	Init() error
	CurrentPlasmaBlockNum() (*big.Int, error)
	PlasmaBlockAt(blockNum *big.Int) (*pctypes.PlasmaBlock, error)
	FinalizeCurrentPlasmaBlock() error
	GetPendingTxs() (*pctypes.PendingTxs, error)
	ProcessRequestBatch(requestBatch *pctypes.PlasmaCashRequestBatch) error
	GetRequestBatchTally() (*pctypes.PlasmaCashRequestBatchTally, error)
}

type DAppChainPlasmaClientImpl struct {
	DAppChainPlasmaClientConfig
	plasmaContract *client.Contract
	caller         loom.Address
}

func (c *DAppChainPlasmaClientImpl) GetPendingTxs() (*pctypes.PendingTxs, error) {
	req := &pctypes.GetPendingTxsRequest{}
	resp := &pctypes.PendingTxs{}
	if _, err := c.plasmaContract.StaticCall("GetPendingTxs", req, c.caller, resp); err != nil {
		return nil, errors.Wrap(err, "failed to call GetPendingTxs")
	}

	return resp, nil
}

func (c *DAppChainPlasmaClientImpl) Init() error {
	dappClient := client.NewDAppChainRPCClient(c.ChainID, c.WriteURI, c.ReadURI)
	contractAddr, err := dappClient.Resolve(c.ContractName)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve Plasma Go contract: %s address", c.ContractName)
	}
	c.plasmaContract = client.NewContract(dappClient, contractAddr.Local)
	c.caller = loom.Address{
		ChainID: c.ChainID,
		Local:   loom.LocalAddressFromPublicKey(c.Signer.PublicKey()),
	}
	return nil
}

func (c *DAppChainPlasmaClientImpl) CurrentPlasmaBlockNum() (*big.Int, error) {
	req := &pctypes.GetCurrentBlockRequest{}
	resp := &pctypes.GetCurrentBlockResponse{}
	if _, err := c.plasmaContract.StaticCall("GetCurrentBlockRequest", req, c.caller, resp); err != nil {
		return nil, errors.Wrap(err, "failed to call GetCurrentBlockRequest")
	}
	return resp.BlockHeight.Value.Int, nil
}

func (c *DAppChainPlasmaClientImpl) PlasmaBlockAt(blockNum *big.Int) (*pctypes.PlasmaBlock, error) {
	req := &pctypes.GetBlockRequest{
		BlockHeight: &ltypes.BigUInt{Value: *loom.NewBigUInt(blockNum)},
	}
	resp := &pctypes.GetBlockResponse{}
	if _, err := c.plasmaContract.StaticCall("GetBlockRequest", req, c.caller, resp); err != nil {
		return nil, errors.Wrap(err, "failed to obtain plasma block from Loom Protocol")
	}
	if resp.Block == nil {
		return nil, errors.New("Loom Protocol returned an empty plasma block")
	}
	return resp.Block, nil
}

func (c *DAppChainPlasmaClientImpl) FinalizeCurrentPlasmaBlock() error {
	breq := &pctypes.SubmitBlockToMainnetRequest{}
	if _, err := c.plasmaContract.Call("SubmitBlockToMainnet", breq, c.Signer, nil); err != nil {
		return errors.Wrap(err, "failed to commit SubmitBlockToMainnet tx")
	}
	return nil
}

func (c *DAppChainPlasmaClientImpl) GetRequestBatchTally() (*pctypes.PlasmaCashRequestBatchTally, error) {
	req := &pctypes.PlasmaCashGetRequestBatchTallyRequest{}
	resp := &pctypes.PlasmaCashRequestBatchTally{}
	if _, err := c.plasmaContract.StaticCall("GetRequestBatchTally", req, c.caller, resp); err != nil {
		return nil, errors.Wrap(err, "failed to get request batch tally")
	}

	return resp, nil
}

func (c *DAppChainPlasmaClientImpl) ProcessRequestBatch(requestBatch *pctypes.PlasmaCashRequestBatch) error {
	if _, err := c.plasmaContract.Call("ProcessRequestBatch", requestBatch, c.Signer, nil); err != nil {
		return errors.Wrap(err, "failed to commit process request batch tx")
	}

	return nil
}
