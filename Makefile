PKG = github.com/loomnetwork/loomchain
GIT_SHA = `git rev-parse --verify HEAD`
GOFLAGS = -tags "evm" -ldflags "-X $(PKG).Build=$(BUILD_NUMBER) -X $(PKG).GitSHA=$(GIT_SHA)"
GOFLAGS_NOEVM = -ldflags "-X $(PKG).Build=$(BUILD_NUMBER) -X $(PKG).GitSHA=$(GIT_SHA)"
PROTOC = protoc --plugin=./protoc-gen-gogo -Ivendor -I$(GOPATH)/src -I/usr/local/include
PLUGIN_DIR = $(GOPATH)/src/github.com/loomnetwork/go-loom
GOGO_PROTOBUF_DIR = $(GOPATH)/src/github.com/gogo/protobuf
GO_ETHEREUM_DIR = $(GOPATH)/src/github.com/ethereum/go-ethereum

.PHONY: all clean test install deps proto builtin oracles tgoracle plasmacash-oracle

all: loom builtin

oracles: tgoracle plasmacash-oracle

builtin: contracts/coin.so.1.0.0 contracts/dpos.so.1.0.0 contracts/plasmacash.so.1.0.0

contracts/coin.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/coin/plugin

contracts/dpos.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/dpos/plugin

contracts/plasmacash.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/plasma_cash/plugin

tgoracle:
	go build $(GOFLAGS) -o $@ $(PKG)/cmd/$@

plasmacash-oracle:
	go build -v $(GOFLAGS) -o $@ $(PKG)/builtin/plugins/plasma_cash/cmd/oracle

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

deps: $(PLUGIN_DIR)
	cd $(PLUGIN_DIR) && git pull
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
		github.com/miguelmota/go-solidity-sha3
	# checkout the last commit before the dev branch was merged into master (and screwed everything up)
	cd $(GOGO_PROTOBUF_DIR) && git checkout 1ef32a8b9fc3f8ec940126907cedb5998f6318e4
	# use a modified stateObject for EVM calls
	cd $(GO_ETHEREUM_DIR) && git checkout bab696378c359c56640fae48dfd3132763dbc64b
	# fetch vendored packages
	dep ensure -vendor-only

test: proto
	go test -timeout 20m -v $(GOFLAGS) $(PKG)/...

test-no-evm: proto
	go test -timeout 20m -v $(GOFLAGS_NOEVM) $(PKG)/...

test-e2e:
	go test -timeout 20m -v $(PKG)/e2e

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
		contracts/plasmacash.so.1.0.0 \
		plasmacash-oracle

