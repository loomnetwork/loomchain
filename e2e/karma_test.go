package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
	
	"github.com/loomnetwork/loomchain/e2e/common"
)

func TestE2eKarma(t *testing.T) {
	tests := []struct {
		name       string
		testFile   string
		validators int
		accounts   int
		genFile    string
		yamlFile   string
	}{
		{"karma", "karma-test.toml", 4, 10, "karma.genesis.json", "karma-test.yaml",},
	}
	common.LoomPath = "../loom"
	common.ContractDir = "../contracts"
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfig(test.name, test.testFile, test.genFile, test.yamlFile, test.validators, test.accounts)
			if err != nil {
				t.Fatal(err)
			}
			
			binary, err := exec.LookPath("go")
			if err != nil {
				t.Fatal(err)
			}
			// build loom binary
			cmdBuildLoom := exec.Cmd{
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
			if err := cmdBuildLoom.Run(); err != nil {
				t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmdBuildLoom.Args, " "), err))
			}
			
			// build example-cli binary
			cmdBuildExmpleCli := exec.Cmd{
				Dir:  config.BaseDir,
				Path: binary,
				Args: []string{binary, "build", "-o", "example-cli", "github.com/loomnetwork/go-loom/examples/cli"},
			}
			if err := cmdBuildExmpleCli.Run(); err != nil {
				t.Fatal(fmt.Errorf("fail to execute command: %s\n%v", strings.Join(cmdBuildExmpleCli.Args, " "), err))
			}
			
			if err := common.DoRun(*config); err != nil {
				t.Fatal(err)
			}
			
			// pause before running the next test
			time.Sleep(500 * time.Millisecond)
		})
	}
}
