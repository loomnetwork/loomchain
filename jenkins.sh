#!/bin/bash

set -ex

PKG=github.com/loomnetwork/loomchain

# Set up a temporary `GOPATH` environment variable
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
make  # On macOS, we don't need any C precompiles like cleveldb
make validators-tool

# Build the oracles
cd $TG_DIR
PKG=$PKG_TRANSFER_GATEWAY make tgoracle
PKG=$PKG_TRANSFER_GATEWAY make tron_tgoracle
PKG=$PKG_TRANSFER_GATEWAY make loomcoin_tgoracle
PKG=$PKG_TRANSFER_GATEWAY make dposv2_oracle
# To simplify post-build steps, we move the oracles to the loomchain directory
mv tgoracle $LOOM_SRC/tgoracle
mv tron_tgoracle $LOOM_SRC/tron_tgoracle
mv loomcoin_tgoracle $LOOM_SRC/loomcoin_tgoracle
# We do not need to move the DPoS oracle

# Build the various loom node variants
cd $LOOM_SRC
make basechain
# Copy the generic loom binary so we can publish it later.
cp loom loom-generic
# Build the `loom-gateway` variant
make loom-gateway
# Replace the generic loom binary with the `loom-gateway` variant
cp loom loom-gateway

make loom-cleveldb
make basechain-cleveldb

# Lint after we've built everything
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
