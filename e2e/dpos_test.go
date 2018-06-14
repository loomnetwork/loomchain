package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestContractDPOS(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"dpos-1-validators.toml", 1, ""},
		{"dpos-2-validators.toml", 2, ""},
		{"dpos-4-validators.toml", 4, ""},
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

		// pause before running the next test
		time.Sleep(500 * time.Millisecond)
	}
}
