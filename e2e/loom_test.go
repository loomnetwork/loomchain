package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestE2eEvm(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
	}{
		{"evm", "loom-test.toml", 4, 10, ""},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.validators, test.accounts)
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
				Args: []string{
					binary,
					"build",
					"-tags",
					"evm",
					"-o",
					"loom",
					"github.com/loomnetwork/loomchain/cmd/loom",
				},
			}
			if err := cmd.Run(); err != nil {
				t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmd.Args, " "), err))
			}

			if err := common.DoRun(*config); err != nil {
				t.Fatal(err)
			}

			// pause before running the next test
			time.Sleep(500 * time.Millisecond)
		})
	}
}
