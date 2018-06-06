# Integration Tests

Create a cluster of n nodes with new k accounts:
```
./validators-tool new --contract-dir path_to_contracts --loom-path paht_to_loom -n 6 -k 3 -f
```

## Blueprint 

### Create a cluster

Build Loom binary and Blueprint internal plugin:
```
make 
make validators-tool
```

The command will create `loom` binary and  `.so` files in `contracts` directory.

Change directory to the path we output `validators-tool` binary:
```
cd ./integration-test
```

Create a cluster of 4 validators and 3 generated keys:
```
./validators-tool new --name=blueprint --contract-dir=../contracts --loom-path ../loom --log-level error -n 4 -k 3 -f -g blueprint.genesis.json
```

The command will create all nodes to directory called `blueprint`.


To run the cluster:
```
 ./validators-tool run --conf blueprint/runner.toml 
```

### Run test cases

Make sure you have `blueprint-cli` binary from [go-loom](https://github.com/loomnetwork/go-loom) copied into current directory. Because `blueprint-cli` will be called from the validator-tool command.

Open a new terminal and run:
```
./dpos-tool test --conf blueprint/runner.toml --test blueprint.toml
```

### DPOS

### Create a cluster

To create a cluster of 4 validators and 3 generated keys in the intergration-test directory, run:
```
./validators-tool new --name=dpos --contract-dir=../contracts --loom-path ../loom --log-level error -n 4 -k 3 -f
```

The command will create all nodes to directory called `dpos`.


To run the cluster:
```
 ./validators-tool run --conf dpos/runner.toml 
```

### Run test cases

Make sure you have `example-cli` binary from [go-loom](https://github.com/loomnetwork/go-loom) copied into current directory. Because `example-cli` will be called from the validator-tool command.

Open a new terminal and run:
```
./validators-tool test --conf dpos/runner.toml --test dpos-4node.toml
```