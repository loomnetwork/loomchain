# Overview

These example illustrate how to create plugins for the Loom admin CLI.
When the command provided by the `create-tx` plugin is executed a new dummy tx will
be created and committed to the example DAppChain (which needs to be running locally).

# Build

```shell
go build -buildmode=plugin -o out/cmds/create-tx.so examples/cmd-plugins/create-tx/main.go
```

# Run

```shell
./ladmin create-tx <value>
```
