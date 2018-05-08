package rpc

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	proto "github.com/gogo/protobuf/proto"
	lp "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain"
	llog "github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	rpcclient "github.com/tendermint/tendermint/rpc/lib/client"
)

type queryableContract struct {
	llog.TMLogger
}

func (c *queryableContract) Meta() (lp.Meta, error) {
	return lp.Meta{
		Name:    "queryable",
		Version: "1.0.0",
	}, nil
}

func (c *queryableContract) Init(ctx lp.Context, req *plugin.Request) error {
	return nil
}

func (c *queryableContract) Call(ctx lp.Context, req *plugin.Request) (*lp.Response, error) {
	return &plugin.Response{}, nil
}

func (c *queryableContract) StaticCall(ctx lp.StaticContext, req *lp.Request) (*lp.Response, error) {
	cmc := &lp.ContractMethodCall{}
	if req.ContentType == lp.EncodingType_PROTOBUF3 {
		if err := proto.Unmarshal(req.Body, cmc); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("unsupported content type")
	}
	if "ping" == cmc.Method {
		var body []byte
		var err error
		if req.Accept == lp.EncodingType_PROTOBUF3 {
			body, err = proto.Marshal(&lp.ContractMethodCall{
				Method: "pong",
			})
			if err != nil {
				return nil, err
			}
			return &plugin.Response{
				ContentType: lp.EncodingType_PROTOBUF3,
				Body:        body,
			}, nil
		}
		return nil, errors.New("unsupported content type")
	}
	return nil, errors.New("invalid query")
}

type queryableContractLoader struct {
	llog.TMLogger
}

func (l *queryableContractLoader) LoadContract(name string) (lp.Contract, error) {
	return &queryableContract{TMLogger: l.TMLogger}, nil
}

type stateProvider struct {
}

func (s *stateProvider) ReadOnlyState() loomchain.State {
	return loomchain.NewStoreState(
		nil,
		store.NewMemStore(),
		abci.Header{},
	)
}

const queryServerHost = "127.0.0.1:9999"

func TestQueryServerContractQuery(t *testing.T) {
	loader := &queryableContractLoader{TMLogger: llog.Root.With("module", "contract")}
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
	params["contract"] = "0x005B17864f3adbF53b1384F2E6f2120c6652F779"
	pingMsg, err := proto.Marshal(&lp.ContractMethodCall{Method: "ping"})
	require.Nil(t, err)
	params["query"] = pingMsg

	var rawResult []byte
	var result lp.ContractMethodCall

	// JSON-RCP 2.0
	rpcClient := rpcclient.NewJSONRPCClient(host)
	_, err = rpcClient.Call("query", params, &rawResult)
	require.Nil(t, err)
	err = proto.Unmarshal(rawResult, &result)
	require.Nil(t, err)
	require.Equal(t, "pong", result.Method)

	// HTTP
	httpClient := rpcclient.NewURIClient(host)
	_, err = httpClient.Call("query", params, &rawResult)
	require.Nil(t, err)
	err = proto.Unmarshal(rawResult, &result)
	require.Nil(t, err)
	require.Equal(t, "pong", result.Method)

	// Invalid query
	pongMsg, err := proto.Marshal(&lp.ContractMethodCall{Method: "pong"})
	require.Nil(t, err)
	params["contract"] = "0x005B17864f3adbF53b1384F2E6f2120c6652F779"
	params["query"] = pongMsg
	_, err = rpcClient.Call("query", params, &rawResult)
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
