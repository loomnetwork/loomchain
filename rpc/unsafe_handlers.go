package rpc

import (
	"log"
	"net/http"
	"os"
	"runtime/pprof"

	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/vm"
	"golang.org/x/crypto/ed25519"
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
	unsafeDeployEVMTestApp(u.app)

	//TODO read query string to know how many iteration
	to := loom.Address{}
	for i := 1; i < 10; i++ {
		unsafeLoadDeliverTx(u.app, i, to)

	}
	w.Write([]byte("unsafeLoadDeliverTx finished\n"))
	w.WriteHeader(200)
}

func unsafeDeployEVMTestApp(app *loomchain.Application) loom.Address {
	//TODO Deploy EVM TX
	contract := loom.MustParseAddress("default:0x9a1aC42a17AAD6Dbc6d21c162989d0f701074044")

	return contract
}

func unsafeLoadDeliverTx(app *loomchain.Application, round int, to loom.Address) {

	origBytes := []byte("origin")
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	panicErr(err)

	origin := loom.Address{
		ChainID: "default",
		Local:   loom.LocalAddressFromPublicKey(pubKey),
	}

	var messageTx []byte

	deployTX, err := proto.Marshal(&vm.DeployTx{
		VmType: vm.VMType_EVM,
		Code:   origBytes,
	})
	panicErr(err)

	messageTx, err = proto.Marshal(&vm.MessageTx{
		Data: deployTX,
		To:   to.MarshalPB(),
		From: origin.MarshalPB(),
	})

	tx, err := proto.Marshal(&loomchain.Transaction{
		Id:   uint32(round),
		Data: messageTx,
	})
	nonceTx, err := proto.Marshal(&auth.NonceTx{
		Inner:    tx,
		Sequence: uint64(round),
	})

	signer := auth.NewEd25519Signer(privKey)
	signedTx := auth.SignTx(signer, nonceTx)
	signedTxBytes, err := proto.Marshal(signedTx)
	panicErr(err)

	app.DeliverTx(signedTxBytes)
}

func panicErr(err error) {
	if err != nil {
		panic("Failed doing something:" + err.Error())
	}
}
