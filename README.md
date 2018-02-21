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

```bash
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

```bash
# build the Go plugin for protoc
go build github.com/gogo/protobuf/protoc-gen-gogo
# regenerate protobufs
protoc --plugin=./protoc-gen-gogo --gogo_out=./core/src/loom -I./core/src/loom ./core/src/loom/loom.proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

## References

 * [Tendermint Docs](https://tendermint.readthedocs.io/en/latest/)
