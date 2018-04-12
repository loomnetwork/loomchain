package vm

import (
	"testing"
	"github.com/loomnetwork/loom"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
	"math/big"
	"github.com/stretchr/testify/require"
	"github.com/gogo/protobuf/proto"
)

func testEvents(t *testing.T) {
	loomState := mockState()
	testEventData := GetFiddleContractData("./testdata/TestEvent.json")
	testEventAddr := deployContract(t, loomState, testEventData.Bytecode, testEventData.RuntimeBytecode)
	sendEventTest(t, loomState, testEventAddr, testEventData, 1375)
	sendEventTest(t, loomState, testEventAddr, testEventData, 12)
	sendEventTest(t, loomState, testEventAddr, testEventData, 4445)
	sendEventTest(t, loomState, testEventAddr, testEventData, 8888888)

	protoEvents := &Events{}
	evmStore :=  NewEvmStore(loomState)
	eventsBytes, _ := evmStore.Get( eventKey)
	err := proto.Unmarshal(eventsBytes, protoEvents)
	require.Nil(t, err)
}

func sendEventTest(t *testing.T, loomState loom.State, contractAddr []byte, data FiddleContractData, value uint) ([]byte) {
	var res loom.TxHandlerResult
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ",err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("sendEvent", big.NewInt(int64(value)))
	transferTx := &SendTx{
		Address: contractAddr,
		Input: inParams,
	}
	transferTxB, err := proto.Marshal(transferTx)
	require.Nil(t, err)

	res, err = ProcessSendTx(loomState, transferTxB)
	result := UnmarshalTxReturn(res)
	if !checkEqual(result, nil) {
		t.Error("sendEvent should not return a value")
	}
	return result
}