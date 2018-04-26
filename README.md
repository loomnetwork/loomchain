# [Loom SDK](https://loomx.io)

Loom application specific side chain software development kit.

## Documentation

`TODO`

## Environment Setup

Requirements

* Go 1.9+
* [Dep](https://github.com/golang/dep)

On Mac
```
brew install dep
```
and on Linux
```
curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
```
## Installing

`TODO`

## Building

Ensure `github.com/loomnetwork/loom-plugin` is in your `GOPATH`, then:

```shell
export LOOM_SRC=$GOPATH/src/github.com/loomnetwork/loom
# clone into gopath
git clone git@github.com:loomnetwork/loom.git $LOOM_SRC
# install deps
cd $LOOM_SRC
make deps
make
# build the example contract
go build -buildmode=plugin -o contracts/helloworld.so plugin/examples/helloworld.go
```

## Running

```shell
# init the blockchain
./loom init
# Copy over example genesis
cp genesis.example.json genesis.json
# run the node
./loom run
```

Run the admin CLI
```shell
./ladmin
```
The admin CLI will load cmd plugins from `out/cmds` by default, this can be overriden
by setting the `LOOM_CMDPLUGINDIR` env var to a different directory.

## Updating Protobuf Messages

```shell
# build the Go plugin for protoc
make proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

## References

 * [Tendermint Docs](https://tendermint.readthedocs.io/en/latest/)
