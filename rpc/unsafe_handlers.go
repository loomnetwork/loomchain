package rpc

import (
	"log"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/loomnetwork/loomchain"
)

type unsafeHandler struct {
	app *loomchain.Application
}

func newUnsafeHandler(app *loomchain.Application) *unsafeHandler {
	return &unsafeHandler{app: app}
}

func (u *unsafeHandler) unsafeLoadDeliverTx(w http.ResponseWriter, req *http.Request) {
	//TODO for now we will always do a cpu profile
	//TODO read query string if, we need cpu, mem, or trace
	f, err := os.Create("cpu_profile_load_deliver_tx.txt")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	w.Write([]byte("unsafeLoadDeliverTx starting\n"))
	//TODO read query string to know how many iteration
	for i := 0; i < 10; i++ {
		unsafeLoadDeliverTx(u.app, i)

	}
	w.Write([]byte("unsafeLoadDeliverTx finished\n"))
	w.WriteHeader(200)
}

func unsafeLoadDeliverTx(app *loomchain.Application, round int) {
	//	app.DeliverTx()

}
