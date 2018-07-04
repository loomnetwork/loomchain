package main

import (
	"testing"
	"time"

	"encoding/json"
	"fmt"
	"github.com/loomnetwork/loomchain/e2e/common"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func TestE2eEvm(t *testing.T) {
	tests := []struct {
		testFile string
		n        int
		genFile  string
	}{
		{"evm-test.toml", 1, ""},
	}
	common.LoomPath = "../../loom"
	common.ContractDir = "../../contracts"
	//
	for _, test := range tests {
		*common.Validators = test.n
		config, err := common.NewConfig("evm", test.testFile, test.genFile)
		if err != nil {
			t.Fatal(err)
		}
		readTestFiles("testbins.json", common.BaseDir, "evm")

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
	}
}

func readTestFiles(inFile, baseDir, name string) error {
	data, err := ioutil.ReadFile(inFile)
	if err != nil {
		return err
	}
	var fileContents []struct {
		Filename string `json:"filename"`
		Contents string `json:"contents"`
	}
	if err := json.Unmarshal(data, &fileContents); err != nil {
		return err
	}

	basedirAbs, err := filepath.Abs(path.Join(baseDir, name))
	if err != nil {
		return err
	}
	for _, fileInfo := range fileContents {
		absFileName := (path.Join(basedirAbs, fileInfo.Filename))
		ioutil.WriteFile(absFileName, []byte(fileInfo.Contents), 0644)
	}
	return nil
}
