package client

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/auth"
	pt "github.com/loomnetwork/loom/plugin"
	lt "github.com/loomnetwork/loom/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

// Implements the DAppChainClient interface via Tendermint RPC
type DAppChainRPCClient struct {
	rpcClient rpcclient.Client
}

func NewDAppChainRPCClient(nodeURI string) *DAppChainRPCClient {
	return &DAppChainRPCClient{
		rpcClient: rpcclient.NewHTTP(nodeURI, "/websocket"),
	}
}

func (c *DAppChainRPCClient) CommitTx(signer lp.Signer, txBytes []byte) ([]byte, error) {
	signedTx := auth.SignTx(signer, txBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	r, err := c.rpcClient.BroadcastTxCommit(signedTxBytes)
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

func (c *DAppChainRPCClient) CommitDeployTx(from lp.Address, signer lp.Signer, vm lp.VMType, code []byte) ([]byte, error) {
	deployTx := &pt.DeployTx{
		VmType: pt.VMType(int32(vm)),
		Code:   code,
	}
	deployTxBytes, err := proto.Marshal(deployTx)
	if err != nil {
		return nil, err
	}
	msgTx := &pt.MessageTx{
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

func (c *DAppChainRPCClient) CommitCallTx(from lp.Address, to lp.Address, signer lp.Signer, input []byte) ([]byte, error) {
	callTxBytes, err := proto.Marshal(&pt.CallTx{Input: input})
	if err != nil {
		return nil, err
	}
	msgTx := &pt.MessageTx{
		// TODO: lp.Address -> lt.Address
		From: nil, // caller
		To:   nil, // contract address
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
