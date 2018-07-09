package main

import (
	"fmt"
	"github.com/loomnetwork/loomchain/e2e/common"
	"os/exec"
	"strings"
	"testing"
)

func TestContractBlueprint(t *testing.T) {
	t.Skip("Blueprint needs internal process plugin to be run")
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"blueprint.toml", 1, "blueprint.genesis.json"},
		{"blueprint.toml", 2, "blueprint.genesis.json"},
		{"blueprint.toml", 4, "blueprint.genesis.json"},
		{"blueprint.toml", 6, "blueprint.genesis.json"},
	}

	for _, test := range tests {
		*common.Validators = test.n
		config, err := common.NewConfig("blueprint", test.testFile, test.genFile)
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

		if err := common.DoRun(*config); err != nil {
			t.Fatal(err)
		}
	}

}
