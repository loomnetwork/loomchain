package client

import (
	"errors"

	"github.com/gogo/protobuf/proto"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom/auth"
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

func (c *DAppChainRPCClient) CommitTx(signer lp.Signer, txBytes []byte) error {
	signedTx := auth.SignTx(signer, txBytes)
	signedTxBytes, err := proto.Marshal(signedTx)
	r, err := c.rpcClient.BroadcastTxCommit(signedTxBytes)
	if err != nil {
		return err
	}
	if r.CheckTx.Code != 0 {
		if len(r.CheckTx.Log) != 0 {
			return errors.New(r.CheckTx.Log)
		}
		return errors.New("CheckTx failed")
	}
	if r.DeliverTx.Code != 0 {
		if len(r.DeliverTx.Log) != 0 {
			return errors.New(r.DeliverTx.Log)
		}
		return errors.New("DeliverTx failed")
	}
	return nil
}
