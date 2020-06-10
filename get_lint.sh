#!/usr/bin/env bash
set -e

# This file downloads all of the binary dependencies, and checks out a
# specific git hash.
#
# repos it installs:
#   github.com/golangci/golangci-lint

## check if GOPATH is set
if [ -z ${GOPATH+x} ]; then
	echo "The GOPATH environment variable is not set. Please refer to the https://github.com/golang/go/wiki/SettingGOPATH page for more details."
	exit 1
fi

mkdir -p "$GOPATH/src/github.com"
cd "$GOPATH/src/github.com" || exit 1

installFromGithub() {
	repo=$1
	commit=$2
	# optional
	subdir=$3
	echo "--> Installing $repo ($commit)..."
	if [ ! -d "$repo" ]; then
		mkdir -p "$repo"
		git clone "https://github.com/$repo.git" "$repo"
	fi
	if [ ! -z ${subdir+x} ] && [ ! -d "$repo/$subdir" ]; then
		echo "ERROR: no such directory $repo/$subdir"
		exit 1
	fi
	pushd "$repo" && \
		git fetch origin && \
		git checkout -q "$commit" && \
		if [ ! -z ${subdir+x} ]; then cd "$subdir" || exit 1; fi && \
		go install && \
		if [ ! -z ${subdir+x} ]; then cd - || exit 1; fi && \
		popd || exit 1
	echo "--> Done"
	echo ""
}

installFromGithub golangci/golangci-lint v1.18.0 cmd/golangci-lint
cd "$GOPATH/bin" && chmod +x golangci-lint
