package engine

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/pkg/errors"
)

var (
	loomCmds = []string{"loom", "blueprint-cli"}
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

func getCommand(conf lib.Config, node node.Node, test lib.TestCase) (exec.Cmd, error) {
	t, err := template.New("cmd").Parse(test.RunCmd)
	if err != nil {
		return exec.Cmd{}, err
	}
	buf := new(bytes.Buffer)
	conf.LoomPath = node.LoomPath
	err = t.Execute(buf, conf)
	if err != nil {
		return exec.Cmd{}, err
	}

	dir := conf.BaseDir
	if test.Dir != "" {
		dir = test.Dir
	}
	return makeCmd(buf.String(), dir, node)
}

func (e *engineCmd) Run(ctx context.Context, eventC chan *node.Event) error {
	if err := e.waitForClusterToStart(); err != nil {
		return errors.Wrap(err, "❌ failed to start cluster")
	}
	fmt.Printf("cluster is ready\n")

	for _, n := range e.tests.TestCases {
		dir := e.conf.BaseDir
		if n.Dir != "" {
			dir = n.Dir
		}
		if err := makeTestFiles(n.Datafiles, dir); err != nil {
			return err
		}

		// special command to check app hash
		if n.RunCmd == "checkapphash" {
			time.Sleep(time.Duration(n.Delay) * time.Millisecond)
			if err := checkAppHash(e.conf.Nodes); err != nil {
				return errors.Wrap(err, "checking apphash")
			}
			continue
		}

		iter := n.Iterations
		if iter == 0 {
			iter = 1
		}
		for i := 0; i < iter; i++ {
			// check all  the nodes
			if n.All {
				for j, v := range e.conf.Nodes {
					cmd, err := getCommand(e.conf, *v, n)
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

					err = checkConditions(e, n, out)
					if err != nil {
						return err
					}
				}
			} else {
				queryNode, ok := e.conf.Nodes[fmt.Sprintf("%d", n.Node)]
				if !ok {
					return fmt.Errorf("node 0 not found")
				}
				cmd, err := getCommand(e.conf, *queryNode, n)
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
						durationArg, err := strconv.ParseInt(cmd.Args[1], 10, 64)
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
					event := node.Event{
						Action:   node.ActionStop,
						Duration: node.Duration{Duration: time.Duration(duration)},
						Delay:    node.Duration{Duration: time.Duration(0)},
						Node:     nodeId,
					}
					eventC <- &event
					out = []byte(fmt.Sprintf("Sending Node Event: %v\n", event))
				} else if cmd.Args[0] == "wait_node_to_start" {
					if len(cmd.Args) > 1 {
						maxRetries := 10
						if len(cmd.Args) > 2 {
							max, err := strconv.Atoi(cmd.Args[2])
							if err == nil {
								maxRetries = max
							}
						}
						nodeStarted := false
						for i := maxRetries; i > 0; i-- {
							node, ok := e.conf.Nodes[cmd.Args[1]]
							if !ok {
								return fmt.Errorf("node %s is not found", cmd.Args[1])
							}
							if err := checkNodeReady(node); err == nil {
								nodeStarted = true
								break
							}
							time.Sleep(time.Duration(time.Second))
						}
						if !nodeStarted {
							return fmt.Errorf("node %s did not start", cmd.Args[1])
						}
					}

				} else if cmd.Args[0] == "wait_for_block_height_to_increase" {
					if len(cmd.Args) > 2 {
						maxWaitingTime := 60 // 60s
						maxRetries := 3
						waitNBlocks, err := strconv.Atoi(cmd.Args[2])
						if err != nil {
							return fmt.Errorf("waiting block number is not defined, err: %s", err)
						}
						var lastBlockHeight int64
						for i := maxRetries; i > 0; i-- {
							lastBlockHeight, err = getLastBlockHeight(e.conf.Nodes[cmd.Args[1]])
							if err != nil {
								break
							}
						}
						if lastBlockHeight == 0 {
							return fmt.Errorf("cannot get last block height from node %s", cmd.Args[1])
						}
						for i := maxWaitingTime; i > 0; i-- {
							currentBlockHeight, _ := getLastBlockHeight(e.conf.Nodes[cmd.Args[1]])
							if currentBlockHeight > lastBlockHeight+int64(waitNBlocks) {
								break
							}
							fmt.Printf("current block height %d\n", currentBlockHeight)
							time.Sleep(time.Duration(time.Second))
						}
					}
				} else if cmd.Args[0] == "wait_for_node_to_catch_up" {
					if len(cmd.Args) > 1 {
						maxWaitingTime := 60 // 60s
						for i := maxWaitingTime; i > 0; i-- {
							cachingUp, err := nodeCatchingUp(e.conf.Nodes[cmd.Args[1]])
							if err == nil && !cachingUp {
								break
							}
							time.Sleep(time.Duration(time.Second))
						}
					}
				} else if cmd.Args[0] == "wait_for_block_height_to_reach" {
					if len(cmd.Args) > 2 {
						maxWaitingTime := 60 // 60s
						targetBlock, err := strconv.Atoi(cmd.Args[2])
						if err != nil {
							return fmt.Errorf("target block number is not defined, err: %s", err)
						}
						for i := maxWaitingTime; i > 0; i-- {
							currentBlockHeight, _ := getLastBlockHeight(e.conf.Nodes[cmd.Args[1]])
							fmt.Printf("current block height %d\n", currentBlockHeight)
							if currentBlockHeight >= int64(targetBlock) {
								break
							}
							time.Sleep(time.Duration(time.Second))
						}
					}
				} else {
					out, err = cmd.CombinedOutput()
				}

				if err != nil {
					fmt.Printf("--> error: %s\n", err)
				}
				fmt.Printf("--> output:\n%s\n", out)

				err = checkConditions(e, n, out)
				if err != nil {
					return err
				}

			}
		}
		if e.conf.CheckAppHash {
			if err := checkAppHash(e.conf.Nodes); err != nil {
				return errors.Wrapf(err, "check apphash failed after test command, %s", n.RunCmd)
			}
		}
	}

	return nil
}

type AppHash struct {
	apphash string
	node    *node.Node
	index   string
}

func sprintAppHashes(block []AppHash) string {
	var hashInfo string
	for _, apphash := range block {
		hashInfo += fmt.Sprintf(
			"node %s apphash 0x%s executable %s\n",
			apphash.index,
			apphash.apphash,
			apphash.node.LoomPath,
		)
	}
	return hashInfo
}

func getBlockHeight(node *node.Node) (string, error) {
	req := fmt.Sprintf("%s/status", node.RPCAddress)
	resp, err := http.Get(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf(
			"post status not OK: %s, response body: %s", resp.Status, string(respBytes),
		)
	}
	var info = struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  struct {
			SyncInfo struct {
				LastBlockAppHash  []byte `json:"last_block_app_hash,omitempty"`
				LatestBlockHeight string `json:"latest_block_height,omitempty"`
			} `json:"sync_info,omitempty"`
		} `json:"result"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		fmt.Println("err", err)
		return "", err
	}
	return info.Result.SyncInfo.LatestBlockHeight, nil
}

func getAppHash(node *node.Node, height string) (string, error) {
	req := fmt.Sprintf("%s/commit?height=%s", node.RPCAddress, height)
	resp, err := http.Get(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		respBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf(
			"post status not OK: %s, response body: %s", resp.Status, string(respBytes),
		)
	}
	var info = struct {
		JSONRPC string `json:"jsonrpc"`
		ID      string `json:"id"`
		Result  struct {
			SignedHeader struct {
				Header struct {
					AppHash string `json:"app_hash,omitempty"`
				} `json:"header,omitempty"`
			} `json:"signed_header,omitempty"`
		} `json:"result"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		fmt.Println("err", err)
		return "", err
	}
	return info.Result.SignedHeader.Header.AppHash, nil
}

func checkAppHash(nodes map[string]*node.Node) error {
	time.Sleep(time.Second * 1)
	fmt.Printf("--> run all: %v \n", "checkapphash")

	blockInfo := []AppHash{}
	var blockHeight string
	if node0, ok := nodes["0"]; ok {
		var err error
		blockHeight, err = getBlockHeight(node0)
		if err != nil {
			return err
		}
	} else {
		return errors.New("no node 0")
	}
	for index, v := range nodes {
		currentAppHash, err := getAppHash(v, blockHeight)
		if err != nil {
			return err
		}
		fmt.Printf("--> Node: %s, AppHash: 0x%s, height %v\n", index, currentAppHash, blockHeight)
		blockInfo = append(blockInfo, AppHash{
			apphash: currentAppHash,
			node:    v,
			index:   index,
		})
	}

	for i := 1; i < len(blockInfo); i++ {
		if blockInfo[i-1].apphash != blockInfo[i].apphash {
			return errors.Errorf("app hash mismatch\n%s", sprintAppHashes(blockInfo))
		}
	}
	return nil
}

func (e *engineCmd) waitForClusterToStart() error {
	maxRetries := 5
	readyNodes := map[string]bool{}
	for i := 0; i < maxRetries; i++ {
		for nodeID, nodeCfg := range e.conf.Nodes {
			if !readyNodes[nodeID] {
				if err := checkNodeReady(nodeCfg); err != nil {
					fmt.Printf("node %s isn't ready yet: %v\n", nodeID, err)
				} else {
					readyNodes[nodeID] = true
				}
			}
		}
		if len(readyNodes) == len(e.conf.Nodes) {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	if len(readyNodes) != len(e.conf.Nodes) {
		return fmt.Errorf("%d/%d nodes are running", len(readyNodes), len(e.conf.Nodes))
	}
	return nil
}

func checkConditions(e *engineCmd, n lib.TestCase, out []byte) error {
	switch n.Condition {
	case "contains":
		var expecteds []string
		for _, expected := range n.Expected {
			t, err := template.New("expected").Parse(expected)
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

		for _, expected := range expecteds {
			if !strings.Contains(string(out), expected) {
				return fmt.Errorf("❌ expect output to contain '%s' got '%s'", expected, string(out))
			}
		}
	case "excludes":
		var excludeds []string
		for _, excluded := range n.Excluded {
			t, err := template.New("excluded").Parse(excluded)
			if err != nil {
				return err
			}
			buf := new(bytes.Buffer)
			err = t.Execute(buf, e.conf)
			if err != nil {
				return err
			}
			excludeds = append(excludeds, buf.String())
		}

		for _, excluded := range excludeds {
			if strings.Contains(string(out), excluded) {
				return fmt.Errorf("❌ expect output to exclude '%s' got '%s'", excluded, string(out))
			}
		}
	case "":
	default:
		return fmt.Errorf("Unrecognized test condition %s.", n.Condition)
	}
	return nil
}

func checkNodeReady(n *node.Node) error {
	// With empty blocks disabled there's no point waiting for the node to process the first two
	// blocks since they'll only be created when the first tx is sent through.
	if !n.Config.CreateEmptyBlocks {
		return nil
	}

	type ResponseInfo struct {
		LastBlockHeight string `json:"last_block_height"`
	}
	type ResultABCIInfo struct {
		Response ResponseInfo `json:"response"`
	}
	type Response struct {
		Result ResultABCIInfo `json:"result"`
	}

	u := fmt.Sprintf("%s/abci_info", n.RPCAddress)
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	rawResp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer rawResp.Body.Close()
	respBytes, _ := ioutil.ReadAll(rawResp.Body)
	var resp Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return err
	}

	lastBlockHeight, err := strconv.ParseInt(resp.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return err
	}
	// We want to wait for both the genesis block and the following confirmation block to be
	// processed by the app before we start interacting with the node.
	if lastBlockHeight < 2 {
		return fmt.Errorf("LastBlockHeight: %d", lastBlockHeight)
	}
	return nil
}

func nodeCatchingUp(n *node.Node) (bool, error) {
	type CatchingUp struct {
		CatchingUp         bool   `json:"catching_up"`
		LastestBlockHeight string `json:"latest_block_height"`
	}
	type SyncInfo struct {
		CatchingUpResult CatchingUp `json:"sync_info"`
	}
	type Response struct {
		Result SyncInfo `json:"result"`
	}

	u := fmt.Sprintf("%s/status", n.RPCAddress)
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	rawResp, err := client.Get(u)
	if err != nil {
		return true, err
	}
	defer rawResp.Body.Close()
	respBytes, _ := ioutil.ReadAll(rawResp.Body)
	var resp Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return true, err
	}

	fmt.Printf("SyncInfo %+v\n", resp.Result)

	return resp.Result.CatchingUpResult.CatchingUp, nil
}

func getLastBlockHeight(n *node.Node) (int64, error) {
	type ResponseInfo struct {
		LastBlockHeight string `json:"last_block_height"`
	}
	type ResultABCIInfo struct {
		Response ResponseInfo `json:"response"`
	}
	type Response struct {
		Result ResultABCIInfo `json:"result"`
	}

	u := fmt.Sprintf("%s/abci_info", n.RPCAddress)
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}
	rawResp, err := client.Get(u)
	if err != nil {
		return 0, err
	}
	defer rawResp.Body.Close()
	respBytes, _ := ioutil.ReadAll(rawResp.Body)
	var resp Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return 0, err
	}

	lastBlockHeight, err := strconv.ParseInt(resp.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return 0, err
	}

	return lastBlockHeight, nil
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
		args[0] = node.LoomPath
		// Make sure we have uri/u endpoint as a default.
		if !strings.Contains(cmdString, "-u ") {
			args = append(args, "-u")
			args = append(args, node.ProxyAppAddress)
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
		if path.Base(cmd) == loomCmd {
			return true
		}
	}
	return false
}
