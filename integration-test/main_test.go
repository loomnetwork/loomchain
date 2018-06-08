package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/loomnetwork/loomchain/integration-test/engine"
	"github.com/loomnetwork/loomchain/integration-test/lib"
	"github.com/loomnetwork/loomchain/integration-test/node"
)

var (
	// assume that this test runs in integration-test directory
	loomPath    = "../loom"
	contractDir = "../contracts"
	baseDir     = "data"
)

var (
	validators = flag.Int("validators", 1, "The number of validators")
	numAccount = flag.Int("num-account", 3, "Number of account to be created")
	force      = flag.Bool("force", true, "Force to create a new directory")
	logLevel   = flag.String("log-level", "debug", "Contract log level")
	logDest    = flag.String("log-destination", "file://loom.log", "Log Destination")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func newConfig(name, testFile, genesisTmpl string) (*lib.Config, error) {
	basedirAbs, err := filepath.Abs(path.Join(baseDir, name))
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(basedirAbs)
	if !*force && err == nil {
		return nil, fmt.Errorf("directory %s exists; please use the flag --force to create new nodes", basedirAbs)
	}

	if *force {
		err = os.RemoveAll(basedirAbs)
		if err != nil {
			return nil, err
		}
	}

	loompathAbs, err := filepath.Abs(loomPath)
	if err != nil {
		return nil, err
	}
	contractdirAbs, err := filepath.Abs(contractDir)
	if err != nil {
		return nil, err
	}
	testFileAbs, err := filepath.Abs(testFile)
	if err != nil {
		return nil, err
	}

	conf := lib.Config{
		Name:        name,
		BaseDir:     basedirAbs,
		LoomPath:    loompathAbs,
		ContractDir: contractdirAbs,
		TestFile:    testFileAbs,
		Nodes:       make(map[string]*node.Node),
	}

	if err := os.MkdirAll(conf.BaseDir, os.ModePerm); err != nil {
		return nil, err
	}

	var accounts []*node.Account
	for i := 0; i < *numAccount; i++ {
		acct, err := node.CreateAccount(i, conf.BaseDir, conf.LoomPath)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acct)
	}

	var nodes []*node.Node
	for i := 0; i < *validators; i++ {
		node := node.NewNode(int64(i), conf.BaseDir, conf.LoomPath, conf.ContractDir, genesisTmpl)
		node.LogLevel = *logLevel
		node.LogDestination = *logDest
		nodes = append(nodes, node)
	}

	for _, node := range nodes {
		if err := node.Init(); err != nil {
			return nil, err
		}
	}

	if err = node.CreateCluster(nodes, accounts); err != nil {
		return nil, err
	}

	for _, node := range nodes {
		conf.Nodes[fmt.Sprintf("%d", node.ID)] = node
		conf.NodeAddressList = append(conf.NodeAddressList, node.Address)
		conf.NodePubKeyList = append(conf.NodePubKeyList, node.PubKey)
	}
	for _, account := range accounts {
		conf.AccountAddressList = append(conf.AccountAddressList, account.Address)
		conf.AccountPrivKeyPathList = append(conf.AccountPrivKeyPathList, account.PrivKeyPath)
		conf.AccountPubKeyList = append(conf.AccountPubKeyList, account.PubKey)
	}
	conf.Accounts = accounts
	if err := lib.WriteConfig(conf, "runner.toml"); err != nil {
		return nil, err
	}
	return &conf, nil
}

func runValidators(ctx context.Context, config lib.Config) error {
	// Trap Interrupts, SIGINTs and SIGTERMs.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigC)

	errC := make(chan error)
	eventC := make(chan *node.Event)
	e := engine.New(config)
	nctx, cancel := context.WithCancel(ctx)
	go func() { errC <- e.Run(nctx, eventC) }()

	for {
		select {
		case err := <-errC:
			cancel()
			return err
		case <-sigC:
			fmt.Printf("stopping runner\n")
			cancel()
			return nil
		}
	}
}

func runTests(ctx context.Context, config lib.Config, tc lib.Tests) error {
	// Trap Interrupts, SIGINTs and SIGTERMs.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigC)

	errC := make(chan error)
	eventC := make(chan *node.Event)
	e := engine.NewCmd(config, tc)

	nctx, cancel := context.WithCancel(ctx)
	go func() { errC <- e.Run(nctx, eventC) }()

	for {
		select {
		case err := <-errC:
			cancel()
			return err
		case <-sigC:
			cancel()
			fmt.Printf("stopping runner\n")
			return nil
		}
	}
}

func doRun(t *testing.T, config lib.Config) {
	// run validators
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := runValidators(ctx, config)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// wait for validators running
	time.Sleep(2000 * time.Millisecond)

	// run test case
	tc, err := lib.ReadTestCases(config.TestFile)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := runTests(ctx, config, tc)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// wait to clean up
	select {
	case <-ctx.Done():
		time.Sleep(500 * time.Millisecond)
	}
}
