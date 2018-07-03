// +build evm

package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/loomnetwork/go-loom/auth"
	"github.com/loomnetwork/go-loom/client/plasma_cash/eth"
	"github.com/loomnetwork/loomchain/builtin/plugins/plasma_cash/oracle"
	"golang.org/x/crypto/ed25519"
)

func main() {
	//TODO read environment variables or command line to get this data
	rootChainHexAddress := "0x00000000"
	privateKeyHex := "0x000000"

	_, loomPrivKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Fatalf("failed to gnerate private key for authority: %v", err)
	}
	ethPrivKeyHexStr := privateKeyHex
	ethPrivKey, err := crypto.HexToECDSA(strings.TrimPrefix(ethPrivKeyHexStr, "0x"))
	if err != nil {
		log.Fatalf("failed to load private key: %v", err)
	}
	plasmaOrc := oracle.NewOracle(&oracle.OracleConfig{
		PlasmaBlockInterval: 1000,
		DAppChainClientCfg: oracle.DAppChainPlasmaClientConfig{
			ChainID:  "default",
			WriteURI: "http://localhost:46658/rpc", //TODO make configs
			ReadURI:  "http://localhost:46658/query",
			Signer:   auth.NewEd25519Signer(loomPrivKey),
		},
		EthClientCfg: eth.EthPlasmaClientConfig{
			//TODO add options for infura also
			EthereumURI:      "http://localhost:8545", //TODO make configs
			PlasmaHexAddress: rootChainHexAddress,
			PrivateKey:       ethPrivKey,
			OverrideGas:      true,
		},
	})
	if err := plasmaOrc.Init(); err != nil {
		log.Fatal(err)
	}

	// Trap Interrupts, SIGINTs and SIGTERMs.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigC)

	plasmaOrc.Run()
	<-sigC
}
