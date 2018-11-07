#!/bin/bash

set -ex

PKG=github.com/loomnetwork/loomchain

# setup temp GOPATH
export GOPATH=/tmp/gopath-$BUILD_TAG
export
export PATH=$GOPATH:$PATH:/var/lib/jenkins/workspace/commongopath/bin

LOOM_SRC=$GOPATH/src/$PKG
mkdir -p $LOOM_SRC
rsync -r --delete . $LOOM_SRC

cd $LOOM_SRC
make clean
make deps
make
make validators-tool
make tgoracle

export LOOM_BIN=`pwd`/loom
export LOOM_VALIDATORS_TOOL=`pwd`/e2e/validators-tool

#make test
#make test-no-evm
go test -timeout 20m -v -tags "evm" github.com/loomnetwork/loomchain/e2e -run ^TestContractCoin$

# setup & run truffle tests
cd e2e/tests/truffle
yarn

cd ../receipts
bash ./run_truffle_tests.sh
