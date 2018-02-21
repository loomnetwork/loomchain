package loom

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

// NodeProxy exposes a REST API for sending txs to a DAppChain node.
type NodeProxy struct {
	rpcClient rpcclient.Client
}

func NewNodeProxy(c rpcclient.Client) *NodeProxy {
	return &NodeProxy{
		rpcClient: c,
	}
}

func (np *NodeProxy) CommitTx(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.Body == nil {
		WriteError(w, errors.New("expecting a non-nil body"))
		return
	}
	defer r.Body.Close()

	tx, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, err)
		return
	}

	resp, err := np.rpcClient.BroadcastTxCommit(tx)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteSuccess(w, resp)
}

// RegisterCommitTx register a mux.Router handler that exposes a POST endpoint for committing
// transaction to the DAppChain. The response to the POST request will contain the output from the
// tx handler that processed the tx in the DAppChain node. Note that the response will only be
// sent back after the block the tx ends up in gets committed.
func (np *NodeProxy) RegisterCommitTx(r *mux.Router) error {
	r.HandleFunc("/tx", np.CommitTx).Methods("POST")
	return nil
}
