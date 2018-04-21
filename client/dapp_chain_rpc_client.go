package client

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/auth"
	lt "github.com/loomnetwork/loom/types"
	"github.com/loomnetwork/loom/vm"
	tmrpcclient "github.com/tendermint/tendermint/rpc/client"
	rpcclient "github.com/tendermint/tendermint/rpc/lib/client"
)

// Implements the DAppChainClient interface via Tendermint RPC
type DAppChainRPCClient struct {
	tmClient    tmrpcclient.Client
	queryClient *rpcclient.JSONRPCClient
	chainID     string
}

// NewDAppChainRPCClient creates a new dumb client that can be used to commit txs and query contract
// state via RPC.
// baseURI should be specified as "tcp://<host>", writePort is the RPC port of the Tendermint node
// (46657 by default), readPort is the RPC port of the query server (47000 by default).
func NewDAppChainRPCClient(baseURI string, writePort, readPort int32) *DAppChainRPCClient {
	return &DAppChainRPCClient{
		tmClient:    tmrpcclient.NewHTTP(fmt.Sprintf("%s:%d", baseURI, writePort), "/websocket"),
		queryClient: rpcclient.NewJSONRPCClient(fmt.Sprintf("%s:%d", baseURI, readPort)),
		chainID:     "default",
	}
}

func (c *DAppChainRPCClient) GetNonce(signer lp.Signer) (uint64, error) {
	params := map[string]interface{}{}
	params["key"] = hex.EncodeToString(signer.PublicKey())
	var result uint64
	_, err := c.queryClient.Call("nonce", params, &result)
	return result, err
}

func (c *DAppChainRPCClient) CommitTx(signer lp.Signer, txBytes []byte) ([]byte, error) {
	// TODO: signing & noncing should be handled by middleware
	nonce, err := c.GetNonce(signer)
	if err != nil {
		return nil, err
	}
	nonceTx := &auth.NonceTx{
		Inner:    txBytes,
		Sequence: nonce + 1,
	}
	nonceTxBytes, err := proto.Marshal(nonceTx)
	if err != nil {
		return nil, err
	}
	signedTx := auth.SignTx(signer, nonceTxBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	if err != nil {
		return nil, err
	}

	r, err := c.tmClient.BroadcastTxCommit(signedTxBytes)
	if err != nil {
		return nil, err
	}
	if r.CheckTx.Code != 0 {
		if len(r.CheckTx.Log) != 0 {
			return nil, errors.New(r.CheckTx.Log)
		}
		return nil, errors.New("CheckTx failed")
	}
	if r.DeliverTx.Code != 0 {
		if len(r.DeliverTx.Log) != 0 {
			return nil, errors.New(r.DeliverTx.Log)
		}
		return nil, errors.New("DeliverTx failed")
	}
	return r.DeliverTx.Data, nil
}

func (c *DAppChainRPCClient) CommitDeployTx(
	from lp.Address,
	signer lp.Signer,
	vmType lp.VMType,
	code []byte) ([]byte, error) {
	deployTx := &vm.DeployTx{
		VmType: vm.VMType(vmType),
		Code:   code,
	}
	deployTxBytes, err := proto.Marshal(deployTx)
	if err != nil {
		return nil, err
	}
	// FIXME: Node will panic if any address is nil
	msgTx := &vm.MessageTx{
		// TODO: lp.Address -> lt.Address
		From: nil, // caller
		To:   nil, // not used
		Data: deployTxBytes,
	}
	msgBytes, err := proto.Marshal(msgTx)
	if err != nil {
		return nil, err
	}
	// tx ids associated with handlers in loadApp()
	tx := &lt.Transaction{
		Id:   2,
		Data: msgBytes,
	}
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}
	return c.CommitTx(signer, txBytes)
}

func (c *DAppChainRPCClient) CommitCallTx(
	caller lp.Address,
	contract lp.Address,
	signer lp.Signer,
	vmType lp.VMType,
	input []byte,
) ([]byte, error) {
	callTxBytes, err := proto.Marshal(&vm.CallTx{
		VmType: vm.VMType(vmType),
		Input:  input,
	})
	if err != nil {
		return nil, err
	}
	msgTx := &vm.MessageTx{
		// FIXME: OMG...
		From: (*(caller.(*loom.Address))).MarshalPB(),
		To:   (*(contract.(*loom.Address))).MarshalPB(),
		Data: callTxBytes,
	}
	msgBytes, err := proto.Marshal(msgTx)
	if err != nil {
		return nil, err
	}
	// tx ids associated with handlers in loadApp()
	tx := &lt.Transaction{
		Id:   2,
		Data: msgBytes,
	}
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return nil, err
	}
	return c.CommitTx(signer, txBytes)
}
