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

func (s *QueryServer) GetAccountBalances(contract []string) (*AccountsBalanceResponse, error) {
	fmt.Println(" --- GetAccountBalances called ---")
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
		return nil, errors.Errorf("Mappings Len == 0 : query %+v", contract)
	}

	mapArr := make(map[string]map[string]string, len(resp.Mappings))
	for _, mp := range resp.Mappings {
		var localAddr, foreignAddr loom.Address
		if mp.From.ChainId == s.ChainID {
			localAddr = loom.UnmarshalAddressPB(mp.From)
			foreignAddr = loom.UnmarshalAddressPB(mp.To)
		} else {
			localAddr = loom.UnmarshalAddressPB(mp.To)
			foreignAddr = loom.UnmarshalAddressPB(mp.From)
		}
		fmt.Println("local ,", localAddr.ChainID, localAddr.Local.String(), "<-> foreign ", foreignAddr.ChainID, foreignAddr.Local.String())
		mapArr2 := make(map[string]string, 0)
		for _, contractAddr := range contract {
			switch contractAddr {
			case "eth":
				ethcoinCtx, err := s.createStaticContractCtx(snapshot, "ethcoin")
				if err != nil {
					if errors.Cause(err) == registry.ErrNotFound {
						mapArr2[contractAddr] = "ethcoin contract not found"
						continue
					}
					return nil, err
				}
				ethBal, err := getEthBalance(ethcoinCtx, localAddr)
				if err != nil {
					fmt.Println("ethBal err ", err)
				}
				fmt.Println("eth bal", ethBal.String())
				mapArr2[contractAddr] = ethBal.String()

			case "loom":
				loomCoinCtx, err := s.createStaticContractCtx(snapshot, "coin")
				if err != nil {
					fmt.Println("loomCoinCtx err ", err)
				}
				loomBal, err := getLoomBalance(loomCoinCtx, localAddr)
				if err != nil {
					fmt.Println("loomBal err ", err)
				}
				mapArr2[contractAddr] = loomBal.Value.String()
			default:
				gatewayCtx, err := s.createStaticContractCtx(snapshot, "gateway")
				if err != nil {
					if errors.Cause(err) == registry.ErrNotFound {
						mapArr2[contractAddr] = "gateway contract not found"
						continue
					}
					return nil, err
				}
				addr, err := getMappedContractAddress(gatewayCtx, loom.MustParseAddress(contractAddr))
				if err != nil {
					fmt.Println("getMappedContractAddress error", err)
				}
				gwCtx := gateway.NewERC20StaticContext(gatewayCtx, loom.UnmarshalAddressPB(addr))
				erc20Bal2, err := gateway.BalanceOf(gwCtx, foreignAddr)
				if err != nil {
					return nil, err
				}
				mapArr2[contractAddr] = erc20Bal2.String()
			}
		}
		mapArr[foreignAddr.ChainID+":"+foreignAddr.Local.String()] = mapArr2
	}

	return &AccountsBalanceResponse{Accounts: mapArr}, nil
}

func getEthBalance(ctx contractpb.StaticContext, address loom.Address) (*glcommon.BigUInt, error) {
	amount, err := ethcoin.BalanceOf(ctx, address)
	if err != nil {
		return nil, err
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
		return nil, err
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

func GetTest() error {
	return nil
}
