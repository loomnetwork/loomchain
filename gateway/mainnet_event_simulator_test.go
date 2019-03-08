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

func TestMainnetEventSimulatorLoomCoinDepositEvents(t *testing.T) {
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

	mainnetEventSimulator, err := newMainnetEventSimulator(orc, "testdata/mainnet_event_source_txs.json")
	require.NoError(t, err)

	// Check the Ethereum block we want to add simulated events to doesn't have any relevant events
	ethBlock := uint64(7330010)
	events, err := orc.fetchEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 0, len(events))

	events, err = mainnetEventSimulator.simulateEvents(ethBlock, ethBlock)
	require.NoError(t, err)
	require.Equal(t, 4, len(events))

}
