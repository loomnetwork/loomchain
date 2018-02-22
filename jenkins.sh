#!/bin/bash

export GOPATH=`pwd`/gopath:`pwd`/core

go build loom/examples/experiment
go build loom/examples/rest-server