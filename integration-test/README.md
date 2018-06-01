# Integration Tests

Create a cluster of n nodes with new k accounts:
```
./dpos-tool new --contract-dir path_to_contracts --loom-path paht_to_loom -n 6 -k 3 -f
./dpos-tool run
```

Make sure you have `example-cli` binary from [go-loom](https://github.com/loomnetwork/go-loom) copied into current directory

Open a new terminal and run:
```
./dpos-tool test
```

