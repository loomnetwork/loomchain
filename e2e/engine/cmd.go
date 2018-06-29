package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
)

type engineCmd struct {
	conf  lib.Config
	tests lib.Tests
	wg    *sync.WaitGroup
	errC  chan error
}

func NewCmd(conf lib.Config, tc lib.Tests) Engine {
	return &engineCmd{
		conf:  conf,
		tests: tc,
		wg:    &sync.WaitGroup{},
		errC:  make(chan error),
	}
}

func (e *engineCmd) Run(ctx context.Context, eventC chan *node.Event) error {
	for _, n := range e.tests.TestCases {
		// evaluate template
		t, err := template.New("cmd").Parse(n.RunCmd)
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		err = t.Execute(buf, e.conf)
		if err != nil {
			return err
		}

		iter := n.Iterations
		if iter == 0 {
			iter = 1
		}

		dir := e.conf.BaseDir
		if n.Dir != "" {
			dir = n.Dir
		}
		base := buf.String()

		// check app hash
		if base == "checkapphash" {
			time.Sleep(time.Duration(n.Delay) * time.Millisecond)
			fmt.Printf("--> run all: %v \n", "checkapphash")
			var apphash []string
			for _, v := range e.conf.Nodes {
				addr := v.ABCIAddress
				resp, err := http.Get(addr + "/abci_info")
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				apphash = append(apphash, string(data))
			}

			fmt.Printf("--> run all: %v \n", apphash)
			hash0 := apphash[0]
			for i := 1; i < len(apphash); i++ {
				if hash0 != apphash[i] {
					return fmt.Errorf("Wrong Block.Header.AppHas")
				}
			}
			continue
		}

		for i := 0; i < iter; i++ {
			// check all  the nodes
			if n.All {
				for _, v := range e.conf.Nodes {
					rpc := v.RPCAddress
					args := strings.Split(base, " ")
					if len(args) == 0 {
						return errors.New("missing command")
					}
					args = append(args, []string{"-r", fmt.Sprintf("%s/query", rpc)}...)
					args = append(args, []string{"-w", fmt.Sprintf("%s/rpc", rpc)}...)
					fmt.Printf("--> run all: %v \n", strings.Join(args, " "))
					// add host
					cmd := exec.Cmd{
						Dir:  dir,
						Path: args[0],
						Args: args,
					}
					if n.Delay > 0 {
						time.Sleep(time.Duration(n.Delay) * time.Millisecond)
					}

					time.Sleep(1 * time.Second)

					out, err := cmd.Output()
					if err != nil {
						fmt.Printf("--> error: %s\n", err)
						continue
					}
					fmt.Printf("--> output:\n%s\n", out)

					var expecteds []string
					for _, expected := range n.Expected {
						t, err = template.New("expected").Parse(expected)
						if err != nil {
							return err
						}
						buf := new(bytes.Buffer)
						err = t.Execute(buf, e.conf)
						if err != nil {
							return err
						}
						expecteds = append(expecteds, buf.String())
					}

					switch n.Condition {
					case "contains":
						for _, expected := range expecteds {
							if !strings.Contains(string(out), expected) {
								return fmt.Errorf("❌ expect output to contain '%s'", expected)
							}
						}
					}
				}
			} else {
				fmt.Printf("--> run: %s\n", buf.String())
				args := strings.Split(buf.String(), " ")
				if len(args) == 0 {
					return errors.New("missing command")
				}
				cmd := exec.Cmd{
					Dir:  dir,
					Path: args[0],
					Args: args,
				}
				if n.Delay > 0 {
					time.Sleep(time.Duration(n.Delay) * time.Millisecond)
				}

				time.Sleep(1 * time.Second)

				out, err := cmd.Output()
				if err != nil {
					fmt.Printf("--> error: %s\n", err)
					continue
				}
				fmt.Printf("--> output:\n%s\n", out)

				var expecteds []string
				for _, expected := range n.Expected {
					t, err = template.New("expected").Parse(expected)
					if err != nil {
						return err
					}
					buf := new(bytes.Buffer)
					err = t.Execute(buf, e.conf)
					if err != nil {
						return err
					}
					expecteds = append(expecteds, buf.String())
				}

				switch n.Condition {
				case "contains":
					for _, expected := range expecteds {
						if !strings.Contains(string(out), expected) {
							return fmt.Errorf("❌ expect output to contain '%s'", expected)
						}
					}
				}
			}
		}
	}

	return nil
}
