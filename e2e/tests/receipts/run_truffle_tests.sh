#!/bin/bash

set -exo pipefail

TEST_DIR=`pwd`

function cleanup {
    bash $TEST_DIR/../cluster.sh --dir $TEST_DIR --stop
}

trap cleanup EXIT
bash ../cluster.sh --init --dir $TEST_DIR --start

cd ../truffle
yarn run gen:eth-key
CLUSTER_DIR=$TEST_DIR/cluster yarn run map-accounts
CLUSTER_DIR=$TEST_DIR/cluster yarn test