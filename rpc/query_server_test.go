package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

type rpcRequest struct {
	Body string `json:"body"`
}
type rpcResponse struct {
	Body string `json:"body"`
}

type queryableContract struct {
	llog.Logger
}

func (c *queryableContract) Meta() plugin.Meta {
	return plugin.Meta{
		Name:    "queryable",
		Version: "1.0.0",
	}
}

func (c *queryableContract) Init(ctx plugin.Context, req *plugin.Request) error {
	return nil
}

func (c *queryableContract) Call(ctx plugin.Context, req *plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{}, nil
}

func (c *queryableContract) StaticCall(ctx plugin.StaticContext, req *plugin.Request) (*plugin.Response, error) {
	rr := &rpcRequest{}
	if req.ContentType == plugin.ContentType_JSON {
		if err := json.Unmarshal(req.Body, rr); err != nil {
			return nil, err
		}
	} else {
		// content type could also be protobuf
		return nil, errors.New("unsupported content type")
	}
	if "ping" == rr.Body {
		var body []byte
		var err error
		if req.Accept == plugin.ContentType_JSON {
			body, err = json.Marshal(&rpcResponse{Body: "pong"})
			if err != nil {
				return nil, err
			}
			return &plugin.Response{
				ContentType: plugin.ContentType_JSON,
				Body:        body,
			}, nil
		}
		// accepted content type could also be protobuf
		return nil, errors.New("unsupported content type")
	}
	return nil, errors.New("invalid query")
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

const queryServerHost = "127.0.0.1:9999"

func TestQueryServerContractQuery(t *testing.T) {
	loader := &queryableContractLoader{Logger: llog.Root.With("module", "contract")}
	host := "tcp://" + queryServerHost
	qs := QueryServer{
		StateProvider: &stateProvider{},
		Host:          host,
		Loader:        loader,
		Logger:        llog.Root.With("module", "query-server"),
	}
	qs.Start()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	params := map[string]interface{}{}
	params["contract"] = []byte("0x0")
	params["query"] = json.RawMessage(`{"body":"ping"}`)
	var result rpcResponse

	// JSON-RCP 2.0
	rpcClient := rpcclient.NewJSONRPCClient(host)
	_, err := rpcClient.Call("query", params, &result)
	require.Nil(t, err)
	require.Equal(t, "pong", result.Body)

	// HTTP
	httpClient := rpcclient.NewURIClient(host)
	_, err = httpClient.Call("query", params, &result)
	require.Nil(t, err)
	require.Equal(t, "pong", result.Body)

	// Invalid query
	params["query"] = json.RawMessage(`{"body":"pong"}`)
	_, err = rpcClient.Call("query", params, &result)
	require.NotNil(t, err)
	require.Equal(t, "Response error: RPC error -32603 - Internal error: invalid query", err.Error())
}

func TestQueryServerNonce(t *testing.T) {
	host := "tcp://" + queryServerHost
	qs := QueryServer{
		StateProvider: &stateProvider{},
		Host:          host,
		Logger:        llog.Root.With("module", "query-server"),
	}
	qs.Start()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	pubKey := "441B9DCC47A734695A508EDF174F7AAF76DD7209DEA2D51D3582DA77CE2756BE"

	_, err := http.Get(fmt.Sprintf("http://%s/nonce?key=\"%s\"", queryServerHost, pubKey))
	require.Nil(t, err)

	params := map[string]interface{}{}
	params["key"] = pubKey
	var result uint64

	// JSON-RCP 2.0
	rpcClient := rpcclient.NewJSONRPCClient(host)
	_, err = rpcClient.Call("nonce", params, &result)
	require.Nil(t, err)
}
