# [Loom SDK](https://loomx.io)

Loom application specific side chain software development kit.

## Documentation

`TODO`

## Environment Setup

Requirements

* Go 1.9+

## Installing

`TODO`

## Building

```shell
export LOOM_SRC=$GOPATH/src/github.com/loomnetwork/loom
# clone into gopath
git clone git@github.com:loomnetwork/loom.git $LOOM_SRC
# install deps
cd $LOOM_SRC
dep ensure
# build the example DAppChain node
go build github.com/loomnetwork/loom/examples/experiment
# build the example REST server that provides app-specific endpoints for querying data stored
# in the example DAppChain
go build github.com/loomnetwork/loom/examples/rest-server
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
protoc --plugin=./protoc-gen-gogo -I$GOPATH/src --gogo_out=$GOPATH/src github.com/loomnetwork/loom/loom.proto
protoc --plugin=./protoc-gen-gogo -I$GOPATH/src --gogo_out=$GOPATH/src github.com/loomnetwork/loom/vm/vm.proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

## References

 * [Tendermint Docs](https://tendermint.readthedocs.io/en/latest/)
