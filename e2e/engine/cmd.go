package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
)

var (
	loomCmds = []string{"loom", "example-cli", "blueprint-cli"}
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

type abciResponseInfo2 struct {
	Data             string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Version          string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	LastBlockHeight  string `protobuf:"varint,3,opt,name=last_block_height,json=lastBlockHeight,proto3" json:"last_block_height,omitempty"`
	LastBlockAppHash []byte `protobuf:"bytes,4,opt,name=last_block_app_hash,json=lastBlockAppHash,proto3" json:"last_block_app_hash,omitempty"`
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
		makeTestFiles(n.Datafiles, dir)

		// special command to check app hash
		if base == "checkapphash" {
			time.Sleep(time.Duration(n.Delay) * time.Millisecond)
			time.Sleep(time.Second * 1)
			fmt.Printf("--> run all: %v \n", "checkapphash")
			var apphash = make(map[string]struct{})
			var lastBlockHeight int64
			for _, v := range e.conf.Nodes {
				u := fmt.Sprintf("%s/abci_info", v.RPCAddress)
				resp, err := http.Get(u)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				if resp.StatusCode != 200 {
					respBytes, _ := ioutil.ReadAll(resp.Body)
					return fmt.Errorf("post status not OK: %s, response body: %s", resp.Status, string(respBytes))
				}
				var info = struct {
					JSONRPC string `json:"jsonrpc"`
					ID      string `json:"id"`
					Result  struct {
						Response abciResponseInfo2 `json:"response"`
					} `json:"result"`
				}{}

				err = json.NewDecoder(resp.Body).Decode(&info)
				if err != nil {
					return err
				}
				newLastBlockHeight, err := strconv.ParseInt(info.Result.Response.LastBlockHeight, 10, 64)
				if err != nil {
					return err
				}
				if lastBlockHeight == 0 {
					lastBlockHeight = newLastBlockHeight
				}
				if lastBlockHeight == newLastBlockHeight {
					apphash[string(info.Result.Response.LastBlockAppHash)] = struct{}{}
					fmt.Printf("--> GET: %s, AppHash: %0xX\n", u, info.Result.Response.LastBlockAppHash)
				}
			}

			// apphash should has only 1 entry
			// this might not be true if network latency is hight
			if len(apphash) != 1 {
				return fmt.Errorf("Wrong Block.Header.AppHash")
			}
			continue
		}

		for i := 0; i < iter; i++ {
			// check all  the nodes
			if n.All {
				for j, v := range e.conf.Nodes {
					cmd, err := makeCmd(base, dir, *v)
					if err != nil {
						return err
					}

					fmt.Printf("--> node %s; run all: %v \n", j, strings.Join(cmd.Args, " "))
					if n.Delay > 0 {
						time.Sleep(time.Duration(n.Delay) * time.Millisecond)
					}

					// sleep 1 second to make sure the last tx is processed
					time.Sleep(1 * time.Second)

					out, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Printf("--> error: %s\n", err)
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
								return fmt.Errorf("❌ expect output to contain '%s' - got '%s'", expected, string(out))
							}
						}
					case "":
					default:
						return fmt.Errorf("Unrecognized test condition %s.", n.Condition)
					}
				}
			} else {
				queryNode, ok := e.conf.Nodes[fmt.Sprintf("%d", n.Node)]
				if !ok {
					return fmt.Errorf("node 0 not found")
				}
				cmd, err := makeCmd(buf.String(), dir, *queryNode)
				if err != nil {
					return err
				}
				fmt.Printf("--> run: %s\n", strings.Join(cmd.Args, " "))
				if n.Delay > 0 {
					time.Sleep(time.Duration(n.Delay) * time.Millisecond)
				}

				// sleep 1 second to make sure the last tx is processed
				time.Sleep(1 * time.Second)

				var out []byte
				if cmd.Args[0] == "check_validators" {
					out, err = checkValidators(queryNode)
				} else if cmd.Args[0] == "kill_and_restart_node" {
					nanosecondsPerSecond := 1000000000
					duration := 4 * nanosecondsPerSecond
					nodeId := 0
					if len(cmd.Args) > 1 {
						durationArg, err  := strconv.ParseInt(cmd.Args[1], 10, 64)
						if err != nil {
							return err
						}

						// convert to nanoseconds
						duration = int(durationArg) * nanosecondsPerSecond

						if len(cmd.Args) > 2 {
							nodeIdArg, err := strconv.ParseInt(cmd.Args[2], 10, 64)
							if err != nil {
								return err
							}

							nodeId = int(nodeIdArg)
						}
					}
					event := node.Event{Action: node.ActionStop, Duration: node.Duration{time.Duration(duration)}, Delay: node.Duration{time.Duration(0)}, Node: nodeId}
					eventC <- &event
					out = []byte(fmt.Sprintf("Sending Node Event: %s\n", event))
				} else {
					out, err = cmd.CombinedOutput()
				}

				if err != nil {
					fmt.Printf("--> error: %s\n", err)
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
							return fmt.Errorf("❌ expect output to contain '%s' got '%s'", expected, string(out))
						}
					}
				case "":
				default:
					return fmt.Errorf("Unrecognized test condition %s.", n.Condition)
				}
			}
		}
	}

	return nil
}

func checkValidators(node *node.Node) ([]byte, error) {
	u := fmt.Sprintf("%s/validators", node.RPCAddress)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, _ := ioutil.ReadAll(resp.Body)
	return respBytes, nil
}

func makeTestFiles(filesInfo []lib.Datafile, dir string) error {
	for _, fileInfo := range filesInfo {
		filename := path.Join(dir, fileInfo.Filename)
		if err := ioutil.WriteFile(filename, []byte(fileInfo.Contents), 0644); err != nil {
			return err
		}
	}
	return nil
}

func makeCmd(cmdString, dir string, node node.Node) (exec.Cmd, error) {
	args := strings.Split(cmdString, " ")
	if len(args) == 0 {
		return exec.Cmd{}, errors.New("missing command")
	}

	if isLoomCmd(args[0]) {
		// Make sure we have query and rpc endpoint as a default.
		// If there is no /query and /rpc, pick the first default one and append to args.
		if !strings.Contains(cmdString, "/rpc") {
			args = append(args, "-w")
			args = append(args, fmt.Sprintf("%s/rpc", node.ProxyAppAddress))
		}
		if !strings.Contains(cmdString, "/query") {
			args = append(args, "-r")
			args = append(args, fmt.Sprintf("%s/query", node.ProxyAppAddress))
		}
	}
	return exec.Cmd{
		Dir:  dir,
		Path: args[0],
		Args: args,
	}, nil
}

func isLoomCmd(cmd string) bool {
	for _, loomCmd := range loomCmds {
		if cmd == loomCmd {
			return true
		}
	}
	return false
}
