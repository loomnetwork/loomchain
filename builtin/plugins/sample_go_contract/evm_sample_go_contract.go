// +build evm

package sample_go_contract

import (
	"errors"
	"io/ioutil"
	"math/big"
	"path"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/sample_go_contract"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
)

const (
	innerEmitterAbiFilename = "./testdata/InnerEmitter.abi"
	outerEmitterAbiFilename = "./testdata/OuterEmitter.abi"
)

func (k *SampleGoContract) TestNestedEvmCalls(ctx contractpb.Context, req *types.SampleGoContractNestedEvmRequest) error {
	if err := testInnerEmitter(ctx, loom.UnmarshalAddressPB(req.InnerEmitter), req.InnerEmitterValue); err != nil {
		return err
	}
	if err := testOuterEmitter(ctx, loom.UnmarshalAddressPB(req.OuterEmitter), req.OuterEmitterValue); err != nil {
		return err
	}
	return nil
}

func testInnerEmitter(ctx contractpb.Context, addr loom.Address, value uint64) error {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return errors.New("cannot find the path of evm_sample_go_contract.go")
	}
	abiPath := path.Join(path.Dir(filename), innerEmitterAbiFilename)
	abiBytes, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return err
	}
	abiEventData, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("sendEvent", big.NewInt(int64(value)))
	if err != nil {
		return err
	}
	evmOut := []byte{}
	err = contractpb.CallEVM(ctx, addr, input, &evmOut)
	if err != nil {
		return err
	}
	return nil
}

func testOuterEmitter(ctx contractpb.Context, addr loom.Address, value uint64) error {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return errors.New("cannot find the path of evm_sample_go_contract.go")
	}
	abiPath := path.Join(path.Dir(filename), outerEmitterAbiFilename)
	abiBytes, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return err
	}
	abiEventData, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return err
	}
	input, err := abiEventData.Pack("sendEvent", big.NewInt(int64(value)))
	if err != nil {
		return err
	}
	evmOut := []byte{}

	err = contractpb.CallEVM(ctx, addr, input, &evmOut)
	if err != nil {
		return err
	}
	return nil
}
