package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestContractCoin(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"coin.toml", 1, "coin.genesis.json"},
		{"coin.toml", 2, "coin.genesis.json"},
		{"coin.toml", 4, "coin.genesis.json"},
		{"coin.toml", 6, "coin.genesis.json"},
		{"coin.toml", 8, "coin.genesis.json"},
	}
	for _, test := range tests {
		*validators = test.n
		config, err := newConfig("coin", test.testFile, test.genFile)
		if err != nil {
			t.Fatal(err)
		}

		binary, err := exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		// required binary
		cmd := exec.Cmd{
			Dir:  config.BaseDir,
			Path: binary,
			Args: []string{binary, "build", "-o", "example-cli", "github.com/loomnetwork/go-loom/examples/cli"},
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmd.Args, " "), err))
		}

		if err := doRun(*config); err != nil {
			t.Fatal(err)
		}
	}
}
