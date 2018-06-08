package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestContractBlueprint(t *testing.T) {
	config, err := newConfig("blueprint", "blueprint.toml", "blueprint.genesis.json")
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

	doRun(t, *config)
}
