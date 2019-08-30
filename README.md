# [Loom SDK](https://loomx.io)

Loom application specific side chain software development kit.

## Documentation
 
 
[Loom SDK Documentation Site](https://loomx.io/developers/)

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

### To build loomchain with cleveldb
On Mac 
```
brew install leveldb
```
and on Linux
```
apt-get install libleveldb-dev libleveldb1v5
```

## Installing

[Install SDK](https://loomx.io/developers/docs/en/prereqs.html)

## Building
Make sure `GOPATH` is defined and run

```shell
LOOM_SRC=$GOPATH/src/github.com/loomnetwork/loomchain
# clone into gopath
git clone git@github.com:loomnetwork/loomchain.git $LOOM_SRC
# install deps
cd $LOOM_SRC
make deps
make
```

## Running

```shell
# init the blockchain with builtin contracts
./loom init
# run the node
./loom run
```

## Generate keys
Use the genkey command. It will create two files with the given names.
```shell
./loom genkey -a publicKeyFilename -k privateKeyFilename
```
## Ethereum smart contracts
Deploy smart contract with `deploy`
```shell
./loom deploy -a pubkeyFile -k prikeyFile -b contractBytecode.bin
New contract deployed with address:  default:0xB448D7db27192d54FeBdA458B81e7383F8641c8A
Runtime bytecode:  [96 96 96 64 82 96 .... ]
```
Make a call to an already deployed contract with `call`
```
./loom call  -a pubkeyFile -k prikeyFile -i inputDataFile -c 0xB448D7db27192d54FeBdA458B81e7383F8641c8A
Call response:  [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 3 219]
```
Details of encoding contract input data can be found in the [Solidity ABI documentation](https://solidity.readthedocs.io/en/develop/abi-spec.html).
You can use `static-call` similarly to run a read only method.
## Updating Protobuf Messages

```shell
# build the Go plugin for protoc
make proto
```

Read https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how
to use the generated protobuf messages.

Testing
