// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gateway

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// GatewayABI is the input ABI used to generate the binding from.
const GatewayABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_validators\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"ETHReceived\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"ERC20Received\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"uid\",\"type\":\"uint256\"}],\"name\":\"ERC721Received\",\"type\":\"event\"},{\"constant\":false,\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"sig\",\"type\":\"bytes\"}],\"name\":\"withdrawERC20\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"uid\",\"type\":\"uint256\"},{\"name\":\"sig\",\"type\":\"bytes\"}],\"name\":\"withdrawERC721\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"amount\",\"type\":\"uint256\"},{\"name\":\"sig\",\"type\":\"bytes\"}],\"name\":\"withdrawETH\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"onERC20Received\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes4\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_uid\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"onERC721Received\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes4\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_erc20\",\"type\":\"address\"}],\"name\":\"setERC20\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_erc721\",\"type\":\"address\"}],\"name\":\"setERC721\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"getBalance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"owner\",\"type\":\"address\"},{\"name\":\"uid\",\"type\":\"uint256\"}],\"name\":\"getNFT\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"

// Gateway is an auto generated Go binding around an Ethereum contract.
type Gateway struct {
	GatewayCaller     // Read-only binding to the contract
	GatewayTransactor // Write-only binding to the contract
	GatewayFilterer   // Log filterer for contract events
}

// GatewayCaller is an auto generated read-only Go binding around an Ethereum contract.
type GatewayCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GatewayTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GatewayTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GatewayFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GatewayFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GatewaySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GatewaySession struct {
	Contract     *Gateway          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GatewayCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GatewayCallerSession struct {
	Contract *GatewayCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// GatewayTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GatewayTransactorSession struct {
	Contract     *GatewayTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// GatewayRaw is an auto generated low-level Go binding around an Ethereum contract.
type GatewayRaw struct {
	Contract *Gateway // Generic contract binding to access the raw methods on
}

// GatewayCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GatewayCallerRaw struct {
	Contract *GatewayCaller // Generic read-only contract binding to access the raw methods on
}

// GatewayTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GatewayTransactorRaw struct {
	Contract *GatewayTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGateway creates a new instance of Gateway, bound to a specific deployed contract.
func NewGateway(address common.Address, backend bind.ContractBackend) (*Gateway, error) {
	contract, err := bindGateway(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Gateway{GatewayCaller: GatewayCaller{contract: contract}, GatewayTransactor: GatewayTransactor{contract: contract}, GatewayFilterer: GatewayFilterer{contract: contract}}, nil
}

// NewGatewayCaller creates a new read-only instance of Gateway, bound to a specific deployed contract.
func NewGatewayCaller(address common.Address, caller bind.ContractCaller) (*GatewayCaller, error) {
	contract, err := bindGateway(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GatewayCaller{contract: contract}, nil
}

// NewGatewayTransactor creates a new write-only instance of Gateway, bound to a specific deployed contract.
func NewGatewayTransactor(address common.Address, transactor bind.ContractTransactor) (*GatewayTransactor, error) {
	contract, err := bindGateway(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GatewayTransactor{contract: contract}, nil
}

// NewGatewayFilterer creates a new log filterer instance of Gateway, bound to a specific deployed contract.
func NewGatewayFilterer(address common.Address, filterer bind.ContractFilterer) (*GatewayFilterer, error) {
	contract, err := bindGateway(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GatewayFilterer{contract: contract}, nil
}

// bindGateway binds a generic wrapper to an already deployed contract.
func bindGateway(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(GatewayABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Gateway *GatewayRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Gateway.Contract.GatewayCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Gateway *GatewayRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Gateway.Contract.GatewayTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Gateway *GatewayRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Gateway.Contract.GatewayTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Gateway *GatewayCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Gateway.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Gateway *GatewayTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Gateway.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Gateway *GatewayTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Gateway.Contract.contract.Transact(opts, method, params...)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(owner address) constant returns(uint256, uint256)
func (_Gateway *GatewayCaller) GetBalance(opts *bind.CallOpts, owner common.Address) (*big.Int, *big.Int, error) {
	var (
		ret0 = new(*big.Int)
		ret1 = new(*big.Int)
	)
	out := &[]interface{}{
		ret0,
		ret1,
	}
	err := _Gateway.contract.Call(opts, out, "getBalance", owner)
	return *ret0, *ret1, err
}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(owner address) constant returns(uint256, uint256)
func (_Gateway *GatewaySession) GetBalance(owner common.Address) (*big.Int, *big.Int, error) {
	return _Gateway.Contract.GetBalance(&_Gateway.CallOpts, owner)
}

// GetBalance is a free data retrieval call binding the contract method 0xf8b2cb4f.
//
// Solidity: function getBalance(owner address) constant returns(uint256, uint256)
func (_Gateway *GatewayCallerSession) GetBalance(owner common.Address) (*big.Int, *big.Int, error) {
	return _Gateway.Contract.GetBalance(&_Gateway.CallOpts, owner)
}

// GetNFT is a free data retrieval call binding the contract method 0x2207bdcf.
//
// Solidity: function getNFT(owner address, uid uint256) constant returns(bool)
func (_Gateway *GatewayCaller) GetNFT(opts *bind.CallOpts, owner common.Address, uid *big.Int) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Gateway.contract.Call(opts, out, "getNFT", owner, uid)
	return *ret0, err
}

// GetNFT is a free data retrieval call binding the contract method 0x2207bdcf.
//
// Solidity: function getNFT(owner address, uid uint256) constant returns(bool)
func (_Gateway *GatewaySession) GetNFT(owner common.Address, uid *big.Int) (bool, error) {
	return _Gateway.Contract.GetNFT(&_Gateway.CallOpts, owner, uid)
}

// GetNFT is a free data retrieval call binding the contract method 0x2207bdcf.
//
// Solidity: function getNFT(owner address, uid uint256) constant returns(bool)
func (_Gateway *GatewayCallerSession) GetNFT(owner common.Address, uid *big.Int) (bool, error) {
	return _Gateway.Contract.GetNFT(&_Gateway.CallOpts, owner, uid)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces( address) constant returns(uint256)
func (_Gateway *GatewayCaller) Nonces(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Gateway.contract.Call(opts, out, "nonces", arg0)
	return *ret0, err
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces( address) constant returns(uint256)
func (_Gateway *GatewaySession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _Gateway.Contract.Nonces(&_Gateway.CallOpts, arg0)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces( address) constant returns(uint256)
func (_Gateway *GatewayCallerSession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _Gateway.Contract.Nonces(&_Gateway.CallOpts, arg0)
}

// OnERC20Received is a paid mutator transaction binding the contract method 0xbc04f0af.
//
// Solidity: function onERC20Received(_from address, amount uint256) returns(bytes4)
func (_Gateway *GatewayTransactor) OnERC20Received(opts *bind.TransactOpts, _from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "onERC20Received", _from, amount)
}

// OnERC20Received is a paid mutator transaction binding the contract method 0xbc04f0af.
//
// Solidity: function onERC20Received(_from address, amount uint256) returns(bytes4)
func (_Gateway *GatewaySession) OnERC20Received(_from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Gateway.Contract.OnERC20Received(&_Gateway.TransactOpts, _from, amount)
}

// OnERC20Received is a paid mutator transaction binding the contract method 0xbc04f0af.
//
// Solidity: function onERC20Received(_from address, amount uint256) returns(bytes4)
func (_Gateway *GatewayTransactorSession) OnERC20Received(_from common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Gateway.Contract.OnERC20Received(&_Gateway.TransactOpts, _from, amount)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0xf0b9e5ba.
//
// Solidity: function onERC721Received(_from address, _uid uint256,  bytes) returns(bytes4)
func (_Gateway *GatewayTransactor) OnERC721Received(opts *bind.TransactOpts, _from common.Address, _uid *big.Int, arg2 []byte) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "onERC721Received", _from, _uid, arg2)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0xf0b9e5ba.
//
// Solidity: function onERC721Received(_from address, _uid uint256,  bytes) returns(bytes4)
func (_Gateway *GatewaySession) OnERC721Received(_from common.Address, _uid *big.Int, arg2 []byte) (*types.Transaction, error) {
	return _Gateway.Contract.OnERC721Received(&_Gateway.TransactOpts, _from, _uid, arg2)
}

// OnERC721Received is a paid mutator transaction binding the contract method 0xf0b9e5ba.
//
// Solidity: function onERC721Received(_from address, _uid uint256,  bytes) returns(bytes4)
func (_Gateway *GatewayTransactorSession) OnERC721Received(_from common.Address, _uid *big.Int, arg2 []byte) (*types.Transaction, error) {
	return _Gateway.Contract.OnERC721Received(&_Gateway.TransactOpts, _from, _uid, arg2)
}

// SetERC20 is a paid mutator transaction binding the contract method 0xc29a6fda.
//
// Solidity: function setERC20(_erc20 address) returns()
func (_Gateway *GatewayTransactor) SetERC20(opts *bind.TransactOpts, _erc20 common.Address) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "setERC20", _erc20)
}

// SetERC20 is a paid mutator transaction binding the contract method 0xc29a6fda.
//
// Solidity: function setERC20(_erc20 address) returns()
func (_Gateway *GatewaySession) SetERC20(_erc20 common.Address) (*types.Transaction, error) {
	return _Gateway.Contract.SetERC20(&_Gateway.TransactOpts, _erc20)
}

// SetERC20 is a paid mutator transaction binding the contract method 0xc29a6fda.
//
// Solidity: function setERC20(_erc20 address) returns()
func (_Gateway *GatewayTransactorSession) SetERC20(_erc20 common.Address) (*types.Transaction, error) {
	return _Gateway.Contract.SetERC20(&_Gateway.TransactOpts, _erc20)
}

// SetERC721 is a paid mutator transaction binding the contract method 0x094144a5.
//
// Solidity: function setERC721(_erc721 address) returns()
func (_Gateway *GatewayTransactor) SetERC721(opts *bind.TransactOpts, _erc721 common.Address) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "setERC721", _erc721)
}

// SetERC721 is a paid mutator transaction binding the contract method 0x094144a5.
//
// Solidity: function setERC721(_erc721 address) returns()
func (_Gateway *GatewaySession) SetERC721(_erc721 common.Address) (*types.Transaction, error) {
	return _Gateway.Contract.SetERC721(&_Gateway.TransactOpts, _erc721)
}

// SetERC721 is a paid mutator transaction binding the contract method 0x094144a5.
//
// Solidity: function setERC721(_erc721 address) returns()
func (_Gateway *GatewayTransactorSession) SetERC721(_erc721 common.Address) (*types.Transaction, error) {
	return _Gateway.Contract.SetERC721(&_Gateway.TransactOpts, _erc721)
}

// WithdrawERC20 is a paid mutator transaction binding the contract method 0x5df22139.
//
// Solidity: function withdrawERC20(amount uint256, sig bytes) returns()
func (_Gateway *GatewayTransactor) WithdrawERC20(opts *bind.TransactOpts, amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "withdrawERC20", amount, sig)
}

// WithdrawERC20 is a paid mutator transaction binding the contract method 0x5df22139.
//
// Solidity: function withdrawERC20(amount uint256, sig bytes) returns()
func (_Gateway *GatewaySession) WithdrawERC20(amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawERC20(&_Gateway.TransactOpts, amount, sig)
}

// WithdrawERC20 is a paid mutator transaction binding the contract method 0x5df22139.
//
// Solidity: function withdrawERC20(amount uint256, sig bytes) returns()
func (_Gateway *GatewayTransactorSession) WithdrawERC20(amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawERC20(&_Gateway.TransactOpts, amount, sig)
}

// WithdrawERC721 is a paid mutator transaction binding the contract method 0x50366619.
//
// Solidity: function withdrawERC721(uid uint256, sig bytes) returns()
func (_Gateway *GatewayTransactor) WithdrawERC721(opts *bind.TransactOpts, uid *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "withdrawERC721", uid, sig)
}

// WithdrawERC721 is a paid mutator transaction binding the contract method 0x50366619.
//
// Solidity: function withdrawERC721(uid uint256, sig bytes) returns()
func (_Gateway *GatewaySession) WithdrawERC721(uid *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawERC721(&_Gateway.TransactOpts, uid, sig)
}

// WithdrawERC721 is a paid mutator transaction binding the contract method 0x50366619.
//
// Solidity: function withdrawERC721(uid uint256, sig bytes) returns()
func (_Gateway *GatewayTransactorSession) WithdrawERC721(uid *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawERC721(&_Gateway.TransactOpts, uid, sig)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0x3ef32986.
//
// Solidity: function withdrawETH(amount uint256, sig bytes) returns()
func (_Gateway *GatewayTransactor) WithdrawETH(opts *bind.TransactOpts, amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.contract.Transact(opts, "withdrawETH", amount, sig)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0x3ef32986.
//
// Solidity: function withdrawETH(amount uint256, sig bytes) returns()
func (_Gateway *GatewaySession) WithdrawETH(amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawETH(&_Gateway.TransactOpts, amount, sig)
}

// WithdrawETH is a paid mutator transaction binding the contract method 0x3ef32986.
//
// Solidity: function withdrawETH(amount uint256, sig bytes) returns()
func (_Gateway *GatewayTransactorSession) WithdrawETH(amount *big.Int, sig []byte) (*types.Transaction, error) {
	return _Gateway.Contract.WithdrawETH(&_Gateway.TransactOpts, amount, sig)
}

// GatewayERC20ReceivedIterator is returned from FilterERC20Received and is used to iterate over the raw logs and unpacked data for ERC20Received events raised by the Gateway contract.
type GatewayERC20ReceivedIterator struct {
	Event *GatewayERC20Received // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GatewayERC20ReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GatewayERC20Received)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GatewayERC20Received)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GatewayERC20ReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GatewayERC20ReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GatewayERC20Received represents a ERC20Received event raised by the Gateway contract.
type GatewayERC20Received struct {
	From   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterERC20Received is a free log retrieval operation binding the contract event 0x7fd6e121e69ec2842a4a29502be0a852f8de98e1b672cadf23dc1786ee3945e6.
//
// Solidity: e ERC20Received(_from address, amount uint256)
func (_Gateway *GatewayFilterer) FilterERC20Received(opts *bind.FilterOpts) (*GatewayERC20ReceivedIterator, error) {

	logs, sub, err := _Gateway.contract.FilterLogs(opts, "ERC20Received")
	if err != nil {
		return nil, err
	}
	return &GatewayERC20ReceivedIterator{contract: _Gateway.contract, event: "ERC20Received", logs: logs, sub: sub}, nil
}

// WatchERC20Received is a free log subscription operation binding the contract event 0x7fd6e121e69ec2842a4a29502be0a852f8de98e1b672cadf23dc1786ee3945e6.
//
// Solidity: e ERC20Received(_from address, amount uint256)
func (_Gateway *GatewayFilterer) WatchERC20Received(opts *bind.WatchOpts, sink chan<- *GatewayERC20Received) (event.Subscription, error) {

	logs, sub, err := _Gateway.contract.WatchLogs(opts, "ERC20Received")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GatewayERC20Received)
				if err := _Gateway.contract.UnpackLog(event, "ERC20Received", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// GatewayERC721ReceivedIterator is returned from FilterERC721Received and is used to iterate over the raw logs and unpacked data for ERC721Received events raised by the Gateway contract.
type GatewayERC721ReceivedIterator struct {
	Event *GatewayERC721Received // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GatewayERC721ReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GatewayERC721Received)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GatewayERC721Received)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GatewayERC721ReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GatewayERC721ReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GatewayERC721Received represents a ERC721Received event raised by the Gateway contract.
type GatewayERC721Received struct {
	From common.Address
	Uid  *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterERC721Received is a free log retrieval operation binding the contract event 0x35a641d6803b18b3c2a97b78c27d31dab914e9626b63b48fb9c5747c93a3f96d.
//
// Solidity: e ERC721Received(_from address, uid uint256)
func (_Gateway *GatewayFilterer) FilterERC721Received(opts *bind.FilterOpts) (*GatewayERC721ReceivedIterator, error) {

	logs, sub, err := _Gateway.contract.FilterLogs(opts, "ERC721Received")
	if err != nil {
		return nil, err
	}
	return &GatewayERC721ReceivedIterator{contract: _Gateway.contract, event: "ERC721Received", logs: logs, sub: sub}, nil
}

// WatchERC721Received is a free log subscription operation binding the contract event 0x35a641d6803b18b3c2a97b78c27d31dab914e9626b63b48fb9c5747c93a3f96d.
//
// Solidity: e ERC721Received(_from address, uid uint256)
func (_Gateway *GatewayFilterer) WatchERC721Received(opts *bind.WatchOpts, sink chan<- *GatewayERC721Received) (event.Subscription, error) {

	logs, sub, err := _Gateway.contract.WatchLogs(opts, "ERC721Received")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GatewayERC721Received)
				if err := _Gateway.contract.UnpackLog(event, "ERC721Received", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// GatewayETHReceivedIterator is returned from FilterETHReceived and is used to iterate over the raw logs and unpacked data for ETHReceived events raised by the Gateway contract.
type GatewayETHReceivedIterator struct {
	Event *GatewayETHReceived // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GatewayETHReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GatewayETHReceived)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GatewayETHReceived)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GatewayETHReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GatewayETHReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GatewayETHReceived represents a ETHReceived event raised by the Gateway contract.
type GatewayETHReceived struct {
	From   common.Address
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterETHReceived is a free log retrieval operation binding the contract event 0xbfe611b001dfcd411432f7bf0d79b82b4b2ee81511edac123a3403c357fb972a.
//
// Solidity: e ETHReceived(_from address, amount uint256)
func (_Gateway *GatewayFilterer) FilterETHReceived(opts *bind.FilterOpts) (*GatewayETHReceivedIterator, error) {

	logs, sub, err := _Gateway.contract.FilterLogs(opts, "ETHReceived")
	if err != nil {
		return nil, err
	}
	return &GatewayETHReceivedIterator{contract: _Gateway.contract, event: "ETHReceived", logs: logs, sub: sub}, nil
}

// WatchETHReceived is a free log subscription operation binding the contract event 0xbfe611b001dfcd411432f7bf0d79b82b4b2ee81511edac123a3403c357fb972a.
//
// Solidity: e ETHReceived(_from address, amount uint256)
func (_Gateway *GatewayFilterer) WatchETHReceived(opts *bind.WatchOpts, sink chan<- *GatewayETHReceived) (event.Subscription, error) {

	logs, sub, err := _Gateway.contract.WatchLogs(opts, "ETHReceived")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GatewayETHReceived)
				if err := _Gateway.contract.UnpackLog(event, "ETHReceived", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
