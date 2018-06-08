package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestContractCoin(t *testing.T) {
	config, err := newConfig("coin", "coin.toml", "")
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

	doRun(t, *config)
}
