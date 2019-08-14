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

	"github.com/pkg/errors"

	"github.com/loomnetwork/loomchain/e2e/engine"
	"github.com/loomnetwork/loomchain/e2e/lib"
	"github.com/loomnetwork/loomchain/e2e/node"
)

const (
	loomExeEv               = "LOOMEXE_PATH"
	loomExe2Ev              = "LOOMEXE_ALTPATH"
	checkAppHash            = "CHECK_APP_HASH"
	minRatioForAppHashCheck = 3
)

var (
	// assume that this test runs in e2e directory
	defaultLoomPath = "../loom"
	ContractDir     = "../contracts"
	BaseDir         = "test-data"
)

var (
	Force    = flag.Bool("Force", true, "Force to create a new directory")
	logLevel = flag.String("log-level", "debug", "Contract log level")
	logDest  = flag.String("log-destination", "file://loom.log", "Log Destination")
	logAppDb = flag.Bool("log-app-db", false, "Log app db usage to file")
)

func NewConfig(
	name, testFile, genesisTmpl, yamlFile string,
	validators, account, numEthAccounts int,
	useFnConsensus bool,
) (*lib.Config, error) {
	checkAppHashEV := os.Getenv(checkAppHash)
	checkAppHash := len(checkAppHashEV) > 0

	loomPath := os.Getenv(loomExeEv)
	if len(loomPath) == 0 {
		loomPath = defaultLoomPath
	}

	loomPath2 := os.Getenv(loomExe2Ev)
	v := uint64(validators)
	altV := uint64(0)
	if len(loomPath2) > 0 {
		v, altV = splitValidators(uint64(validators))
	}

	return GenerateConfig(
		name, testFile, genesisTmpl, yamlFile, BaseDir, ContractDir, loomPath, loomPath2,
		v, altV,
		account, numEthAccounts,
		useFnConsensus, *Force, doCheckAppHash(checkAppHash, uint64(v), uint64(altV)),
	)
}

func splitValidators(validators uint64) (uint64, uint64) {
	if validators == 0 {
		return 0, 0
	}

	if validators == 1 {
		return 1, 0
	}

	return validators - 1, 1
}

func doCheckAppHash(checkAppHash bool, validators, altValidators uint64) bool {
	if !checkAppHash || validators == 0 || altValidators == 0 {
		return false
	}
	return validators/altValidators >= minRatioForAppHashCheck ||
		altValidators/validators >= minRatioForAppHashCheck
}

func GenerateConfig(
	name, testFile, genesisTmpl, yamlFile, basedir, contractdir, loompath, loompath2 string,
	validators, altValidators uint64,
	account, numEthAccounts int,
	useFnConsensus, force, checkAppHash bool,
) (*lib.Config, error) {
	basedirAbs, err := filepath.Abs(path.Join(basedir, name))
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(basedirAbs)
	if !force && err == nil {
		return nil, fmt.Errorf("directory %s exists; please use the flag --force to create new nodes", basedirAbs)
	}

	if force {
		err = os.RemoveAll(basedirAbs)
		if err != nil {
			return nil, err
		}
	}

	contractdirAbs, err := filepath.Abs(contractdir)
	if err != nil {
		return nil, err
	}
	testFileAbs, err := filepath.Abs(testFile)
	if err != nil {
		return nil, err
	}

	conf := lib.Config{
		Name:         name,
		BaseDir:      basedirAbs,
		ContractDir:  contractdirAbs,
		TestFile:     testFileAbs,
		Nodes:        make(map[string]*node.Node),
		CheckAppHash: checkAppHash,
	}

	if err := os.MkdirAll(conf.BaseDir, os.ModePerm); err != nil {
		return nil, err
	}

	loompathAbs, err := filepath.Abs(loompath)
	if err != nil {
		return nil, err
	}
	if validators > 0 {
		if _, err := os.Stat(loompathAbs); os.IsNotExist(err) {
			return nil, errors.Errorf("cannot find loom executable %s", loompathAbs)
		}
	}

	var accounts []*node.Account
	for i := 0; i < account; i++ {
		acct, err := node.CreateAccount(i, conf.BaseDir, loompathAbs)
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
	for i := uint64(0); i < validators; i++ {
		n := node.NewNode(int64(i), conf.BaseDir, loompathAbs, conf.ContractDir, genesisTmpl, yamlFile)
		n.LogLevel = *logLevel
		n.LogDestination = *logDest
		n.LogAppDb = *logAppDb
		nodes = append(nodes, n)
		fmt.Printf("Node %v running %s\n", i, loompath)
	}

	loompathAbs2, err := filepath.Abs(loompath2)
	if err != nil {
		return nil, err
	}
	if altValidators > 0 {
		if _, err := os.Stat(loompathAbs2); os.IsNotExist(err) {
			return nil, errors.Errorf("cannot find alternate loom executable %s", loompathAbs2)
		}
	}

	for i := validators; i < validators+altValidators; i++ {
		n := node.NewNode(int64(i), conf.BaseDir, loompathAbs2, conf.ContractDir, genesisTmpl, yamlFile)
		n.LogLevel = *logLevel
		n.LogDestination = *logDest
		n.LogAppDb = *logAppDb
		nodes = append(nodes, n)
		fmt.Printf("Node %v running %s\n", i, loompath2)
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
	go func() {
		errC <- e.Run(nctx, eventC)
	}()

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
	go func() {
		errC <- e.Run(nctx, eventC)
	}()

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
