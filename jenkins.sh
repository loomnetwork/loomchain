#!/bin/bash

set -ex

export GOPATH=`pwd`/build

rm -rf $GOPATH
mkdir -p $GOPATH/src/github.com/loomnetwork
ln -s `pwd` $GOPATH/src/github.com/loomnetwork/loom

go build github.com/loomnetwork/loom/examples/experiment
go build github.com/loomnetwork/loom/examples/rest-server
go test github.com/loomnetwork/loom
