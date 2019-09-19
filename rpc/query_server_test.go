package rpc

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	proto "github.com/gogo/protobuf/proto"
	lp "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/eth/subs"
	llog "github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	dbm "github.com/tendermint/tendermint/libs/db"
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

func (l *queryableContractLoader) LoadContract(name string, blockHeight int64) (lp.Contract, error) {
	return &queryableContract{TMLogger: l.TMLogger}, nil
}

func (l *queryableContractLoader) UnloadContracts() {}

type stateProvider struct {
	ChainID string
}

func (s *stateProvider) ReadOnlyState() state.State {
	return state.NewStoreState(
		nil,
		store.NewMemStore(),
		abci.Header{
			ChainID: s.ChainID,
		},
		nil,
		nil,
	)
}

var testlog llog.TMLogger

func TestQueryServer(t *testing.T) {
	llog.Setup("debug", "file://-")
	testlog = llog.Root.With("module", "query-server")
	t.Run("Contract Query", testQueryServerContractQuery)
	t.Run("Query Nonce", testQueryServerNonce)
	t.Run("Query Metric", testQueryMetric)
	t.Run("Query Contract Events", testQueryServerContractEvents)
	t.Run("Query Contract Events Without Event", testQueryServerContractEventsNoEventStore)
	t.Run("Query Contract Information", testQueryServerGetContractRecord)
}

func testQueryServerContractQuery(t *testing.T) {
	loader := &queryableContractLoader{TMLogger: llog.Root.With("module", "contract")}
	createRegistry, err := registry.NewRegistryFactory(registry.LatestRegistryVersion)
	require.NoError(t, err)
	var qs QueryService = &QueryServer{
		StateProvider:  &stateProvider{},
		Loader:         loader,
		CreateRegistry: createRegistry,
		BlockStore:     store.NewMockBlockStore(),
		AuthCfg:        auth.DefaultConfig(),
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
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
		ChainID: "default",
		StateProvider: &stateProvider{
			ChainID: "default",
		},
		BlockStore: store.NewMockBlockStore(),
		AuthCfg:    auth.DefaultConfig(),
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	pubKey := "441B9DCC47A734695A508EDF174F7AAF76DD7209DEA2D51D3582DA77CE2756BE"
	account := "default:0xb16a379ec18d4093666f8f38b11a3071c920207d"

	// Query for nonce using public key
	_, err := http.Get(fmt.Sprintf("%s/nonce?key=\"%s\"", ts.URL, pubKey))
	require.NoError(t, err)

	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	var result uint64

	_, err = rpcClient.Call("nonce", map[string]interface{}{"key": pubKey}, &result)
	require.NoError(t, err)

	// Query for nonce using account address
	_, err = http.Get(fmt.Sprintf("%s/nonce?account=\"%s\"", ts.URL, account))
	require.NoError(t, err)

	_, err = rpcClient.Call("nonce", map[string]interface{}{"account": account}, &result)
	require.NoError(t, err)

	// Query for nonce using both account address & public key
	_, err = http.Get(fmt.Sprintf("%s/nonce?key=\"%s\"&account=\"%s\"", ts.URL, pubKey, account))
	require.NoError(t, err)

	_, err = rpcClient.Call("nonce", map[string]interface{}{"key": pubKey, "account": account}, &result)
	require.NoError(t, err)
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
		Namespace:  "loomchain",
		Subsystem:  "query_service",
		Name:       "request_latency_microseconds",
		Help:       "Total duration of requests in microseconds.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, fieldKeys)

	loader := &queryableContractLoader{TMLogger: llog.Root.With("module", "contract")}
	createRegistry, err := registry.NewRegistryFactory(registry.LatestRegistryVersion)
	require.NoError(t, err)
	// create query service
	var qs QueryService = &QueryServer{
		ChainID: "default",
		StateProvider: &stateProvider{
			ChainID: "default",
		},
		Loader:         loader,
		CreateRegistry: createRegistry,
		BlockStore:     store.NewMockBlockStore(),
		AuthCfg:        auth.DefaultConfig(),
	}
	qs = InstrumentingMiddleware{requestCount, requestLatency, qs}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	// HTTP
	pubKey := "441B9DCC47A734695A508EDF174F7AAF76DD7209DEA2D51D3582DA77CE2756BE"
	_, err = http.Get(fmt.Sprintf("%s/nonce?key=\"%s\"", ts.URL, pubKey))
	if err != nil {
		t.Fatal(err)
	}
	// JSON-RCP 2.0
	params := map[string]interface{}{}
	params["key"] = pubKey
	var result uint64
	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	_, err = rpcClient.Call("nonce", params, &result)
	if err != nil {
		t.Fatal(err)
	}
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
	require.Contains(t, string(data), wkey, "want metric got none")
	wkey = `loomchain_query_service_request_count{error="true",method="Query"} 2`
	require.Contains(t, string(data), wkey, "want metric got none")
}

func testQueryServerContractEvents(t *testing.T) {
	memdb := dbm.NewMemDB()
	eventStore := store.NewKVEventStore(memdb)

	contractID := eventStore.GetContractID("plugin1")

	// populate events store
	var eventData []*types.EventData
	for i := 0; i < 10; i++ {
		event := types.EventData{
			BlockHeight:      1,
			TransactionIndex: uint64(i),
			EncodedBody:      []byte(fmt.Sprintf("event-%d-%d", 1, i)),
		}
		eventStore.SaveEvent(contractID, 1, uint16(i), &event)
		eventData = append(eventData, &event)
	}

	// build RPC QueryService
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
		BlockStore:    store.NewMockBlockStore(),
		EventStore:    eventStore,
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)

	t.Run("Test invalid FromBlock", func(t *testing.T) {
		// RPC request to fetch events
		params := map[string]interface{}{}

		// from block missing
		params["toBlock"] = 1
		params["contract"] = "plugin1"

		// JSON-RPC 2.0
		result := &types.ContractEventsResult{}
		_, err := rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)

		// from block = 0
		params["fromBlock"] = 0
		params["toBlock"] = 1
		params["contract"] = "plugin1"

		_, err = rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)
	})

	t.Run("Test query range check", func(t *testing.T) {
		// RPC request to fetch events
		params := map[string]interface{}{}

		// to block missing (should default to to=from)
		params["fromBlock"] = 1
		params["contract"] = "plugin1"

		result := &types.ContractEventsResult{}
		params["fromBlock"] = 1
		params["toBlock"] = 25
		params["contract"] = "plugin1"

		// ToBlock beyond default max range of 20
		params["fromBlock"] = 1
		params["toBlock"] = 25
		params["contract"] = "plugin1"

		_, err := rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)

		// exceeds custom max range
		params["fromBlock"] = 1
		params["toBlock"] = 25
		params["contract"] = "plugin1"

		_, err = rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)
	})

	t.Run("Test query max range cap", func(t *testing.T) {

		// RPC request to fetch events
		params := map[string]interface{}{}

		params["fromBlock"] = 1
		params["toBlock"] = 110
		params["contract"] = "plugin1"

		// JSON-RPC 2.0
		result := &types.ContractEventsResult{}
		_, err := rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)
	})
}

func testQueryServerContractEventsNoEventStore(t *testing.T) {
	// build RPC QueryService
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
		BlockStore:    store.NewMockBlockStore(),
		EventStore:    nil,
	}
	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)

	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)

	t.Run("Query should return error", func(t *testing.T) {
		// RPC request to fetch events
		params := map[string]interface{}{}

		// from block missing
		params["toBlock"] = 1
		params["contract"] = "plugin1"

		// JSON-RPC 2.0
		result := &types.ContractEventsResult{}
		_, err := rpcClient.Call("contractevents", params, result)
		require.NotNil(t, err)
	})
}

func testQueryServerGetContractRecord(t *testing.T) {
	var qs QueryService = &QueryServer{
		StateProvider: &stateProvider{},
		BlockStore:    store.NewMockBlockStore(),
	}

	bus := &QueryEventBus{
		Subs:    *loomchain.NewSubscriptionSet(),
		EthSubs: *subs.NewLegacyEthSubscriptionSet(),
	}
	handler := MakeQueryServiceHandler(qs, testlog, bus)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	// give the server some time to spin up
	time.Sleep(100 * time.Millisecond)
	rpcClient := rpcclient.NewJSONRPCClient(ts.URL)
	t.Run("Contract query should return error", func(t *testing.T) {
		params := map[string]interface{}{}
		params["contract"] = ""
		resp := &types.ContractRecordResponse{}
		_, err := rpcClient.Call("contractrecord", params, resp)
		require.NotNil(t, err)
	})
}
