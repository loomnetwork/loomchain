package plugin

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	proto "github.com/gogo/protobuf/proto"
	"golang.org/x/crypto/sha3"

	loom "github.com/loomnetwork/go-loom"
	lp "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/vm"
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

func permPrefix(addr loom.Address) []byte {
	return util.PrefixKey(contractPrefix(addr), []byte("permission"))
}

type PluginVM struct {
	Loader       Loader
	State        loomchain.State
	Registry     registry.Registry
	EventHandler loomchain.EventHandler
	logger       *log.Logger
}

func NewPluginVM(
	loader Loader,
	state loomchain.State,
	registry registry.Registry,
	eventHandler loomchain.EventHandler,
	logLevel string,
) *PluginVM {
	return &PluginVM{
		Loader:       loader,
		State:        state,
		Registry:     registry,
		EventHandler: eventHandler,
		logger:       log.NewFilter(log.Default.Logger, log.AllowDebug()),
	}
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

	isInit := len(input) == 0
	if isInit {
		input = pluginCode.Input
	}

	req := &Request{}
	err = proto.Unmarshal(input, req)
	if err != nil {
		return nil, err
	}

	contractCtx := &contractContext{
		caller:       caller,
		address:      addr,
		State:        loomchain.StateWithPrefix(dataPrefix(addr), vm.State),
		VM:           vm,
		Registry:     vm.Registry,
		eventHandler: vm.EventHandler,
		readOnly:     readOnly,
		pluginName:   pluginCode.Name,
		logger:       vm.logger,
		req:          req,
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

func createAddress(parent loom.Address, nonce uint64) loom.Address {
	var nonceBuf bytes.Buffer
	binary.Write(&nonceBuf, binary.BigEndian, nonce)
	data := util.PrefixKey(parent.Bytes(), nonceBuf.Bytes())
	hash := sha3.Sum256(data)
	return loom.Address{
		ChainID: parent.ChainID,
		Local:   hash[12:],
	}
}

func (vm *PluginVM) Create(caller loom.Address, code []byte) ([]byte, loom.Address, error) {
	nonce := auth.Nonce(vm.State, caller)
	contractAddr := createAddress(caller, nonce)

	ret, err := vm.run(caller, contractAddr, code, nil, false)
	if err != nil {
		return nil, contractAddr, err
	}

	vm.State.Set(textKey(contractAddr), ret)
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
	loomchain.State
	vm.VM
	registry.Registry
	eventHandler loomchain.EventHandler
	readOnly     bool
	pluginName   string
	logger       *log.Logger
	req          *Request
}

var _ lp.Context = &contractContext{}

func (c *contractContext) ValidatorPower(pubKey []byte) int64 {
	// TODO
	return 0
}

func (c *contractContext) Call(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.Call(c.address, addr, input)
}

func (c *contractContext) CallEVM(addr loom.Address, input []byte) ([]byte, error) {
	evm := vm.NewLoomVm(c.VM.(*PluginVM).State, c.eventHandler)
	return evm.Call(c.caller, addr, input)
}

func (c *contractContext) StaticCall(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.StaticCall(c.address, addr, input)
}

func (c *contractContext) Resolve(name string) (loom.Address, error) {
	return c.Registry.Resolve(name)
}

func (c *contractContext) Message() lp.Message {
	return lp.Message{
		Sender: c.caller,
	}
}

func (c *contractContext) ContractAddress() loom.Address {
	return c.address
}

// HasPermission checks whether the sender of the tx has any of the permission given in `roles` on `token`
func (c *contractContext) HasPermission(token []byte, roles []string) (bool, []string) {
	addr := c.Message().Sender
	return HasPermissionFor(addr, token, roles)
}

// HasPermissionFor checks whether the given `addr` has any of the permission given in `roles` on `token`
func (c *contractContext) HasPermissionFor(addr loom.Address, token []byte, roles []string) (bool, []string) {
	found := false
	foundRoles := []string{}
	for _, role := range roles {
		v := c.Get(c.rolePermKey(addr, token, role))
		if v != nil && string(v) == role {
			found = true
			foundRoles = append(foundRoles, role)
		}
	}
	return found, foundRoles
}

// GrantPermissionTo sets a given `role` permission on `token` for the given `addr`
func (c *contractContext) GrantPermissionTo(addr loom.Address, token []byte, role string) {
	c.Set(c.rolePermKey(addr, token, role), []byte("true"))
}

func (c *contractContext) rolePermKey(addr loom.Address, token []byte, role string) []byte {
	return []byte(fmt.Sprintf("%stoken:%s:role:%s", permPrefix(addr), token, []byte(role)))
}

// GrantPermission sets a given `role` permission on `token` for the sender of the tx
func (c *contractContext) GrantPermission(token []byte, roles []string) {
	for _, r := range roles {
		c.GrantPermissionTo(c.Message().Sender, token, r)
	}
}

func (c *contractContext) Now() time.Time {
	return time.Unix(c.State.Block().Time, 0)
}

type emitData struct {
	Caller     loom.Address `json:"caller"`
	Address    loom.Address `json:"address"`
	PluginName string       `json:"plugin"`
	Data       []byte       `json:"encodedData"`
	RawRequest []byte       `json:"rawRequest"`
}

func (c *contractContext) Emit(event []byte) {
	c.logger.Debug("emitting event", "bytes", event)
	if c.readOnly {
		return
	}
	data := emitData{
		Caller:     c.caller,
		Address:    c.address,
		PluginName: c.pluginName,
		Data:       event,
		RawRequest: c.req.Body,
	}
	emitMsg, err := json.Marshal(&data)
	if err != nil {
		c.logger.Error("Error in event marshalling for event: %s", string(event))
	}
	c.eventHandler.Post(c.State, emitMsg)
}
