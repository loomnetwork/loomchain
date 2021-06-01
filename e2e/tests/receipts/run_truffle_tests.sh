#!/bin/bash

set -exo pipefail

TEST_DIR=`pwd`

function cleanup {
    bash $TEST_DIR/../cluster.sh --dir $TEST_DIR --stop
}

trap cleanup EXIT
bash ../cluster.sh --init --dir $TEST_DIR --start

cd ../truffle
# Wait for all built-in contracts to be deployed to the test cluster.
sleep 5

# Run Truffle tests using Truffle HDWallet provider & /eth endpoint
CLUSTER_DIR=$TEST_DIR/cluster yarn run map-accounts
CLUSTER_DIR=$TEST_DIR/cluster \
TRUFFLE_PROVIDER=hdwallet \
yarn test:hdwallet

# Run Truffle tests using Loom Truffle provider
CLUSTER_DIR=$TEST_DIR/cluster \
TRUFFLE_PROVIDER=loom \
yarn test