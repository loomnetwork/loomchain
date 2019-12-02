#!/bin/bash

# This script can be used to spin up a local cluster for testing.

set -exo pipefail

echo "LOOM_BIN = ${LOOM_BIN}"

INIT_CLUSTER=false
START_CLUSTER=false
STOP_CLUSTER=false
CLUSTER_DIR=`pwd`
CLUSTER_GENESIS="genesis.json"
CLUSTER_CONFIG="loom.yml"


while [[ "$#" > 0 ]]; do case $1 in
  -i|--init) INIT_CLUSTER=true; shift;;
  -g|--genesis) CLUSTER_GENESIS=$2; shift; shift;;
  -d|--dir) CLUSTER_DIR=$2; shift; shift;;
  -c|--cfg) CLUSTER_CONFIG=$2; shift; shift;;
  --start) START_CLUSTER=true; shift;;
  --stop) STOP_CLUSTER=true; shift;;
  *) echo "Unknown parameter: $1"; shift; shift;;
esac; done


function init_cluster {
    cd $CLUSTER_DIR
    
    if [[ -n "$CLUSTER_GENESIS" ]]; then
        FLAGS="-g $CLUSTER_GENESIS"
    fi

    if [[ -n "$CLUSTER_CONFIG" ]]; then
        FLAGS="$FLAGS -c $CLUSTER_CONFIG"
    fi

    $LOOM_VALIDATORS_TOOL new \
        $FLAGS \
        --base-dir $CLUSTER_DIR \
        --contract-dir "" \
        --name cluster \
        --loom-path $LOOM_BIN \
        --force
    
    echo "Initialized cluster in ${CLUSTER_DIR}/cluster"
}

function run_cluster {
    cd $CLUSTER_DIR
    
    $LOOM_VALIDATORS_TOOL run --conf cluster/runner.toml > cluster.log 2>&1 &
    LOOM_PID=$!
    echo $LOOM_PID > $CLUSTER_DIR/loom.pid
}

function stop_cluster {
    if [[ -z "$LOOM_PID" ]]; then
        LOOM_PID=`cat ${CLUSTER_DIR}/loom.pid`
        rm -f $CLUSTER_DIR/loom.pid
    fi
    if [[ -n "$LOOM_PID" ]]; then
        echo "Stopping cluster (${LOOM_PID})"
        kill -15 "${LOOM_PID}" &> /dev/null
    fi
    STOP_CLUSTER=false
}

if [[ "$INIT_CLUSTER" == true ]]; then
    init_cluster
fi

if [[ "$START_CLUSTER" == true ]]; then
    run_cluster
fi

if [[ "$STOP_CLUSTER" == true ]]; then
    stop_cluster
fi
