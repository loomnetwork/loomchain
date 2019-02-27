package auth

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
	"github.com/pkg/errors"
)

func GetSignatureTxMiddleware(
	createAddressMappingCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (res loomchain.TxHandlerResult, err error) {
		var r loomchain.TxHandlerResult

		var tx SignedTx
		err = proto.Unmarshal(txBytes, &tx)
		if err != nil {
			return r, err
		}

		var localAddr loom.LocalAddress
		switch tx.ChainId {
		case auth.ChainId_EOS:
			localAddr, err = verifyEos(tx)
		case auth.ChainId_ETH:
			localAddr, err = verifyEth(tx)
		case auth.ChainId_Tron:
			localAddr, err = verifyTron(tx)
		case auth.ChainId_Cosmos:
			localAddr, err = verifyCosmos(tx)
		}
		if err != nil {
			return r, err
		}

		addressMappingCtx, err := createAddressMappingCtx(state)

		if int(tx.ChainId) >= len(auth.ChainId_name) {
			return r, fmt.Errorf("invalid chain id %v", tx.ChainId)
		}
		origin, err := GetMappedOrigin(localAddr, auth.ChainId_name[int32(tx.ChainId)], state.Block().ChainID, addressMappingCtx)
		if err != nil {
			return r, err
		}

		ctx := context.WithValue(state.Context(), ContextKeyOrigin, origin)
		return next(state.WithContext(ctx), tx.Inner, isCheckTx)
	})
}

func findMapping(ctx contractpb.Context, addrAlias loom.Address) (loom.Address, error) {
	am := address_mapper.AddressMapper{}
	resp, err := am.GetMapping(ctx, &address_mapper.GetMappingRequest{
		From: addrAlias.MarshalPB(),
	})
	if err != nil {
		return loom.Address{}, err
	}

	return loom.UnmarshalAddressPB(resp.To), fmt.Errorf("not implemented")
}

func verifyEos(tx SignedTx) (loom.LocalAddress, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func verifyEth(tx SignedTx) (loom.LocalAddress, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func verifyTron(tx SignedTx) (loom.LocalAddress, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func verifyCosmos(tx SignedTx) (loom.LocalAddress, error) {
	return loom.LocalAddressFromPublicKeyV2(tx.PublicKey), fmt.Errorf("not implemented")
}

func GetMappedOrigin(localAlias loom.LocalAddress, txChainId, appChainId string, ctx contractpb.Context) (loom.Address, error) {
	if appChainId == txChainId {
		return loom.Address{
			ChainID: appChainId,
			Local:   localAlias,
		}, nil
	}

	origin, err := findMapping(ctx, loom.Address{
		ChainID: txChainId,
		Local:   localAlias,
	})
	if err != nil {
		return loom.Address{}, errors.Wrapf(err, "find mapped address for %v", localAlias.String())
	}
	if origin.ChainID != appChainId {
		return loom.Address{}, fmt.Errorf("mapped address %v from wrong chain %v", origin.String(), appChainId)
	}

	return origin, nil
}
