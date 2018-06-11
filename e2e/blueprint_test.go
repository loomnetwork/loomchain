package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestContractBlueprint(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
	}{
		{"blueprint.toml", 1},
		{"blueprint.toml", 2},
		{"blueprint.toml", 4},
		{"blueprint.toml", 6},
	}

	for _, test := range tests {
		*validators = test.n
		config, err := newConfig("blueprint", test.testFile, "blueprint.genesis.json")
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
			Args: []string{binary, "build", "-o", "blueprint-cli", "github.com/loomnetwork/go-loom/cli/blueprint"},
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmd.Args, " "), err))
		}

		if err := doRun(*config); err != nil {
			t.Fatal(err)
		}
	}

}
