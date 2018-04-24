package plugin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/sha3"

	"github.com/loomnetwork/loom"
	lp "github.com/loomnetwork/loom-plugin"
	"github.com/loomnetwork/loom-plugin/types"
	"github.com/loomnetwork/loom-plugin/util"
	"github.com/loomnetwork/loom/auth"
	"github.com/loomnetwork/loom/vm"
)

type Request = types.Request
type Response = types.Response
type PluginCode = types.PluginCode

const EncodingType_JSON = types.EncodingType_JSON

type Loader interface {
	LoadContract(name string) (lp.Contract, error)
}

func contractPrefix(addr lp.Address) []byte {
	return util.PrefixKey([]byte("contract"), []byte(addr.Local))
}

func textKey(addr lp.Address) []byte {
	return util.PrefixKey(contractPrefix(addr), []byte("text"))
}

func dataPrefix(addr lp.Address) []byte {
	return util.PrefixKey(contractPrefix(addr), []byte("data"))
}

type PluginVM struct {
	Loader Loader
	State  loom.State
}

var _ vm.VM = &PluginVM{}

func (vm *PluginVM) run(
	caller,
	addr lp.Address,
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
		err = contract.Init(contractCtx, req)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(&PluginCode{
			Name: pluginCode.Name,
		})
	}

	if readOnly {
		res, err = contract.StaticCall(contractCtx, req)
	} else {
		res, err = contract.Call(contractCtx, req)
	}

	if err != nil {
		return nil, err
	}

	return proto.Marshal(res)
}

func createAddress(parent lp.Address, nonce uint64) lp.Address {
	var nonceBuf bytes.Buffer
	binary.Write(&nonceBuf, binary.BigEndian, nonce)
	data := util.PrefixKey(parent.Bytes(), nonceBuf.Bytes())
	hash := sha3.Sum256(data)
	return lp.Address{
		ChainID: parent.ChainID,
		Local:   hash[12:],
	}
}

func (vm *PluginVM) Create(caller lp.Address, code []byte) ([]byte, lp.Address, error) {
	nonce := auth.Nonce(vm.State, caller)
	contractAddr := createAddress(caller, nonce)

	ret, err := vm.run(caller, contractAddr, code, nil, false)
	if err != nil {
		return nil, contractAddr, err
	}

	vm.State.Set(textKey(contractAddr), ret)
	return ret, contractAddr, nil
}

func (vm *PluginVM) Call(caller, addr lp.Address, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, code, input, false)
}

func (vm *PluginVM) StaticCall(caller, addr lp.Address, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, code, input, true)
}

type contractContext struct {
	caller  lp.Address
	address lp.Address
	loom.State
	vm.VM
}

var _ lp.Context = &contractContext{}

func (c *contractContext) Call(addr lp.Address, input []byte) ([]byte, error) {
	return c.VM.Call(c.address, addr, input)
}

func (c *contractContext) StaticCall(addr lp.Address, input []byte) ([]byte, error) {
	return c.VM.StaticCall(c.address, addr, input)
}

func (c *contractContext) Message() types.Message {
	return types.Message{
		Sender: c.caller.MarshalPB(),
	}
}

func (c *contractContext) ContractAddress() lp.Address {
	return c.address
}

func (c *contractContext) Now() time.Time {
	return time.Unix(c.State.Block().Time, 0)
}

func (c *contractContext) Emit(event []byte) {

}
