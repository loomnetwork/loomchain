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

#TODO lets just have a single copy of this stuff
go get golang.org/x/crypto/ed25519
go get google.golang.org/grpc
go get github.com/gogo/protobuf/proto
go get github.com/hashicorp/go-plugin
go get github.com/spf13/cobra
go get github.com/ethereum/go-ethereum

cd $GOPATH/src/github.com/loomnetwork
git clone git@github.com:loomnetwork/loom-plugin.git

cd $LOOM_SRC
dep ensure -vendor-only
go build -tags "evm" $PKG/cmd/loom
go test -tags "evm" $PKG/...
