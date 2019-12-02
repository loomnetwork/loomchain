#!/bin/bash

set -ex

PKG=github.com/loomnetwork/loomchain

# setup temp GOPATH
export GOPATH=/tmp/gopath-$BUILD_TAG
export
export PATH=$GOPATH:$PATH:/var/lib/jenkins/workspace/commongopath/bin:$GOPATH/bin
export PKG_TRANSFER_GATEWAY=github.com/loomnetwork/loomchain/vendor/github.com/loomnetwork/transfer-gateway

LOOM_SRC=$GOPATH/src/$PKG
TG_DIR=$GOPATH/src/$PKG_TRANSFER_GATEWAY
mkdir -p $LOOM_SRC
rsync -r --delete . $LOOM_SRC

if [[ "$OSTYPE" == "linux-gnu" ]]; then
export CGO_CFLAGS="-I/usr/local/include/leveldb"
export CGO_LDFLAGS="-L/usr/local/lib/ -L/usr/lib/x86_64-linux-gnu/ -lsnappy"
#elif [[ "$OSTYPE" == "darwin"* ]]; then #osx
fi

cd $LOOM_SRC
make clean
make get_lint
make deps
make  # on OSX we don't need any C precompiles like cleveldb
make validators-tool

# build the oracles
cd $TG_DIR
PKG=$PKG_TRANSFER_GATEWAY make tgoracle
PKG=$PKG_TRANSFER_GATEWAY make tron_tgoracle
PKG=$PKG_TRANSFER_GATEWAY make loomcoin_tgoracle
PKG=$PKG_TRANSFER_GATEWAY make dposv2_oracle
# move them to the loomchain dir to make post-build steps simpler
mv tgoracle $LOOM_SRC/tgoracle
mv tron_tgoracle $LOOM_SRC/tron_tgoracle
mv loomcoin_tgoracle $LOOM_SRC/loomcoin_tgoracle
# don't care about dpos oracle, don't need to move it

# build the various loom node variants
cd $LOOM_SRC
make basechain
# copy the generic loom binary so it can be published later, the loom binary will be replaced by the
# gateway variant when make loom-gateway executes
cp loom loom-generic
make loom-gateway
cp loom loom-gateway

make loom-cleveldb
make basechain-cleveldb

# lint after building everything
make lint || true
make linterrors

export LOOM_BIN=`pwd`/loom
export LOOM_VALIDATORS_TOOL=`pwd`/e2e/validators-tool

export GORACE="log_path=`pwd`/racelog"
#make loom-race
#make test-race

# export LOOMEXE_PATH="../loom"
# export LOOMEXE_ALTPATH="../loom2"
# export CHECK_APP_HASH="true"
make test

##make test-no-evm
##make no-evm-tests
##make test-app-store-race

#setup & run truffle tests
cd e2e/tests/truffle
yarn

cd ../receipts
bash ./run_truffle_tests.sh
