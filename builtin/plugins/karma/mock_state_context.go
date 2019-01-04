package karma

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"

	"github.com/loomnetwork/loomchain"
)

type FEvent struct {
	Event  []byte
	Topics []string
}

type FakeStateContext struct {
	caller        loom.Address
	address       loom.Address
	block         loom.BlockHeader
	state 	      loomchain.State
	contractNonce uint64
	contracts     map[string]plugin.Contract
	registry      map[string]*plugin.ContractRecord
	validators    loom.ValidatorSet
	Events        []FEvent
	ethBalances   map[string]*loom.BigUInt
}

var _ plugin.Context = &FakeStateContext{}

func createAddress(parent loom.Address, nonce uint64) loom.Address {
	var nonceBuf bytes.Buffer
	_ = binary.Write(&nonceBuf, binary.BigEndian, nonce)
	data := util.PrefixKey(parent.Bytes(), nonceBuf.Bytes())
	hash := sha3.Sum256(data)
	return loom.Address{
		ChainID: parent.ChainID,
		Local:   hash[12:],
	}
}

func CreateFakeStateContext(state loomchain.State, caller, address loom.Address) *FakeStateContext {
	return &FakeStateContext{
		caller:      caller,
		address:     address,
		state:       loomchain.StateWithPrefix(loom.DataPrefix(address), state),
		contracts:   make(map[string]plugin.Contract),
		registry:    make(map[string]*plugin.ContractRecord),
		validators:  loom.NewValidatorSet(),
		Events:      make([]FEvent, 0),
		ethBalances: make(map[string]*loom.BigUInt),
	}
}

func (c *FakeStateContext) shallowClone() *FakeStateContext {
	return &FakeStateContext{
		caller:        c.caller,
		address:       c.address,
		block:         c.block,
		state:         c.state,
		contractNonce: c.contractNonce,
		contracts:     c.contracts,
		registry:      c.registry,
		validators:    c.validators,
		Events:        c.Events,
		ethBalances:   c.ethBalances,
	}
}

func (c *FakeStateContext) WithBlock(header loom.BlockHeader) *FakeStateContext {
	clone := c.shallowClone()
	clone.block = header
	return clone
}

func (c *FakeStateContext) WithSender(caller loom.Address) *FakeStateContext {
	clone := c.shallowClone()
	clone.caller = caller
	return clone
}

func (c *FakeStateContext) WithAddress(addr loom.Address) *FakeStateContext {
	clone := c.shallowClone()
	clone.address = addr
	return clone
}

func (c *FakeStateContext) CreateContract(contract plugin.Contract) loom.Address {
	addr := createAddress(c.address, c.contractNonce)
	c.contractNonce++
	c.contracts[addr.String()] = contract
	return addr
}

func (c *FakeStateContext) RegisterContract(contractName string, contractAddr, creatorAddr loom.Address) {
	c.registry[contractAddr.String()] = &plugin.ContractRecord{
		ContractName:    contractName,
		ContractAddress: contractAddr,
		CreatorAddress:  creatorAddr,
	}
}

func (c *FakeStateContext) Call(addr loom.Address, input []byte) ([]byte, error) {
	contract := c.contracts[addr.String()]

	ctx := c.WithSender(c.address).WithAddress(addr)

	var req plugin.Request
	err := proto.Unmarshal(input, &req)
	if err != nil {
		return nil, err
	}

	resp, err := contract.Call(ctx, &req)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(resp)
}

func (c *FakeStateContext) StaticCall(addr loom.Address, input []byte) ([]byte, error) {
	contract := c.contracts[addr.String()]

	ctx := c.WithSender(c.address).WithAddress(addr)

	var req plugin.Request
	err := proto.Unmarshal(input, &req)
	if err != nil {
		return nil, err
	}

	resp, err := contract.StaticCall(ctx, &req)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(resp)
}

func (c *FakeStateContext) CallEVM(addr loom.Address, input []byte, value *loom.BigUInt) ([]byte, error) {
	return nil, nil
}

func (c *FakeStateContext) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	return nil, nil
}

func (c *FakeStateContext) Resolve(name string) (loom.Address, error) {
	for addrStr, contract := range c.contracts {
		meta, err := contract.Meta()
		if err != nil {
			return loom.Address{}, err
		}
		if meta.Name == name {
			return loom.MustParseAddress(addrStr), nil
		}
	}
	return loom.Address{}, fmt.Errorf("failed  to resolve address of contract '%s'", name)
}

func (c *FakeStateContext) ValidatorPower(pubKey []byte) int64 {
	return 0
}

func (c *FakeStateContext) Message() plugin.Message {
	return plugin.Message{
		Sender: c.caller,
	}
}

func (c *FakeStateContext) Block() types.BlockHeader {
	return c.block
}

func (c *FakeStateContext) ContractAddress() loom.Address {
	return c.address
}

func (c *FakeStateContext) GetEvmTxReceipt([]byte) (ptypes.EvmTxReceipt, error) {
	return ptypes.EvmTxReceipt{}, nil
}

func (c *FakeStateContext) SetTime(t time.Time) {
	c.block.Time = t.Unix()
}

func (c *FakeStateContext) Now() time.Time {
	return time.Unix(c.block.Time, 0)
}

func (c *FakeStateContext) EmitTopics(event []byte, topics ...string) {
	//Store last emitted strings, to make it testable
	c.Events = append(c.Events, FEvent{event, topics})
}

func (c *FakeStateContext) Emit(event []byte) {
}

// Prefix the given key with the contract address
func (c *FakeStateContext) makeKey(key []byte) string {
	return string(util.PrefixKey(c.address.Bytes(), key))
}

// Strip the contract address from the given key (i.e. inverse of makeKey)
func (c *FakeStateContext) recoverKey(key string, prefix []byte) ([]byte, error) {
	return util.UnprefixKey([]byte(key), util.PrefixKey(c.address.Bytes(), prefix))
}

func (c *FakeStateContext) Range(prefix []byte) plugin.RangeData {
	return c.state.Range(prefix)
}

func (c *FakeStateContext) Get(key []byte) []byte {
	return c.state.Get(key)
}

func (c *FakeStateContext) Has(key []byte) bool {
	return c.state.Has(key)
}

func (c *FakeStateContext) Set(key []byte, value []byte) {
	c.state.Set(key, value)
}

func (c *FakeStateContext) Delete(key []byte) {
	c.state.Delete(key)
}

func (c *FakeStateContext) SetValidatorPower(pubKey []byte, power int64) {
	c.validators.Set(&loom.Validator{PubKey: pubKey, Power: power})
}

func (c *FakeStateContext) Validators() []*loom.Validator {
	return c.validators.Slice()
}

func (c *FakeStateContext) ContractRecord(contractAddr loom.Address) (*plugin.ContractRecord, error) {
	rec := c.registry[contractAddr.String()]
	if rec == nil {
		return nil, errors.New("contract not found in registry")
	}
	return rec, nil
}
