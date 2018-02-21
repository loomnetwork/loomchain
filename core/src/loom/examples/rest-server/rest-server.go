package main

import (
	"errors"
	"fmt"
	"log"
	"loom"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
	rpcserver "github.com/tendermint/tendermint/rpc/lib/server"
	"github.com/tendermint/tmlibs/cli"
	tmcmn "github.com/tendermint/tmlibs/common"
	tmlog "github.com/tendermint/tmlibs/log"
)

/*
The example REST server provides the following endpoints:
- /app/tx (POST) - forwards a tx to the DAppChain node, the node expects to receive a signed tx
				   containing a DummyTx with a key & value. The key & value will be stored by the
				   node, and can be queried via the /app/dummy endpoint.
- /app/dummy/<key> (GET) - looks up & returns the dummy tx data matching the given key.
*/

var serverCLICmd = &cobra.Command{
	Use:   "server",
	Short: "REST/WS server for interacting with a Loom DAppChain",
	RunE:  startServer,
}

const (
	nodeFlag = "node"
	hostFlag = "host"
	appPath  = "/app/"
)

func init() {
	_ = serverCLICmd.Flags().String(nodeFlag, "tcp://0.0.0.0:46657", "node URL, in the form tcp://<host>:<port>")
	_ = serverCLICmd.PersistentFlags().String(hostFlag, "tcp://127.0.0.1:8998", "host & port the server should listen on")
}

func startServer(cmd *cobra.Command, args []string) error {
	rootDir := viper.GetString(cli.HomeFlag)
	fmt.Printf("rootDir %s", rootDir)

	// Create RPC client to communicate with the DAppChain node
	rpcClient := rpcclient.NewHTTP(viper.GetString(nodeFlag), "/websocket")
	// Create a REST/JSONRPC/WS proxy to forward txs to the DAppChain node
	nodeProxy := loom.NewNodeProxy(rpcClient)
	// Register some additional application specific REST routes under /app/
	router := mux.NewRouter().Path(appPath).Subrouter()
	smux := http.NewServeMux()
	smux.Handle(appPath, router)
	appRoutes := newAppRoutes(rpcClient)
	if err := appRoutes.Register(router); err != nil {
		log.Fatal(err)
	}

	tmLogger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	proxyRPCRoutes := nodeProxy.RPCRoutes()
	rpcserver.RegisterRPCFuncs(smux, proxyRPCRoutes, tmLogger)
	wm := rpcserver.NewWebsocketManager(proxyRPCRoutes)
	smux.HandleFunc("/websocket", wm.WebsocketHandler)
	host := viper.GetString(hostFlag)
	log.Printf("Serving on %s", host)
	_, err := rpcserver.StartHTTPServer(host, smux, tmLogger)
	if err != nil {
		panic(err)
	}
	// Wait forever
	tmcmn.TrapSignal(func() {
		// cleanup
	})
	return nil
}

func main() {
	rootDir := "."
	cmd := cli.PrepareMainCmd(serverCLICmd, "LOOM_SDK_SAMPLE", rootDir)
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

type appRoutes struct {
	rpcClient rpcclient.Client
}

func newAppRoutes(c rpcclient.Client) *appRoutes {
	return &appRoutes{rpcClient: c}
}

func (ar *appRoutes) getLastDummyTxKey(w http.ResponseWriter, r *http.Request) {
	resp, err := ar.rpcClient.ABCIQuery("app/last-key", nil)
	if err != nil {
		loom.WriteError(w, err)
		return
	}
	if resp.Response.Code != 0 {
		loom.WriteError(w, errors.New(resp.Response.Log))
	}
	key := string(resp.Response.Value)
	loom.WriteSuccess(w, key)
}

// Looks up the dummy value stored for a specific key, and returns the key & value in the HTTP
// response body.
func (ar *appRoutes) getDummyData(w http.ResponseWriter, r *http.Request) {
	args := mux.Vars(r)
	key := args["key"]

	// TODO: Build an abstraction, users of the SDK shouldn't have to deal with the ABCI API
	//       directly, and certainly not ABCIQuery's wierd hex-bytes stuff.
	resp, err := ar.rpcClient.ABCIQuery("app/dummy", tmcmn.HexBytes(key))
	if err != nil {
		loom.WriteError(w, err)
		return
	}
	if resp.Response.Code != 0 {
		loom.WriteError(w, errors.New(resp.Response.Log))
	}
	// NOTE: We're reusing the DummyTx struct here for the sake of convenience, but you're not
	// constrained to returning tx types, any struct or primitive value that's serializable to
	// JSON can be easily returned in the HTTP response body.
	data := loom.DummyTx{
		Key: key,
		Val: string(resp.Response.Value),
	}
	loom.WriteSuccess(w, data)
}

// Register registers application specific mux.Router handlers.
func (ar *appRoutes) Register(r *mux.Router) error {
	r.HandleFunc("/last-key", ar.getLastDummyTxKey).Methods("GET")
	r.HandleFunc("/dummy/{key}", ar.getDummyData).Methods("GET")
	return nil
}
