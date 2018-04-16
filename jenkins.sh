#!/bin/bash

set -ex

PKG=github.com/loomnetwork/loom

# setup temp GOPATH
export GOPATH=/tmp/gopath-$BUILD_TAG
export 
export PATH=$GOPATH:$PATH:/var/lib/jenkins/workspace/commongopath/bin

LOOM_SRC=$GOPATH/src/$PKG
mkdir -p $LOOM_SRC
rsync -r --delete . $LOOM_SRC

cd $GOPATH/src/github.com/loomnetwork
git clone github.com/loomnetwork/loom-plugin

cd $LOOM_SRC
dep ensure -vendor-only
go build $PKG/cmd/loom
go test $PKG/...
