package deployer_whitelist

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	"github.com/loomnetwork/go-loom/cli"
	"github.com/loomnetwork/go-loom/client"
	"github.com/pkg/errors"
)

func getMappedAccount(mapper *client.Contract, account loom.Address) (loom.Address, error) {
	req := &amtypes.AddressMapperGetMappingRequest{
		From: account.MarshalPB(),
	}
	resp := &amtypes.AddressMapperGetMappingResponse{}
	_, err := mapper.StaticCall("GetMapping", req, account, resp)
	if err != nil {
		return loom.Address{}, err
	}
	return loom.UnmarshalAddressPB(resp.To), nil
}

func getAddressPrefix(addr string) string {
	strs := strings.Split(addr, ":")
	if len(strs) > 0 {
		return strs[0]
	}
	return ""
}

func parseAddress(address string) (loom.Address, error) {
	var addr loom.Address
	addr, err := cli.ParseAddress(address)
	if err != nil {
		return addr, errors.Wrap(err, "failed to parse address")
	}
	//Resolve address if chainID does not match prefix
	if addr.ChainID != cli.TxFlags.ChainID {
		rpcClient := client.NewDAppChainRPCClient(cli.TxFlags.ChainID, cli.TxFlags.WriteURI, cli.TxFlags.ReadURI)
		mapperAddr, err := rpcClient.Resolve("addressmapper")
		if err != nil {
			return addr, errors.Wrap(err, "failed to resolve DAppChain Address Mapper address")
		}
		mapper := client.NewContract(rpcClient, mapperAddr.Local)
		mappedAccount, err := getMappedAccount(mapper, addr)
		if err != nil {
			return addr, fmt.Errorf("No account information found for %v", addr)
		}
		addr = mappedAccount
	}
	return addr, nil
}

func formatJSON(pb proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		Indent:       "  ",
		EmitDefaults: true,
	}
	return marshaler.MarshalToString(pb)
}
