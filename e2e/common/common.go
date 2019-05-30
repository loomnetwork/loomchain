package common

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/loomnetwork/loomchain/e2e/engine"
	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
)

var (
	// assume that this test runs in e2e directory
	LoomPath    = "../loom"
	ContractDir = "../contracts"
	BaseDir     = "test-data"
)

var (
	Force    = flag.Bool("Force", true, "Force to create a new directory")
	LogLevel = flag.String("log-level", "debug", "Contract log level")
	LogDest  = flag.String("log-destination", "file://loom.log", "Log Destination")
	LogAppDb = flag.Bool("log-app-db", false, "Log app db usage to file")
)

func NewConfig(name, testFile, genesisTmpl, yamlFile string, validators, account, numEthAccounts int, useFnConsensus bool) (*lib.Config, error) {
	basedirAbs, err := filepath.Abs(path.Join(BaseDir, name))
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(basedirAbs)
	if !*Force && err == nil {
		return nil, fmt.Errorf("directory %s exists; please use the flag --force to create new nodes", basedirAbs)
	}

	if *Force {
		err = os.RemoveAll(basedirAbs)
		if err != nil {
			return nil, err
		}
	}

	loompathAbs, err := filepath.Abs(LoomPath)
	if err != nil {
		return nil, err
	}
	contractdirAbs, err := filepath.Abs(ContractDir)
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
	for i := 0; i < account; i++ {
		acct, err := node.CreateAccount(i, conf.BaseDir, conf.LoomPath)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acct)
	}

	var ethAccounts []*node.EthAccount
	for i := 0; i < numEthAccounts; i++ {
		acct, err := node.CreateEthAccount(i, conf.BaseDir)
		if err != nil {
			return nil, err
		}
		ethAccounts = append(ethAccounts, acct)
	}

	var tronAccounts []*node.TronAccount
	for i := 0; i < numEthAccounts; i++ {
		acct, err := node.CreateTronAccount(i, conf.BaseDir)
		if err != nil {
			return nil, err
		}
		tronAccounts = append(tronAccounts, acct)
	}

	var nodes []*node.Node
	for i := 0; i < validators; i++ {
		n := node.NewNode(int64(i), conf.BaseDir, conf.LoomPath, conf.ContractDir, genesisTmpl, yamlFile)
		n.LogLevel = *LogLevel
		n.LogDestination = *LogDest
		n.LogAppDb = *LogAppDb
		nodes = append(nodes, n)
	}

	for _, n := range nodes {
		if err := n.Init(accounts); err != nil {
			return nil, err
		}
	}

	if err = node.CreateCluster(nodes, accounts, useFnConsensus); err != nil {
		return nil, err
	}

	for _, n := range nodes {
		conf.Nodes[fmt.Sprintf("%d", n.ID)] = n
		conf.NodeAddressList = append(conf.NodeAddressList, n.Address)
		conf.NodeBase64AddressList = append(conf.NodeBase64AddressList, n.Local)
		conf.NodePubKeyList = append(conf.NodePubKeyList, n.PubKey)
		conf.NodePrivKeyPathList = append(conf.NodePrivKeyPathList, n.PrivKeyPath)
		conf.NodeProxyAppAddressList = append(conf.NodeProxyAppAddressList, n.ProxyAppAddress)
	}
	for _, account := range accounts {
		conf.AccountAddressList = append(conf.AccountAddressList, account.Address)
		conf.AccountPrivKeyPathList = append(conf.AccountPrivKeyPathList, account.PrivKeyPath)
		conf.AccountPubKeyList = append(conf.AccountPubKeyList, account.PubKey)
	}
	conf.Accounts = accounts

	for _, ethAccount := range ethAccounts {
		conf.EthAccountAddressList = append(conf.EthAccountAddressList, ethAccount.Address)
		conf.EthAccountPrivKeyPathList = append(conf.EthAccountPrivKeyPathList, ethAccount.PrivKeyPath)
		conf.EthAccountPubKeyList = append(conf.EthAccountPubKeyList, ethAccount.PubKey)
	}
	conf.EthAccounts = ethAccounts

	for _, tronAccount := range tronAccounts {
		conf.TronAccountAddressList = append(conf.TronAccountAddressList, tronAccount.Address)
		conf.TronAccountPrivKeyPathList = append(conf.TronAccountPrivKeyPathList, tronAccount.PrivKeyPath)
		conf.TronAccountPubKeyList = append(conf.TronAccountPubKeyList, tronAccount.PubKey)
	}
	conf.TronAccounts = tronAccounts

	if err := lib.WriteConfig(conf, "runner.toml"); err != nil {
		return nil, err
	}
	return &conf, nil
}

func DoRun(config lib.Config) error {
	// run validators
	ctx, cancel := context.WithCancel(context.Background())
	errC := make(chan error)
	// eventC is shared between runValidators & runTests so that tests can
	// interact with validators
	eventC := make(chan *node.Event)

	go func() {
		err := runValidators(ctx, config, eventC)
		errC <- err
	}()

	// wait for validators running
	time.Sleep(3000 * time.Millisecond)

	// run test case
	tc, err := lib.ReadTestCases(config.TestFile)
	if err != nil {
		cancel()
		return err
	}

	go func() {
		err := runTests(ctx, config, tc, eventC)
		errC <- err
	}()

	// wait to clean up
	select {
	case err := <-errC:
		cancel()
		time.Sleep(1000 * time.Millisecond)
		return err
	case <-ctx.Done():
	}
	cancel()
	time.Sleep(1000 * time.Millisecond)

	return nil
}

func runValidators(ctx context.Context, config lib.Config, eventC chan *node.Event) error {
	// Trap Interrupts, SIGINTs and SIGTERMs.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigC)

	errC := make(chan error)
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

func runTests(ctx context.Context, config lib.Config, tc lib.Tests, eventC chan *node.Event) error {
	// Trap Interrupts, SIGINTs and SIGTERMs.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigC)

	errC := make(chan error)
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
