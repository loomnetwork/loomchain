// +build evm

package throttle

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain/evm/utils"
)

func isEthDeploy(txBytes []byte) (bool, error) {
	var tx types.Transaction
	if err := rlp.DecodeBytes(txBytes, &tx); err != nil {
		return false, errors.Wrap(err, "decoding ethereum transaction")
	}
	return tx.To() == nil, nil
}

// used for unit tests
func ethTxBytes(sequence uint64, to loom.Address, data []byte) ([]byte, error) {
	bigZero := big.NewInt(0)
	var tx *types.Transaction
	if to.IsEmpty() {
		tx = types.NewContractCreation(
			sequence,
			big.NewInt(24),
			0,
			bigZero,
			data,
		)
	} else {
		tx = types.NewTransaction(
			sequence,
			common.BytesToAddress(to.Local),
			big.NewInt(11),
			0,
			bigZero,
			data,
		)
	}
	chainConfig := utils.DefaultChainConfig()
	signer := types.MakeSigner(&chainConfig, chainConfig.EIP155Block)
	ethKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	tx, err = types.SignTx(tx, signer, ethKey)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(&tx)
}
