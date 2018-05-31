# Integration Tests

Create a cluster of n nodes with new k accounts:
```
./dpos-tool new --contract-dir path_to_contracts --loom-path paht_to_loom -n 6 -k 3
./dpos-tool run
```


Open a new terminal and run:
```
./dpos-tool test
```

