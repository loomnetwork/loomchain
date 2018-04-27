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
make deps
make
make test
