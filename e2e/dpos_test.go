package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestContractDPOS(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"dpos-1-validators.toml", 1, ""},
		// Skip more than 2 nodes sice the result is still inconsistent
		// {"dpos-2-validators.toml", 2},
		// {"dpos-4-validators.toml", 4},
		// {"dpos-4-validators.toml", 8},
		// {"dpos-6-validators.toml", 8},
		// {"dpos-6-validators.toml", 10},
	}

	for _, test := range tests {
		*validators = test.n
		config, err := newConfig("dpos", test.testFile, test.genFile)
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
