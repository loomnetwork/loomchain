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

Ensure `github.com/loomnetwork/go-loom` is in your `GOPATH`, then:

```shell
export LOOM_SRC=$GOPATH/src/github.com/loomnetwork/loomchain
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

## Ethereum smart contracts
Deploy smart contract
```shell
./loom deploy -a pubkeyFile -k prikeyFile -b contractBytecode.bin
New contract deployed with address:  default:0xB448D7db27192d54FeBdA458B81e7383F8641c8A
Runtime bytecode:  [96 96 96 64 82 96 .... ]
```
Make a call to an already deployed contract
```
./loom  -a pubkeyFile -k prikeyFile -i inputDataFile -c default:0xB448D7db27192d54FeBdA458B81e7383F8641c8A
Call response:  [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 3 219]
```
Details of encoding contract input data can be found in the [Solidity ABI documentation](https://solidity.readthedocs.io/en/develop/abi-spec.html).

## Updating Protobuf Messages

```shell
# build the Go plugin for protoc
make proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

## References

 * [Tendermint Docs](https://tendermint.readthedocs.io/en/latest/)
