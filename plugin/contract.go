package plugin

import (
	"errors"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/loom"
	"github.com/loomnetwork/loom/store"
	"github.com/loomnetwork/loom/util"
	"github.com/loomnetwork/loom/vm"
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

type Contract interface {
	Meta() Meta
	Init(ctx Context, req *Request) (*Response, error)
	Call(ctx Context, req *Request) (*Response, error)
	StaticCall(ctx StaticContext, req *Request) (*Response, error)
}

type Loader interface {
	LoadContract(name string) (Contract, error)
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

type PluginVM struct {
	Loader Loader
	State  loom.State
}

var _ vm.VM = &PluginVM{}

func (vm *PluginVM) run(
	caller,
	addr loom.Address,
	code,
	input []byte,
	readOnly bool,
) ([]byte, error) {
	var pluginCode PluginCode
	err := proto.Unmarshal(code, &pluginCode)
	if err != nil {
		return nil, err
	}

	contract, err := vm.Loader.LoadContract(pluginCode.Name)
	if err != nil {
		return nil, err
	}

	contractCtx := &contractContext{
		caller:  caller,
		address: addr,
		State:   loom.StateWithPrefix(dataPrefix(addr), vm.State),
		VM:      vm,
	}

	isInit := len(input) == 0
	if isInit {
		input = pluginCode.Input
	}

	req := &Request{}
	err = proto.Unmarshal(input, req)
	if err != nil {
		return nil, err
	}

	var res *Response
	if isInit {
		res, err = contract.Init(contractCtx, req)
	} else if readOnly {
		res, err = contract.StaticCall(contractCtx, req)
	} else {
		res, err = contract.Call(contractCtx, req)
	}

	if err != nil {
		return nil, err
	}

	return proto.Marshal(res)
}

func (vm *PluginVM) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	// TODO: create dynamic address
	contractAddr := loom.Address{
		ChainID: caller.ChainID,
		Local:   loom.LocalAddress(make([]byte, 20, 20)),
	}

	ret, err := vm.run(caller, contractAddr, code, nil, false)
	if err != nil {
		return nil, contractAddr, err
	}

	vm.State.Set(textKey(contractAddr), code)
	return ret, contractAddr, nil
}

func (vm *PluginVM) Call(caller, addr loom.Address, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, code, input, false)
}

func (vm *PluginVM) StaticCall(caller, addr loom.Address, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, code, input, true)
}

type contractContext struct {
	caller  loom.Address
	address loom.Address
	loom.State
	vm.VM
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
