package contract

import (
	"errors"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	"github.com/loomnetwork/loom/util"
)

type StaticAPI interface {
	StaticCall(addr loom.Address, input []byte) ([]byte, error)
}

type VolatileAPI interface {
	Call(addr loom.Address, input []byte) ([]byte, error)
}

type Message struct {
	Sender loom.Address
}

type StaticContext interface {
	StaticAPI
	loom.ReadOnlyState
	Now() time.Time
	Message() Message
	ContractAddress() loom.Address
}

type Context interface {
	StaticContext
	VolatileAPI
	store.KVWriter
	Emit(event []byte)
}

type PluginContract interface {
	Meta() PluginMeta
	Call(ctx Context, input []byte) ([]byte, error)
	StaticCall(ctx StaticContext, input []byte) ([]byte, error)
}

type PluginLoader interface {
	LoadContract(name string) (PluginContract, error)
}

type CallTxHandler struct {
	PluginLoader
}

func (h *CallTxHandler) ProcessTx(
	state loom.State,
	txBytes []byte,
) (loom.TxHandlerResult, error) {
	var r loom.TxHandlerResult

	var pbMsg MessageTx
	err := proto.Unmarshal(txBytes, &pbMsg)
	if err != nil {
		return r, err
	}

	var caller, addr loom.Address
	caller.UnmarshalPB(pbMsg.From)
	addr.UnmarshalPB(pbMsg.To)

	vm := &PluginVM{
		Loader: h.PluginLoader,
		State:  state,
	}

	_, err = vm.Call(caller, addr, pbMsg.Data)
	return r, err
}

func contractPrefix(addr loom.Address) []byte {
	return util.PrefixKey([]byte("contract"), []byte(addr.Local))
}

func textKey(addr loom.Address) []byte {
	return util.PrefixKey(contractPrefix(addr), []byte("text"))
}

func dataPrefix(addr loom.Address) []byte {
	return util.PrefixKey(contractPrefix(addr), []byte("data"))
}

type VM interface {
	Create(caller loom.Address, code []byte) ([]byte, loom.Address, error)
	Call(caller, addr loom.Address, input []byte) ([]byte, error)
	StaticCall(caller, addr loom.Address, input []byte) ([]byte, error)
}

type PluginVM struct {
	Loader PluginLoader
	State  loom.State
}

func (vm *PluginVM) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	// TODO: create dynamic address
	contractAddr := loom.Address{
		ChainID: caller.ChainID,
		Local:   loom.LocalAddress(make([]byte, 20, 20)),
	}

	_, err := vm.Loader.LoadContract(string(code))
	if err != nil {
		return nil, contractAddr, err
	}

	vm.State.Set(textKey(contractAddr), code)

	return nil, contractAddr, nil
}

func (vm *PluginVM) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	code := vm.State.Get(textKey(addr))
	contract, err := vm.Loader.LoadContract(string(code))
	if err != nil {
		return nil, err
	}

	contractCtx := &contractContext{
		caller:  caller,
		address: addr,
		State:   loom.StateWithPrefix(dataPrefix(addr), vm.State),
		VM:      vm,
	}
	return contract.Call(contractCtx, input)
}

func (vm *PluginVM) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}

type contractContext struct {
	caller  loom.Address
	address loom.Address
	loom.State
	VM
}

func (c *contractContext) Call(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.Call(c.address, addr, input)
}

func (c *contractContext) StaticCall(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.StaticCall(c.address, addr, input)
}

func (c *contractContext) Message() Message {
	return Message{
		Sender: c.caller,
	}
}

func (c *contractContext) ContractAddress() loom.Address {
	return c.address
}

func (c *contractContext) Now() time.Time {
	return time.Unix(c.State.Block().Time, 0)
}

func (c *contractContext) Emit(event []byte) {

}
