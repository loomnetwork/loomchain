package plugin

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/sha3"

	loom "github.com/loomnetwork/go-loom"
	lp "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	levm "github.com/loomnetwork/loomchain/evm"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/pkg/errors"
)

type (
	Request    = lp.Request
	Response   = lp.Response
	PluginCode = lp.Code
)

var (
	EncodingType_JSON = lp.EncodingType_JSON
)

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
	Loader       Loader
	State        loomchain.State
	Registry     registry.Registry
	EventHandler loomchain.EventHandler
	logger       *loom.Logger
	// If this is nil the EVM won't have access to any account balances.
	newABMFactory NewAccountBalanceManagerFactoryFunc
}

func NewPluginVM(
	loader Loader,
	state loomchain.State,
	registry registry.Registry,
	eventHandler loomchain.EventHandler,
	logger *loom.Logger,
	newABMFactory NewAccountBalanceManagerFactoryFunc,
) *PluginVM {
	return &PluginVM{
		Loader:        loader,
		State:         state,
		Registry:      registry,
		EventHandler:  eventHandler,
		logger:        logger,
		newABMFactory: newABMFactory,
	}
}

var _ vm.VM = &PluginVM{}

func (vm *PluginVM) createContractContext(
	caller,
	addr loom.Address,
	readOnly bool,
) *contractContext {
	return &contractContext{
		caller:       caller,
		address:      addr,
		State:        loomchain.StateWithPrefix(dataPrefix(addr), vm.State),
		VM:           vm,
		Registry:     vm.Registry,
		eventHandler: vm.EventHandler,
		readOnly:     readOnly,
		req:          &Request{},
		logger:       vm.logger,
	}
}

func validateInitAttempt(
	reg registry.Registry,
	caller loom.Address,
	contractName,
	contractVersion string) error {

	addr, err := reg.Resolve(contractName, contractVersion)
	if err == nil {
		return fmt.Errorf("contract with name: %s and version: %s already exists.", contractName, contractVersion)
	}

	// Get master entry. If it doesnt exists, than
	// it means plugin is being registered for first time
	// otherwise proceed with validation.
	addr, err = reg.Resolve(contractName, "")
	if err != nil {
		return nil
	}

	record, err := reg.GetRecord(addr)
	if err != nil {
		return err
	}

	if caller.Compare(loom.UnmarshalAddressPB(record.Owner)) != 0 {
		return fmt.Errorf("owner of initial version doesnt match caller.")
	}

	return nil
}

func (vm *PluginVM) run(
	caller,
	addr loom.Address,
	contractVersion string,
	code,
	input []byte,
	readOnly bool,
) ([]byte, error) {
	var pluginCode PluginCode
	err := proto.Unmarshal(code, &pluginCode)
	if err != nil {
		return nil, err
	}

	isInit := len(input) == 0
	if isInit {
		err := validateInitAttempt(vm.Registry, caller, pluginCode.Name, contractVersion)
		if err != nil {
			return nil, err
		}
	} else {
		if contractVersion == "" {
			contractVersion = pluginCode.InitialVersion
		}
	}

	contract, err := vm.Loader.LoadContract(pluginCode.Name, contractVersion)
	if err != nil {
		return nil, err
	}

	if isInit {
		input = pluginCode.Input
	}

	req := &Request{}
	err = proto.Unmarshal(input, req)
	if err != nil {
		return nil, err
	}

	contractCtx := vm.createContractContext(caller, addr, readOnly)
	contractCtx.pluginName = pluginCode.Name
	contractCtx.req = req

	var res *Response
	if isInit {
		err = contract.Init(contractCtx, req)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(&PluginCode{
			Name:           pluginCode.Name,
			InitialVersion: contractVersion,
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

func CreateAddress(parent loom.Address, nonce uint64) loom.Address {
	var nonceBuf bytes.Buffer
	binary.Write(&nonceBuf, binary.BigEndian, nonce)
	data := util.PrefixKey(parent.Bytes(), nonceBuf.Bytes())
	hash := sha3.Sum256(data)
	return loom.Address{
		ChainID: parent.ChainID,
		Local:   hash[12:],
	}
}

func (vm *PluginVM) Create(caller loom.Address, contractVersion string, code []byte, value *loom.BigUInt) ([]byte, loom.Address, error) {
	nonce := auth.Nonce(vm.State, caller)
	contractAddr := CreateAddress(caller, nonce)

	ret, err := vm.run(caller, contractAddr, contractVersion, code, nil, false)
	if err != nil {
		return nil, contractAddr, err
	}

	vm.State.Set(textKey(contractAddr), ret)
	return ret, contractAddr, nil
}

func (vm *PluginVM) Call(caller, addr loom.Address, contractVersion string, input []byte, value *loom.BigUInt) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, contractVersion, code, input, false)
}

func (vm *PluginVM) StaticCall(caller, addr loom.Address, contractVersion string, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	code := vm.State.Get(textKey(addr))
	return vm.run(caller, addr, contractVersion, code, input, true)
}

func (vm *PluginVM) CallEVM(caller, addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	var createABM levm.AccountBalanceManagerFactoryFunc
	var err error
	if vm.newABMFactory != nil {
		createABM, err = vm.newABMFactory(vm)
		if err != nil {
			return nil, err
		}
	}
	evm := levm.NewLoomVm(vm.State, vm.EventHandler, createABM)
	return evm.Call(caller, addr, "", input, value)
}

func (vm *PluginVM) StaticCallEVM(caller, addr loom.Address, input []byte) ([]byte, error) {
	var createABM levm.AccountBalanceManagerFactoryFunc
	var err error
	if vm.newABMFactory != nil {
		createABM, err = vm.newABMFactory(vm)
		if err != nil {
			return nil, err
		}
	}
	evm := levm.NewLoomVm(vm.State, vm.EventHandler, createABM)
	return evm.StaticCall(caller, addr, "", input)
}

func (vm *PluginVM) GetCode(addr loom.Address) ([]byte, error) {
	return []byte{}, nil
}

// Implements plugin.Context interface (go-loom/plugin/contract.go)
type contractContext struct {
	caller  loom.Address
	address loom.Address
	loomchain.State
	VM *PluginVM
	registry.Registry
	eventHandler loomchain.EventHandler
	readOnly     bool
	pluginName   string
	logger       *loom.Logger
	req          *Request
}

var _ lp.Context = &contractContext{}

func (c *contractContext) ValidatorPower(pubKey []byte) int64 {
	// TODO
	return 0
}

func (c *contractContext) Call(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.Call(c.address, addr, "", input, loom.NewBigUIntFromInt(0))
}

func (c *contractContext) CallEVM(addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	return c.VM.CallEVM(c.address, addr, input, value)
}

func (c *contractContext) StaticCall(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.StaticCall(c.address, addr, "", input)
}

func (c *contractContext) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.StaticCallEVM(c.address, addr, input)
}

func (c *contractContext) Resolve(name string) (loom.Address, error) {
	return c.Registry.Resolve(name, "")
}

func (c *contractContext) Message() lp.Message {
	return lp.Message{
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
	c.EmitTopics(event)
}

func (c *contractContext) EmitTopics(event []byte, topics ...string) {
	log.Debug("emitting event", "bytes", event)
	if c.readOnly {
		return
	}
	data := loomchain.EventData{
		Topics:          topics,
		Caller:          c.caller.MarshalPB(),
		Address:         c.address.MarshalPB(),
		PluginName:      c.pluginName,
		EncodedBody:     event,
		OriginalRequest: c.req.Body,
	}
	height := uint64(c.State.Block().Height)
	c.eventHandler.Post(height, &data)
}

func (c *contractContext) ContractRecord(contractAddr loom.Address) (*lp.ContractRecord, error) {
	rec, err := c.Registry.GetRecord(contractAddr)
	if err != nil {
		return nil, err
	}
	return &lp.ContractRecord{
		ContractName:    rec.Name,
		ContractAddress: loom.UnmarshalAddressPB(rec.Address),
		CreatorAddress:  loom.UnmarshalAddressPB(rec.Owner),
	}, nil
}
