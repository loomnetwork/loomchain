package rpc

//NOTE THIS PACKAGE IS TO  STRESS A TESTNET INSTANCE
//NEVER USE THIS CODE IN A REAL NETWORK

import (
	"log"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/loomnetwork/loomchain"
)

type unsafeHandler struct {
	app *loomchain.Application
	svc QueryService
}

func newUnsafeHandler(app *loomchain.Application, svc QueryService) *unsafeHandler {
	return &unsafeHandler{app: app, svc: svc}
}

func (u *unsafeHandler) unsafeStartCPU(w http.ResponseWriter, req *http.Request) {
	f, err := os.Create("cpu_profile.txt")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)

	w.Write([]byte("starting\n"))
	w.WriteHeader(200)
}

func (u *unsafeHandler) unsafeStopCPU(w http.ResponseWriter, req *http.Request) {
	pprof.StopCPUProfile()
	w.WriteHeader(200)
}

/*
Simple way to play around with Read only mode

#deploy
./loom deploy -b ./e2e/test-data/evm/SimpleStore.bin -n SimpleStore -k ./e2e/test-data/evm/privkey-0

#write
./loom callevm -i  ./e2e/test-data/evm/inputSet987.bin -n SimpleStore -k ./e2e/test-data/evm/privkey-0

#read
./loom static-call-evm -i ./e2e/test-data/evm/inputGet.bin -n SimpleStore

*/
