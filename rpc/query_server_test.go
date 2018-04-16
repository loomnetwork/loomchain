package rpc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/loomnetwork/loom"
	llog "github.com/loomnetwork/loom/log"
	"github.com/loomnetwork/loom/plugin"
	"github.com/loomnetwork/loom/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	rpcclient "github.com/tendermint/tendermint/rpc/lib/client"
)

type queryableContract struct {
	llog.Logger
}

func (c *queryableContract) Meta() plugin.Meta {
	return plugin.Meta{
		Name:    "queryable",
		Version: "1.0.0",
	}
}

func (c *queryableContract) Init(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{}, nil
}

func (c *queryableContract) Call(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{}, nil
}

func (c *queryableContract) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error) {
	c.Logger.Info(fmt.Sprintf("contract.StaticCall('%s')", string(req.Body)))
	if bytes.Equal([]byte("ping"), req.Body) {
		return &plugin.Response{
			ContentType: plugin.ContentType_JSON,
			Body:        []byte("pong"),
		}, nil
	}
	return &plugin.Response{}, nil
}

type queryableContractLoader struct {
	llog.Logger
}

func (l *queryableContractLoader) LoadContract(name string) (plugin.Contract, error) {
	return &queryableContract{Logger: l.Logger}, nil
}

type stateProvider struct {
}

func (s *stateProvider) ReadOnlyState() loom.State {
	return loom.NewStoreState(
		nil,
		store.NewMemStore(),
		abci.Header{},
	)
}

func TestQueryServer(t *testing.T) {
	loader := &queryableContractLoader{Logger: llog.Root.With("module", "contract")}
	host := "tcp://127.0.0.1:9999"
	qs := QueryServer{
		StateProvider: &stateProvider{},
		Host:          host,
		Loader:        loader,
		Logger:        llog.Root.With("module", "query-server"),
	}
	qs.Start()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	client := rpcclient.NewJSONRPCClient(host)
	params := map[string]interface{}{}
	params["contract"] = []byte("0x0")
	params["query"] = []byte("ping")

	var result string
	_, err := client.Call("query", params, &result)
	require.Nil(t, err)
	// []byte result gets encoded as a base64 string
	r, err := base64.StdEncoding.DecodeString(result)
	require.Nil(t, err)
	require.Equal(t, "pong", string(r))
}
