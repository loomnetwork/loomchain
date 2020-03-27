// +build gateway

package rpc

import (
	"fmt"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"

	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	tgtypes "github.com/loomnetwork/go-loom/builtin/types/transfer_gateway"
	glcommon "github.com/loomnetwork/go-loom/common"
	gtypes "github.com/loomnetwork/go-loom/types"
	am "github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	loomcoin "github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/transfer-gateway/builtin/plugins/gateway"
)

func (s *QueryServer) GetAccountBalances(contracts []string) (*AccountsBalanceResponse, error) {
	snapshot := s.StateProvider.ReadOnlyState()
	defer snapshot.Release()

	addrMapperCtx, err := s.createAddressMapperCtx(s.StateProvider.ReadOnlyState())
	if err != nil {
		return nil, err
	}
	var resp *amtypes.AddressMapperListMappingResponse
	resp, err = am.ListMapping(addrMapperCtx)
	if err != nil {
		return nil, err
	}

	if len(resp.Mappings) == 0 {
		return nil, errors.Errorf("Empty address mapping list")
	}

	ethcoinCtx, err := s.createStaticContractCtx(snapshot, "ethcoin")
	if err != nil {
		if errors.Cause(err) == registry.ErrNotFound {
			account[contractAddr] = "ethcoin contract not found"
			continue
		}
		return nil, err
	}

	loomCoinCtx, err := s.createStaticContractCtx(snapshot, "coin")
	if err != nil {
		if errors.Cause(err) == registry.ErrNotFound {
			account[contractAddr] = "coin contract not found"
			continue
		}
		return nil, err
	}

	gatewayCtx, err := s.createStaticContractCtx(snapshot, "gateway")
	if err != nil {
		if errors.Cause(err) == registry.ErrNotFound {
			account[contractAddr] = "gateway contract not found"
			continue
		}
		return nil, err
	}

	mapContractContext := make(map[string]contractpb.StaticContext, len(contractAddr))
	for _, contract := range contracts {
		if contract == "eth" || contract == "loom" {
			continue
		}
		addr, _ := getMappedContractAddress(gatewayCtx, loom.MustParseAddress(contract))
		if err != nil {
			account[contract] = errors.Wrap(err, "Invalid contract address").Error()
			continue
		}
		mapContractContext[contract] = gateway.NewERC20StaticContext(gatewayCtx, loom.UnmarshalAddressPB(addr))
	}

	accounts := make(map[string]map[string]string, len(resp.Mappings))
	for _, mp := range resp.Mappings {
		var localAddr, foreignAddr loom.Address
		if mp.From.ChainId == s.ChainID {
			localAddr = loom.UnmarshalAddressPB(mp.From)
			foreignAddr = loom.UnmarshalAddressPB(mp.To)
		} else {
			localAddr = loom.UnmarshalAddressPB(mp.To)
			foreignAddr = loom.UnmarshalAddressPB(mp.From)
		}
		account := make(map[string]string, 0)
		for _, contract := range contracts {
			switch contract {
			case "eth":
				ethBal, err := getEthBalance(ethcoinCtx, localAddr)
				if err != nil {
					account[contract] = errors.Wrapf(err, "error getting balance of %s", localAddr.Local.String())
					continue
				}
				account[contract] = ethBal.String()
			case "loom":
				loomBal, err := getLoomBalance(loomCoinCtx, localAddr)
				if err != nil {
					account[contract] = errors.Wrapf(err, "error getting balance of %s", localAddr.Local.String())
					continue
				}
				account[contract] = loomBal.Value.String()
			default:
				erc20Bal, err := gateway.BalanceOf(mapContractContext[contract], localAddr)
				if err != nil {
					return nil, err
				}
				account[contract] = erc20Bal.String()
			}
		}
		accounts[foreignAddr.ChainID+":"+foreignAddr.Local.String()] = account
	}

	return &AccountsBalanceResponse{Accounts: accounts}, nil
}

func getEthBalance(ctx contractpb.StaticContext, address loom.Address) (*glcommon.BigUInt, error) {
	amount, err := ethcoin.BalanceOf(ctx, address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting balance of %s", address.Local.String())
	}
	if amount == nil {
		return glcommon.BigZero(), nil
	}
	fmt.Println("getEthBalance called amount = ", amount.String())
	return amount, nil
}

func getLoomBalance(ctx contractpb.StaticContext, address loom.Address) (*gtypes.BigUInt, error) {
	amount, err := loomcoin.BalanceOf(ctx, address)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting balance of %s", address.Local.String())
	}
	if amount == nil {
		return &gtypes.BigUInt{
			Value: *glcommon.BigZero(),
		}, nil
	}
	return &gtypes.BigUInt{Value: *amount}, nil
}

func getMappedContractAddress(ctx contractpb.StaticContext, address loom.Address) (*gtypes.Address, error) {
	var resp *tgtypes.TransferGatewayGetContractMappingResponse
	resp, err := gateway.GetContractMapping(ctx, &tgtypes.TransferGatewayGetContractMappingRequest{From: address.MarshalPB()})
	if err != nil {
		return nil, err
	}
	return resp.MappedAddress, nil
}
