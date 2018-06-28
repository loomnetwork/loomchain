package rpc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	proto "github.com/gogo/protobuf/proto"
	lp "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/subs"
	llog "github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/store"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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

func (l *queryableContractLoader) UnloadContracts() {}

type stateProvider struct {
}

func (s *stateProvider) ReadOnlyState() loomchain.State {
	return loomchain.NewStoreState(
		nil,
		store.NewMemStore(),
		abci.Header{},
	)
}

var testlog llog.TMLogger

func TestQueryServer(t *testing.T) {
	llog.Setup("debug", "file://-")
	testlog = llog.Root.With("module", "query-server")
	t.Run("Contract Query", testQueryServerContractQuery)
	t.Run("Query Nonce", testQueryServerNonce)
	t.Run("Query Metric", testQueryMetric)
}

func testQueryServerContractQuery(t *testing.T) {
	loader := &queryableContractLoader{TMLogger: llog.Root.With("module", "contract")}
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
		Loader:        loader,
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
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
	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	_, err = rpcClient.Call("query", params, &rawResult)
	require.Nil(t, err)
	err = proto.Unmarshal(rawResult, &result)
	require.Nil(t, err)
	require.Equal(t, "pong", result.Method)

	// HTTP
	httpClient := rpcclient.NewURIClient(ts.URL)
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

func testQueryServerNonce(t *testing.T) {
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	pubKey := "441B9DCC47A734695A508EDF174F7AAF76DD7209DEA2D51D3582DA77CE2756BE"

	_, err := http.Get(fmt.Sprintf("%s/nonce?key=\"%s\"", ts.URL, pubKey))
	require.Nil(t, err)

	params := map[string]interface{}{}
	params["key"] = pubKey
	var result uint64

	// JSON-RCP 2.0
	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	_, err = rpcClient.Call("nonce", params, &result)
	require.Nil(t, err)
}

func testQueryMetric(t *testing.T) {
	// add metrics
	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "loomchain",
		Subsystem: "query_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "loomchain",
		Subsystem: "query_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	loader := &queryableContractLoader{TMLogger: llog.Root.With("module", "contract")}

	// create query service
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
		Loader:        loader,
	}
	qs = InstrumentingMiddleware{requestCount, requestLatency, qs}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	// HTTP
	pubKey := "441B9DCC47A734695A508EDF174F7AAF76DD7209DEA2D51D3582DA77CE2756BE"
	_, err := http.Get(fmt.Sprintf("%s/nonce?key=\"%s\"", ts.URL, pubKey))
	if err != nil {
		t.Fatal(err)
	}
	// JSON-RCP 2.0
	params := map[string]interface{}{}
	params["key"] = pubKey
	var result uint64
	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	_, err = rpcClient.Call("nonce", params, &result)

	var rawResult []byte
	// HTTP
	httpClient := rpcclient.NewURIClient(ts.URL)
	_, _ = httpClient.Call("query", params, &rawResult)

	// Invalid query
	pongMsg, _ := proto.Marshal(&lp.ContractMethodCall{Method: "pong"})
	params["contract"] = "0x005B17864f3adbF53b1384F2E6f2120c6652F779"
	params["query"] = pongMsg
	_, _ = rpcClient.Call("query", params, &rawResult)
	// require.Equal(t, "Response error: RPC error -32603 - Internal error: invalid query", err.Error())

	// query metrics
	resp, err := http.Get(fmt.Sprintf("%s/metrics", ts.URL))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("want metric status code 200, got %d", resp.StatusCode)
	}
	data, _ := ioutil.ReadAll(resp.Body)

	wkey := `loomchain_query_service_request_count{error="false",method="Nonce"} 2`
	if !strings.Contains(string(data), wkey) {
		t.Errorf("want metric '%s', got none", wkey)
	}
	wkey = `loomchain_query_service_request_count{error="true",method="Query"} 2`
	if !strings.Contains(string(data), wkey) {
		t.Errorf("want metric '%s', got none", wkey)
	}
}
