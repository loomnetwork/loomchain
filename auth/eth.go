// +build evm

package auth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	sha3 "github.com/miguelmota/go-solidity-sha3"
	"github.com/pkg/errors"
)

func verifySolidity66Byte(tx SignedTx) ([]byte, error) {
	from, to, nonce, err := getFromToNonce(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "retriving hash data from tx %v", tx)
	}
	hash := sha3.SoliditySHA3(
		sha3.Address(common.BytesToAddress(from.Local)),
		sha3.Address(common.BytesToAddress(to.Local)),
		sha3.Uint64(nonce),
		tx.Inner,
	)

	ethAddr, err := evmcompat.RecoverAddressFromTypedSig(hash, tx.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "verify solidity key")
	}

	return ethAddr.Bytes(), nil
}
