#!/bin/bash

# This script tests backwards compatibility of EVM receipts handling (ReceiptsVersion: 1) between
# loom build 495 and a newer build.
#
# It works by spinning up a cluster using build 495, sending some txs to the cluster Truffle tests,
# then shutting it down, and spinning it back up on the newer build.
#
# The loom binaries are expected to be in the current directory, and should be named loom-495, and
# loom-497
#
# Example usage:
# ./b495_receipts_v1_upgrade_test.sh --v1 `pwd`/loom-495 --v2 `pwd`/loom-497

set -exo pipefail

TEST_DIR=`pwd`
TRUFFLE_DIR=`pwd`/../truffle
CLUSTER_RUNNING=false
CLUSTER=`pwd`/../cluster.sh
LOOM_BIN_V1=`pwd`/loom-495
LOOM_BIN_V2=`pwd`/loom-497

while [[ "$#" > 0 ]]; do case $1 in
  --v1) LOOM_BIN_V1=$2; shift; shift;;
  --v2) LOOM_BIN_V2=$2; shift; shift;;
  *) echo "Unknown parameter: $1"; shift; shift;;
esac; done

function stop_cluster {
    bash $CLUSTER --dir $TEST_DIR --stop
    CLUSTER_RUNNING=false
}

function cleanup {
    if [[ "$CLUSTER_RUNNING" == true ]]; then
        stop_cluster
    fi
}

trap cleanup EXIT

rm -rf $TEST_DIR/cluster

echo "Spinning up cluster with $LOOM_BIN_V1"
LOOM_BIN=$LOOM_BIN_V1 \
bash $CLUSTER --init --dir $TEST_DIR --start --cfg `pwd`/b495_receipts_v1_loom.yml

CLUSTER_RUNNING=true

pushd $TRUFFLE_DIR
CLUSTER_DIR=$TEST_DIR/cluster yarn test
popd

# give the nodes a bit of time to sync up
sleep 5

stop_cluster

# give the nodes a bit of time to shut down
sleep 5

for i in `seq 0 3`;
do
    pushd $TEST_DIR/cluster/${i}
    # stash logs for later
    mv loom.log loom1.log
    popd
done

# start the cluster up again with the new loom build
echo "Spinning up cluster with $LOOM_BIN_V2"
LOOM_BIN=$LOOM_BIN_V2 \
bash $CLUSTER --start
CLUSTER_RUNNING=true

# give the nodes a bit of time to sync up
sleep 5

# check the cluster is operational
pushd $TRUFFLE_DIR
CLUSTER_DIR=$TEST_DIR/cluster yarn test
popd

# give the nodes a bit of time digest
sleep 1
