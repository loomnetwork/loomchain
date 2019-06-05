// +build evm

package rpc

import (
	"bytes"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain/evm/utils"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/rpc/eth"
)

type ethTestTx struct {
	tx *types.Transaction
}

var (
	bigZero    = big.NewInt(0)
	ethTestTxs = []ethTestTx{
		{
			types.NewTransaction(
				7,
				common.HexToAddress("0xb16a379ec18d4093666f8f38b11a3071c920207d"),
				big.NewInt(11),
				0,
				bigZero,
				[]byte("input parameters"),
			),
		},
		{
			types.NewContractCreation(
				9,
				big.NewInt(24),
				0,
				bigZero,
				[]byte("contract bytecode"),
			),
		},
		{
			types.NewContractCreation(
				1,
				bigZero,
				0,
				bigZero,
				[]byte("6060604052341561000f57600080fd5b60d38061001d6000396000f3006060604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c14606e575b600080fd5b3415605857600080fd5b606c60048080359060200190919050506094565b005b3415607857600080fd5b607e609e565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058202b229fba38c096f9c9c81ba2633fb4a7b418032de7862b60d1509a4054e2d6bb0029"),
			),
		},
	}
)

func TestTendermintPRCFunc(t *testing.T) {
	log.Setup("debug", "file://-")
	testlog = log.Root.With("module", "query-server")

	qs := &MockQueryService{}
	mt := &MockTendermintRpc{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil, mt)
	chainConfig := utils.DefaultChainConfig()
	signer := types.MakeSigner(&chainConfig, blockNumber)
	for _, testTx := range ethTestTxs {
		ethKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		tx, err := types.SignTx(testTx.tx, signer, ethKey)

		local, err := types.Sender(signer, tx)
		require.NoError(t, err)
		inData, err := tx.MarshalJSON()

		payload := `{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["` + eth.EncBytes(inData) + `"],"id":99}`
		req := httptest.NewRequest("POST", "http://localhost/eth", strings.NewReader(string(payload)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Result().StatusCode)

		require.NoError(t, err)
		require.Equal(t, 0, bytes.Compare(local.Bytes(), mt.txs[0].From.Bytes()))
		compareTxs(t, testTx.tx, mt.txs[0].Tx)
	}
}

func compareTxs(t *testing.T, tx1, tx2 *types.Transaction) {
	require.Equal(t, tx1.Nonce(), tx2.Nonce())
	require.Equal(t, 0, bytes.Compare(tx1.Data(), tx2.Data()))
	if tx1.To() == nil {
		require.Nil(t, tx2.To())
	} else {
		require.Equal(t, 0, bytes.Compare(tx1.To().Bytes(), tx2.To().Bytes()))
	}
	require.Equal(t, 0, tx1.Value().Cmp(tx2.Value()))
}

/**/
