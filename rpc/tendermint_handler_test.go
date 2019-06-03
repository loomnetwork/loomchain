// +build evm

package rpc

import (
	"bytes"
	"fmt"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"

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
	}
)

func TestTendermintPRCFunc(t *testing.T) {
	log.Setup("debug", "file://-")
	testlog = log.Root.With("module", "query-server")

	qs := &MockQueryService{}
	mt := &MockTendermintRpc{}
	handler := MakeEthQueryServiceHandler(qs, testlog, nil, mt)

	signer := types.MakeSigner(DefaultChainConfig(), blockNumber)
	for _, testTx := range ethTestTxs {
		ethKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		tx, err := types.SignTx(testTx.tx, signer, ethKey)

		local, err := types.Sender(signer, tx)
		require.NoError(t, err)
		fmt.Println("top sender", local.String())

		inData, err := tx.MarshalJSON()

		payload := `{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["` + eth.EncBytes(inData) + `"],"id":99}`
		fmt.Println("payload", payload)
		req := httptest.NewRequest("POST", "http://localhost/eth", strings.NewReader(string(payload)))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Result().StatusCode)

		require.NoError(t, err)
		fmt.Println("from mt sender", mt.txs[0].From.String())
		require.Equal(t, 0, bytes.Compare(local.Bytes(), mt.txs[0].From.Bytes()))
		compareTxs(t, testTx.tx, mt.txs[0].Tx)
	}
}

func compareTxs(t *testing.T, tx1, tx2 *types.Transaction) {
	require.Equal(t, tx1.Nonce(), tx2.Nonce())
	require.Equal(t, 0, bytes.Compare(tx1.Data(), tx2.Data()))
	require.Equal(t, 0, bytes.Compare(tx1.To().Bytes(), tx2.To().Bytes()))
	require.Equal(t, 0, tx1.Value().Cmp(tx2.Value()))
}

/**/
