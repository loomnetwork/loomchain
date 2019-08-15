package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestContractBlueprint(t *testing.T) {
	t.Skip("Blueprint needs internal process plugin to be run")
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
	}{
		{"blueprint-1", "blueprint.toml", 1, 10, "blueprint.genesis.json"},
		{"blueprint-2", "blueprint.toml", 2, 10, "blueprint.genesis.json"},
		{"blueprint-4", "blueprint.toml", 4, 10, "blueprint.genesis.json"},
		{"blueprint-6", "blueprint.toml", 6, 10, "blueprint.genesis.json"},
	}

	for _, test := range tests {
		config, err := common.NewConfig(test.name, test.testFile, test.genFile, "", test.validators, test.accounts, 0, false)
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