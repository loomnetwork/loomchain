# Integration test

We provide the integration test engine to do integration test locally. Basically, the engine will create a new cluster of validators with the preset contract setup using `loom` command. The validators run using different ports to avoid conflicts. One multiple validators are setup, we can send a series of commands to test the contract.

For ease of test case, the engine will read the test in `toml` format, looking for `TestCases` entry with `RunCmd` key and execute that command. Of course, the binary intended to run is required. You can find the example of test cases in `dpos-1-validator.toml` for more example test cases.

One you have the test cases, there are 2 ways to test your contract. The first one is to write a Go test test and the second is to use validator tool.

## Integration Test Using Go test

You can find examples of Go test case in `coin_test.go` and `dpos_test.go`.

To run integration test using `go test`, make sure you are in `loomchain` directory and then run (the test will take minutes to compete):
```
go -v test github.com/loomnetwork/loomchain/e2e
```

## Stand Alone Tests Using Validator Tool

You have to get `validators-tool` binary. Run `make validators-tool` in loomchain root directy to build one.

to create a cluster of n nodes with new k accounts, run:
```
./validators-tool new --contract-dir path_to_contracts --loom-path paht_to_loom -n 6 -k 10 -f
```

We have a correct setup for Blueprint and DPOS contract for testing.

### 1. Blueprint 

#### Create a cluster

Build Loom binary and Blueprint internal plugin:
```
make 
make validators-tool
```

The command will create `loom` binary and  `.so` files in `contracts` directory.

Change directory to the path we output `validators-tool` binary:
```
cd ./e2e
```

Create a cluster of 4 validators and 10 generated keys:
```
./validators-tool new --name=blueprint --contract-dir=../contracts --loom-path ../loom --log-level error -n 4 -k 10 -f -g blueprint.genesis.json
```

The command will create all nodes to directory called `blueprint`.


To run the cluster:
```
 ./validators-tool run --conf blueprint/runner.toml 
```

#### Run test cases

Make sure you have `blueprint-cli` binary from [go-loom](https://github.com/loomnetwork/go-loom) copied into current directory. Because `blueprint-cli` will be called from the validator-tool command.

Open a new terminal and run:
```
./dpos-tool test --conf blueprint/runner.toml --test blueprint.toml
```

#### 2. DPOS

#### Create a cluster

To create a cluster of 4 validators and 10 generated keys in the intergration-test directory, run:
```
./validators-tool new --name=dpos --contract-dir=../contracts --loom-path ../loom --log-level error -n 4 -k 10 -f
```

If you want to customize init param for the contract, use -g to profile genesis.json template to the command.
```
./validators-tool new --name=dpos --contract-dir=../contracts --loom-path ../loom --log-level error -n 4 -k 10 -f -g dpos.genesis.json
```

The command will create all nodes to directory called `dpos`.


To run the cluster:
```
 ./validators-tool run --conf dpos/runner.toml 
```

#### Run test cases

Make sure you have `example-cli` binary from [go-loom](https://github.com/loomnetwork/go-loom) copied into current directory. Because `example-cli` will be called from the validator-tool command.

Open a new terminal and run:
```
./validators-tool test --conf dpos/runner.toml --test dpos-4node.toml
```