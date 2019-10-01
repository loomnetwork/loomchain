package evm

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"

	"github.com/loomnetwork/loomchain"
)

func NewMappedAccountPCF(
	_state loomchain.State,
	createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error),
) *mappedAccount {
	return &mappedAccount{
		state:                  _state,
		createAddressMapperCtx: createAddressMapperCtx,
	}
}

type mappedAccount struct {
	state                  loomchain.State
	createAddressMapperCtx func(state loomchain.State) (contractpb.StaticContext, error)
}

func (ma mappedAccount) RequiredGas(input []byte) uint64 {
	return uint64(0)
}

func (ma mappedAccount) Run(input []byte) ([]byte, error) {
	/*	addr := input[:20]
		fromChain := input[20:]

		writeURI := gatewayCmdFlags.URI + "/rpc"
		readURI := gatewayCmdFlags.URI + "/query"
		rpcClient = client.NewDAppChainRPCClient(gatewayCmdFlags.ChainID, writeURI, readURI)
		rpcClient := getDAppChainClient()
		mapperAddr, err := rpcClient.Resolve("addressmapper")
		if err != nil {
			return errors.Wrap(err, "failed to resolve DAppChain Address Mapper address")
		}
		mapper := client.NewContract(rpcClient, mapperAddr.Local)
	*/
	return nil, nil
}
