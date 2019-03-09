// +build evm

package gateway

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/loomchain/gateway/ethcontract"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
)

func newTestLoomCoinOracle(t *testing.T) *Oracle {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	address := loom.LocalAddressFromPublicKey(pub[:])
	signer := auth.NewEd25519Signer(priv[:])
	mainnetGatewayAddr := common.HexToAddress("0x8f8E8b3C4De76A31971Fe6a87297D8f703bE8570")
	orc := &Oracle{
		chainID: "default",
		logger:  loom.NewLoomLogger("debug", ""),
		address: loom.Address{ChainID: "default", Local: address},
		signer:  signer,
		mainnetGatewayAddress: loom.Address{
			ChainID: "eth",
			Local:   mainnetGatewayAddr.Bytes(),
		},
		metrics:          NewMetrics("loom_tg_oracle"),
		status:           Status{},
		isLoomCoinOracle: true,
	}

	orc.ethClient, err = ConnectToMainnet("https://mainnet.infura.io/v3/a5a5151fecba45229aa77f0725c10241")
	require.NoError(t, err)

	orc.solGateway, err = ethcontract.NewMainnetGatewayContract(mainnetGatewayAddr, orc.ethClient)
	require.NoError(t, err)

	return orc
}

func TestMainnetEventSimulatorLoomCoinDepositEvents1(t *testing.T) {
	orc := newTestLoomCoinOracle(t)
	// 4 LOOM deposit txs that emitted one event each
	mainnetEventSimulator, err := newMainnetEventSimulator(orc, "testdata/mainnet_loomcoin_deposit_txs_1.json")
	require.NoError(t, err)

	// Check the Ethereum block we want to add simulated events to doesn't have any relevant events
	ethBlock := uint64(7330010)
	events, err := orc.fetchEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 0, len(events))

	events, err = mainnetEventSimulator.simulateEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 4, len(events))

	for _, ev := range events {
		require.Equal(t, ethBlock, ev.EthBlock)
		require.NotNil(t, ev.GetDeposit())
		require.Equal(t, TokenKind_LoomCoin, ev.GetDeposit().TokenKind)
	}
}

func TestMainnetEventSimulatorLoomCoinDepositEvents2(t *testing.T) {
	orc := newTestLoomCoinOracle(t)
	// 4 LOOM deposit txs, and 1 LOOM transfer tx, only the events from the deposit txs are
	// picked up by the simulator/oracle - the event from the transfer tx doesn't have sufficient
	// info to be processed by the TG.
	mainnetEventSimulator, err := newMainnetEventSimulator(orc, "testdata/mainnet_loomcoin_deposit_txs_2.json")
	require.NoError(t, err)

	// Check the Ethereum block we want to add simulated events to doesn't have any relevant events
	ethBlock := uint64(7330010)
	events, err := orc.fetchEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 0, len(events))

	_, err = mainnetEventSimulator.simulateEvents(ethBlock, ethBlock)
	require.Error(t, err)
	require.Equal(t, "number of Mainnet events (4) doesn't match number of source txs (5)", err.Error())
}

func TestMainnetEventSimulatorLoomCoinDepositEvents3(t *testing.T) {
	orc := newTestLoomCoinOracle(t)
	// 4 LOOM deposit txs, but one is a duplicate
	mainnetEventSimulator, err := newMainnetEventSimulator(orc, "testdata/mainnet_loomcoin_deposit_txs_3.json")
	require.NoError(t, err)

	// Check the Ethereum block we want to add simulated events to doesn't have any relevant events
	ethBlock := uint64(7330010)
	events, err := orc.fetchEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 0, len(events))

	_, err = mainnetEventSimulator.simulateEvents(ethBlock, ethBlock)
	require.Error(t, err)
	require.Equal(t, "number of Mainnet events (3) doesn't match number of source txs (4)", err.Error())
}
