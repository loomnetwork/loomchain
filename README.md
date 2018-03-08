# [Loom SDK](https://loomx.io)

Loom experimental protobuf test.

## Documentation

`TODO`

## Environment Setup

Requirements

* Go 1.9+

Setup GOPATH
```shell
export GOPATH=`pwd`/gopath:`pwd`/core
```

## Building

```shell
# build the example DAppChain node
go build loom/examples/experiment
# build the example REST server that provides app-specific endpoints for querying data stored
# in the example DAppChain
go build loom/examples/rest-server
```

## Running

Run the node
```shell
./experiment
```

# Updating Protobuf Messages

Install the [`protoc` compiler](https://github.com/google/protobuf/releases),
and run the following commands:

```shell
# build the Go plugin for protoc
go build github.com/gogo/protobuf/protoc-gen-gogo
# regenerate protobufs
export LOOM_SRC=`pwd`/core/src
protoc --plugin=./protoc-gen-gogo -I$LOOM_SRC --gogo_out=$LOOM_SRC loom/loom.proto
protoc --plugin=./protoc-gen-gogo -I$LOOM_SRC --gogo_out=$LOOM_SRC loom/vm/vm.proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

## References

 * [Tendermint Docs](https://tendermint.readthedocs.io/en/latest/)
