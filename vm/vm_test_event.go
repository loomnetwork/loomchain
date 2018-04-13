package vm

/*
func testEvents(t *testing.T, vm VM) {
	//loomState := mockState()
	testEventData := GetFiddleContractData("./testdata/TestEvent.json")
	testEventAddr := deployContract(t, vm, caller, testEventData.Bytecode, testEventData.RuntimeBytecode)
	sendEventTest(t, vm, testEventAddr, testEventData, 1375)
	sendEventTest(t, vm, testEventAddr, testEventData, 12)
	sendEventTest(t, vm, testEventAddr, testEventData, 4445)
	sendEventTest(t, vm, testEventAddr, testEventData, 8888888)

	//protoEvents := &Events{}
	//evmStore :=  NewEvmStore(vm)
	//eventsBytes, _ := evmStore.Get( eventKey)
	//err := proto.Unmarshal(eventsBytes, protoEvents)
	//require.Nil(t, err)
}

func sendEventTest(t *testing.T, vm VM, contractAddr []byte, data FiddleContractData, value uint) ([]byte) {
	//var res loom.TxHandlerResult
	abiKitty, err := abi.JSON(strings.NewReader(data.Iterface))
	if err != nil {
		t.Error("could not read kitty interface ",err)
		return []byte{}
	}
	inParams, err := abiKitty.Pack("sendEvent", big.NewInt(int64(value)))
	inParams = inParams

	//transferTx := &SendTx{
	//	Address: contractAddr,
	//	Input: inParams,
	//}
	//transferTxB, err := proto.Marshal(transferTx)
	//require.Nil(t, err)

	//res, err = ProcessSendTx(loomState, transferTxB)
	//result := UnmarshalTxReturn(res)

	//vm.Call()
	//if !checkEqual(result, nil) {
	//	t.Error("sendEvent should not return a value")
	//}
	return []byte{}
}*/