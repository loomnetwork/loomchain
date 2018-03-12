#!/bin/bash

set -ex

PKG=github.com/loomnetwork/loom

# setup temp GOPATH
export GOPATH=/tmp/gopath-$BUILD_TAG
export PATH=$GOPATH:$PATH

LOOM_SRC=$GOPATH/src/$PKG
mkdir -p $LOOM_SRC
rsync -r --delete . $LOOM_SRC

go get github.com/tools/godep

cd $LOOM_SRC
dep ensure -vendor-only
go build $PKG/examples/helloworld
go build $PKG/examples/rest-server
go test $PKG/...
