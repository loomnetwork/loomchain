PKG = github.com/loomnetwork/loomchain
GIT_SHA = `git rev-parse --verify HEAD`
GOFLAGS = -tags "evm" -ldflags "-X $(PKG).Build=$(BUILD_NUMBER) -X $(PKG).GitSHA=$(GIT_SHA)"
GOFLAGS_NOEVM = -ldflags "-X $(PKG).Build=$(BUILD_NUMBER) -X $(PKG).GitSHA=$(GIT_SHA)"
PROTOC = protoc --plugin=./protoc-gen-gogo -Ivendor -I$(GOPATH)/src -I/usr/local/include
PLUGIN_DIR = $(GOPATH)/src/github.com/loomnetwork/go-loom
GOLANG_PROTOBUF_DIR = $(GOPATH)/src/github.com/golang/protobuf
GOGO_PROTOBUF_DIR = $(GOPATH)/src/github.com/gogo/protobuf
GO_ETHEREUM_DIR = $(GOPATH)/src/github.com/ethereum/go-ethereum

.PHONY: all clean test install deps proto builtin oracles tgoracle loomcoin_tgoracle pcoracle

all: loom builtin

oracles: tgoracle pcoracle

builtin: contracts/coin.so.1.0.0 contracts/dpos.so.1.0.0 contracts/dpos.so.2.0.0 contracts/plasmacash.so.1.0.0

contracts/coin.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/coin/plugin

contracts/dpos.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/dpos/plugin

contracts/dpos.so.2.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/dposv2/plugin

contracts/plasmacash.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/plasma_cash/plugin

tgoracle:
	go build $(GOFLAGS) -o $@ $(PKG)/cmd/$@

loomcoin_tgoracle:
	go build $(GOFLAGS) -o $@ $(PKG)/cmd/$@

pcoracle:
	go build $(GOFLAGS) -o $@ $(PKG)/cmd/$@

loom: proto
	go build $(GOFLAGS) $(PKG)/cmd/$@

install: proto
	go install $(GOFLAGS) $(PKG)/cmd/loom

protoc-gen-gogo:
	go build github.com/gogo/protobuf/protoc-gen-gogo

%.pb.go: %.proto protoc-gen-gogo
	if [ -e "protoc-gen-gogo.exe" ]; then mv protoc-gen-gogo.exe protoc-gen-gogo; fi
	$(PROTOC) --gogo_out=$(GOPATH)/src $(PKG)/$<

proto: registry/registry.pb.go

$(PLUGIN_DIR):
	git clone -q git@github.com:loomnetwork/go-loom.git $@

$(GO_ETHEREUM_DIR):
	git clone -q git@github.com:loomnetwork/go-ethereum.git $@

validators-tool:
	go build -o e2e/validators-tool $(PKG)/e2e/cmd

deps: $(PLUGIN_DIR) $(GO_ETHEREUM_DIR)
	go get \
		golang.org/x/crypto/ed25519 \
		google.golang.org/grpc \
		github.com/gogo/protobuf/gogoproto \
		github.com/gogo/protobuf/proto \
		github.com/hashicorp/go-plugin \
		github.com/spf13/cobra \
		github.com/spf13/pflag \
		github.com/go-kit/kit/log \
		github.com/grpc-ecosystem/go-grpc-prometheus \
		github.com/prometheus/client_golang/prometheus \
		github.com/go-kit/kit/log \
		github.com/BurntSushi/toml \
		github.com/ulule/limiter \
		github.com/loomnetwork/mamamerkle \
		github.com/miguelmota/go-solidity-sha3 \
		github.com/certusone/yubihsm-go \
		golang.org/x/sys/cpu
	cd $(PLUGIN_DIR) && git checkout v2 && git pull origin v2
	cd $(GOLANG_PROTOBUF_DIR) && git checkout v1.1.0
	# checkout the last commit before the dev branch was merged into master (and screwed everything up)
	cd $(GOGO_PROTOBUF_DIR) && git checkout v1.1.1
	# use a modified stateObject for EVM calls
	cd $(GO_ETHEREUM_DIR) && git checkout bab696378c359c56640fae48dfd3132763dbc64b
	# fetch vendored packages
	dep ensure -vendor-only

#TODO we should turn back vet on, it broke when we upgraded go versions
test: proto
	go test -timeout 20m -v -vet=off $(GOFLAGS) $(PKG)/...

test-no-evm: proto
	go test -timeout 20m -v -vet=off $(GOFLAGS_NOEVM) $(PKG)/...

test-e2e:
	go test -timeout 20m -v -vet=off $(PKG)/e2e

vet:
	go vet ./...

vet-evm:
	go vet -tags evm ./...

clean:
	go clean
	rm -f \
		loom \
		protoc-gen-gogo \
		contracts/coin.so.1.0.0 \
		contracts/dpos.so.1.0.0 \
		contracts/dpos.so.2.0.0 \
		contracts/plasmacash.so.1.0.0 \
		pcoracle
