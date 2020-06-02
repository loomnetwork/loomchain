# [Basechain](https://loomx.io)

Basechain is an interoperable DPoS blockchain that is live in production, EVM-compatible, audited, and battle-tested.


## Prerequisites

* A running Linux or macOS system.
* Go 1.9+
  You can enter the following command to check that Go is installed on your system:

  ```shell
  go version
  ```

  If Go is installed, the output looks like this:

  ```
  go version go1.13.8 darwin/amd64
  ```

  If Go is not installed or you're running an older version, you can refer to the [Getting Started](https://golang.org/doc/install) page of the Go documentation for more details about how you can install or upgrade Go.

* Dep. For details about installing Dep, see the [Dep](https://github.com/golang/dep) page.

* (OPTIONAL) LevelDB.

  If you're running macOS, you can enter the following command to install LevelDB:

  ```shell
  brew install leveldb
  ```

  If you're running Linux, you can enter the following command to install LevelDB:

  ```shell
  apt-get install libleveldb-dev libleveldb1v5
  ```

* The `GOPATH` environment variable is defined.

  Use the following command to check that the `GOPATH` environment variable is defined:

  ```shell
  echo $GOPATH
  ```

  If th `GOPATH` environment variable is defined, the output should look like this:

  ```shell
  /Users/andrei/go
  ```

  See the [GOPATH](https://github.com/golang/go/wiki/GOPATH) page of the Go documentation for more details about the `GOPATH` environment variable.


## Build the binary

1. Set the value of the `LOOM_SRC` environment variable as follows:

  ```shell
  LOOM_SRC=$GOPATH/src/github.com/loomnetwork/loomchain
  ```
2. Clone the repository into the `$LOOM_SRC` directory:

  ```shell
  git clone git@github.com:loomnetwork/loomchain.git $LOOM_SRC
  ```

3. Install dependencies:

  ```shell
  cd $LOOM_SRC && make deps
  ```
4. Build the binary:

  ```shell
  make
  ```

5. Copy the `./loom` binary to a directory of your choice.


## Run

1. Use the following command to initialize the blockchain with the built-in contracts:

  ```shell
  ./loom init
  ```

2. Enter the following command to run the node:

  ```shell
  ./loom run
  ```

<!--
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
-->

## Update Protobuf Messages

Enter the following command to update  protobuf messages:

```shell
# build the Go plugin for protoc
make proto
```

> See the [Go Generated Code](https://developers.google.com/protocol-buffers/docs/reference/go-generated to understand how) of the Google Developer Documentaton for more details about how you can use the generated protobuf messages.

## Useful Links

* [Developer Documentation](https://loomx.io/developers/)