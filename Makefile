GOLint = \
	github.com/golangci/golangci-lint/cmd/golangci-lint \

PKG = github.com/loomnetwork/loomchain
PKG_GAMECHAIN = github.com/loomnetwork/gamechain
PKG_BATTLEGROUND = $(PKG_GAMECHAIN)/battleground
# Allow location of transfer-gateway package to be overriden via env var
PKG_TRANSFER_GATEWAY?=github.com/loomnetwork/transfer-gateway
PKG_BINANCE_TGORACLE=github.com/loomnetwork/binance-tgoracle

PROTOC = protoc --plugin=./protoc-gen-gogo -Ivendor -I$(GOPATH)/src

PLUGIN_DIR = $(GOPATH)/src/github.com/loomnetwork/go-loom
GOLANG_PROTOBUF_DIR = $(GOPATH)/src/github.com/golang/protobuf
GENPROTO_DIR = $(GOPATH)/src/google.golang.org/genproto
YUBIHSM_DIR = $(GOPATH)/src/github.com/certusone/yubihsm-go
GOGO_PROTOBUF_DIR = $(GOPATH)/src/github.com/gogo/protobuf
GRPC_DIR = $(GOPATH)/src/google.golang.org/grpc
GO_ETHEREUM_DIR = $(GOPATH)/src/github.com/ethereum/go-ethereum
SSHA3_DIR = $(GOPATH)/src/github.com/miguelmota/go-solidity-sha3
HASHICORP_DIR = $(GOPATH)/src/github.com/hashicorp/go-plugin
GO_TESTING_INTERFACE_DIR = $(GOPATH)/src/github.com/mitchellh/go-testing-interface
LEVIGO_DIR = $(GOPATH)/src/github.com/jmhodges/levigo
GAMECHAIN_DIR = $(GOPATH)/src/github.com/loomnetwork/gamechain
BTCD_DIR = $(GOPATH)/src/github.com/btcsuite/btcd
PROMETHEUS_PROCFS_DIR=$(GOPATH)/src/github.com/prometheus/procfs
TRANSFER_GATEWAY_DIR=$(GOPATH)/src/$(PKG_TRANSFER_GATEWAY)
BINANCE_TGORACLE_DIR=$(GOPATH)/src/$(PKG_BINANCE_TGORACLE)

# NOTE: To build on Jenkins using a custom go-loom branch update the `deps` target below to checkout
#       that branch, you only need to update GO_LOOM_GIT_REV if you wish to lock the build to a
#       specific commit.
GO_LOOM_GIT_REV = HEAD
# Specifies the loomnetwork/transfer-gateway branch/revision to use.
TG_GIT_REV = HEAD
# loomnetwork/go-ethereum loomchain branch
ETHEREUM_GIT_REV = 6128fa1a8c767035d3da6ef0c27ebb7778ce3713
# use go-plugin we get 'timeout waiting for connection info' error
HASHICORP_GIT_REV = f4c3476bd38585f9ec669d10ed1686abd52b9961
LEVIGO_GIT_REV = c42d9e0ca023e2198120196f842701bb4c55d7b9
BTCD_GIT_REV = 7d2daa5bfef28c5e282571bc06416516936115ee
# This is locked down to this particular revision because this is the last revision before the
# google.golang.org/genproto was recompiled with a new version of protoc, which produces pb.go files
# that don't appear to be compatible with the gogo protobuf & protoc versions we use.
# google.golang.org/genproto seems to be pulled in by the grpc package.
GENPROTO_GIT_REV = b515fa19cec88c32f305a962f34ae60068947aea
# Specifies the loomnetwork/binance-tgoracle branch/revision to use.
BINANCE_TG_GIT_REV = HEAD
# Lock down certusone/yubihsm-go revision
YUBIHSM_REV = 892fb9b370f3cbb486fc1f53d4a1d89e9f552af0

BUILD_DATE = `date -Iseconds`
GIT_SHA = `git rev-parse --verify HEAD`
GO_LOOM_GIT_SHA = `cd ${PLUGIN_DIR} && git rev-parse --verify ${GO_LOOM_GIT_REV}`
TG_GIT_SHA = `cd ${TRANSFER_GATEWAY_DIR} && git rev-parse --verify ${TG_GIT_REV}`
ETHEREUM_GIT_SHA = `cd ${GO_ETHEREUM_DIR} && git rev-parse --verify ${ETHEREUM_GIT_REV}`
HASHICORP_GIT_SHA = `cd ${HASHICORP_DIR} && git rev-parse --verify ${HASHICORP_GIT_REV}`
GAMECHAIN_GIT_SHA = `cd ${GAMECHAIN_DIR} && git rev-parse --verify HEAD`
BTCD_GIT_SHA = `cd ${BTCD_DIR} && git rev-parse --verify ${BTCD_GIT_REV}`

GOFLAGS_BASE = \
	-X $(PKG).Build=$(BUILD_NUMBER) \
	-X $(PKG).GitSHA=$(GIT_SHA) \
	-X $(PKG).GoLoomGitSHA=$(GO_LOOM_GIT_SHA) \
	-X $(PKG).EthGitSHA=$(ETHEREUM_GIT_SHA) \
	-X $(PKG).HashicorpGitSHA=$(HASHICORP_GIT_SHA) \
	-X $(PKG).BtcdGitSHA=$(BTCD_GIT_SHA)
GOFLAGS = -tags "evm" -ldflags "$(GOFLAGS_BASE)"
GOFLAGS_GAMECHAIN_BASE = -X $(PKG_BATTLEGROUND).BuildDate=$(BUILD_DATE) -X $(PKG_BATTLEGROUND).BuildGitSha=$(GAMECHAIN_GIT_SHA) -X $(PKG_BATTLEGROUND).BuildNumber=$(BUILD_NUMBER)
GOFLAGS_GAMECHAIN = -tags "evm gamechain" -ldflags "$(GOFLAGS_BASE) $(GOFLAGS_GAMECHAIN_BASE)"
GOFLAGS_GATEWAY = -tags "evm gateway" -ldflags "$(GOFLAGS_BASE) -X $(PKG).TransferGatewaySHA=$(TG_GIT_SHA) -X $(PKG).BuildVariant=gateway"
GOFLAGS_BASECHAIN = -tags "evm basechain gateway" -ldflags "$(GOFLAGS_BASE) -X $(PKG).TransferGatewaySHA=$(TG_GIT_SHA) -X $(PKG).BuildVariant=basechain"
GOFLAGS_BASECHAIN_CLEVELDB = -tags "evm basechain gateway gcc" -ldflags "$(GOFLAGS_BASE) -X $(PKG).TransferGatewaySHA=$(TG_GIT_SHA) -X $(PKG).BuildVariant=basechain"
GOFLAGS_CLEVELDB = -tags "evm gcc" -ldflags "$(GOFLAGS_BASE)"
GOFLAGS_GAMECHAIN_CLEVELDB = -tags "evm gamechain gcc" -ldflags "$(GOFLAGS_BASE) $(GOFLAGS_GAMECHAIN_BASE)"
GOFLAGS_NOEVM = -ldflags "$(GOFLAGS_BASE)"

WINDOWS_BUILD_VARS = CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=amd64 BIN_EXTENSION=.exe

E2E_TESTS_TIMEOUT = 39m

.PHONY: all clean test install get_lint update_lint deps proto builtin oracles tgoracle loomcoin_tgoracle bsc_tgoracle tron_tgoracle binance_tgoracle pcoracle dposv2_oracle basechain-cleveldb loom-cleveldb lint

all: loom builtin

oracles: tgoracle pcoracle bsc_tgoracle

builtin: contracts/coin.so.1.0.0 contracts/dpos.so.2.0.0 contracts/dpos.so.3.0.0 contracts/plasmacash.so.1.0.0

contracts/coin.so.1.0.0:
	go build -buildmode=plugin -o $@ $(GOFLAGS) $(PKG)/builtin/plugins/coin/plugin

contracts/dpos.so.2.0.0:
	go build -buildmode=plugin -o $@ $(GOFLAGS) $(PKG)/builtin/plugins/dposv2/plugin

contracts/dpos.so.3.0.0:
	go build -buildmode=plugin -o $@ $(GOFLAGS) $(PKG)/builtin/plugins/dposv3/plugin

contracts/plasmacash.so.1.0.0:
	go build -buildmode=plugin -o $@ $(GOFLAGS) $(PKG)/builtin/plugins/plasma_cash/plugin

tgoracle: $(TRANSFER_GATEWAY_DIR)
	cd $(TRANSFER_GATEWAY_DIR) && make tgoracle

loomcoin_tgoracle: $(TRANSFER_GATEWAY_DIR)
	cd $(TRANSFER_GATEWAY_DIR) && make loomcoin_tgoracle

bsc_tgoracle: $(TRANSFER_GATEWAY_DIR)
	cd $(TRANSFER_GATEWAY_DIR) && make bsc_tgoracle

tron_tgoracle: $(TRANSFER_GATEWAY_DIR)
	cd $(TRANSFER_GATEWAY_DIR) && make tron_tgoracle

binance_tgoracle: $(BINANCE_TGORACLE_DIR)
	cd $(BINANCE_TGORACLE_DIR) && make binance_tgoracle

pcoracle:
	go build $(GOFLAGS) -o $@ $(PKG)/cmd/$@

loom: proto
	go build $(GOFLAGS) $(PKG)/cmd/$@

loom-gateway: proto $(TRANSFER_GATEWAY_DIR)
	go build $(GOFLAGS_GATEWAY) $(PKG)/cmd/loom

loom-windows:
	$(WINDOWS_BUILD_VARS) make loom

gamechain: proto
	go build $(GOFLAGS_GAMECHAIN) -o gamechain$(BIN_EXTENSION) $(PKG)/cmd/loom

gamechain-cleveldb: proto  c-leveldb
	go build $(GOFLAGS_GAMECHAIN_CLEVELDB) -o gamechain$(BIN_EXTENSION) $(PKG)/cmd/loom

gamechain-windows: proto
	$(WINDOWS_BUILD_VARS) make gamechain

loom-cleveldb: proto c-leveldb
	go build $(GOFLAGS_CLEVELDB) -o $@ $(PKG)/cmd/loom

basechain: proto $(TRANSFER_GATEWAY_DIR)
	go build $(GOFLAGS_BASECHAIN) -o $@ $(PKG)/cmd/loom

basechain-cleveldb: proto c-leveldb $(TRANSFER_GATEWAY_DIR)
	go build $(GOFLAGS_BASECHAIN_CLEVELDB) -o $@ $(PKG)/cmd/loom

basechain-windows:
	$(WINDOWS_BUILD_VARS) make basechain

loom-race: proto
	go build -race $(GOFLAGS) -o loom-race $(PKG)/cmd/loom

install: proto
	go install $(GOFLAGS) $(PKG)/cmd/loom

protoc-gen-gogo:
	which protoc
	protoc --version
	go build github.com/gogo/protobuf/protoc-gen-gogo

%.pb.go: %.proto protoc-gen-gogo
	if [ -e "protoc-gen-gogo.exe" ]; then mv protoc-gen-gogo.exe protoc-gen-gogo; fi
	$(PROTOC) --gogo_out=$(GOPATH)/src $(PKG)/$<

get_lint:
	@echo "--> Installing lint"
	chmod +x get_lint.sh
	./get_lint.sh

update_lint:
	@echo "--> Updating lint"
	./get_lint.sh

lint:
	$(GOPATH)/bin/golangci-lint run --build-tags="evm gateway" | tee lintreport

linterrors:
	chmod +x parselintreport.sh
	./parselintreport.sh

proto: registry/registry.pb.go

c-leveldb:
	go get github.com/jmhodges/levigo
	cd $(LEVIGO_DIR) && git checkout master && git pull && git checkout $(LEVIGO_GIT_REV)

$(PLUGIN_DIR):
	git clone -q git@github.com:loomnetwork/go-loom.git $@

$(GO_ETHEREUM_DIR):
	git clone -q git@github.com:loomnetwork/go-ethereum.git $@

$(SSHA3_DIR):
	git clone -q git@github.com:loomnetwork/go-solidity-sha3.git $@

$(TRANSFER_GATEWAY_DIR):
	git clone -q git@github.com:loomnetwork/transfer-gateway.git $@
	cd $(TRANSFER_GATEWAY_DIR) && git checkout master && git pull && git checkout $(TG_GIT_REV)

$(BINANCE_TGORACLE_DIR):
	git clone -q git@github.com:loomnetwork/binance-tgoracle.git $@
	cd $(BINANCE_TGORACLE_DIR) && git checkout master && git pull && git checkout $(BINANCE_TG_GIT_REV)

validators-tool: $(TRANSFER_GATEWAY_DIR)
	go build -tags gateway -o e2e/validators-tool $(PKG)/e2e/cmd

deps: $(PLUGIN_DIR) $(GO_ETHEREUM_DIR) $(SSHA3_DIR)
	# Temp workaround for https://github.com/prometheus/procfs/issues/221
	git clone -q git@github.com:prometheus/procfs $(PROMETHEUS_PROCFS_DIR)  ; true
	cd $(PROMETHEUS_PROCFS_DIR) && git checkout master && git pull && git checkout d3b299e382e6acf1baa852560d862eca4ff643c8
	# Lock down Prometheus golang client to v1.2.1 (newer versions use a different protobuf version)
	git clone -q git@github.com:prometheus/client_golang $(GOPATH)/src/github.com/prometheus/client_golang ; true
	cd $(GOPATH)/src/github.com/prometheus/client_golang && git checkout master && git pull && git checkout v1.2.1
	# prometheus/client_model is pulled by prometheus/client_golang so lock it down as well
	git clone -q git@github.com:prometheus/client_model $(GOPATH)/src/github.com/prometheus/client_model ; true
	cd $(GOPATH)/src/github.com/prometheus/client_model && git checkout master && git pull && git checkout 14fe0d1b01d4d5fc031dd4bec1823bd3ebbe8016
	# prometheus/common is pulled by prometheus/client_golang so lock it down as well
	git clone -q git@github.com:prometheus/common $(GOPATH)/src/github.com/prometheus/common ; true
	cd $(GOPATH)/src/github.com/prometheus/common && git checkout main && git pull && git checkout v0.7.0
	git clone -q git@github.com:googleapis/go-genproto.git $(GENPROTO_DIR); true
	cd $(GENPROTO_DIR) && git checkout master && git pull && git checkout $(GENPROTO_GIT_REV)

	export GO111MODULE=off
#		google.golang.org/grpc \	
	go get \
		golang.org/x/crypto/ed25519 \
		github.com/gogo/protobuf/gogoproto \
		github.com/gogo/protobuf/proto \
		github.com/spf13/cobra \
		github.com/spf13/pflag \
		github.com/go-kit/kit/log \
		github.com/grpc-ecosystem/go-grpc-prometheus \
		github.com/BurntSushi/toml \
		github.com/ulule/limiter \
		github.com/loomnetwork/mamamerkle \
		golang.org/x/sys/cpu \
		github.com/certusone/yubihsm-go \
		github.com/gorilla/websocket \
		github.com/phonkee/go-pubsub \
		github.com/inconshreveable/mousetrap \
		github.com/posener/wstest \
		github.com/hashicorp/go-hclog \
		github.com/hashicorp/yamux \
		github.com/oklog/run

	# When you want to reference a different branch of go-loom change GO_LOOM_GIT_REV above
	cd $(PLUGIN_DIR) && git checkout master && git pull && git checkout $(GO_LOOM_GIT_REV)
	git clone -q git@github.com:golang/protobuf.git $(GOPATH)/src/github.com/golang/protobuf ; true
	cd $(GOLANG_PROTOBUF_DIR) && git checkout v1.1.0
	git clone -q git@github.com:gogo/protobuf.git $(GOGO_PROTOBUF_DIR); true
	cd $(GOGO_PROTOBUF_DIR) && git checkout v1.1.1
	git clone -q git@github.com:grpc/grpc-go.git $(GRPC_DIR); true
	cd $(GRPC_DIR) && git checkout v1.20.1
	cd $(GO_ETHEREUM_DIR) && git checkout master && git pull && git checkout $(ETHEREUM_GIT_REV) && rm -rf crypto/bn256 && git checkout master crypto/bn256
	git clone -q git@github.com:hashicorp/go-plugin.git $(HASHICORP_DIR); true
	cd $(HASHICORP_DIR) && git checkout $(HASHICORP_GIT_REV)
	# go-testing-interface is a dependency of hashicorp/go-plugin,
	# latest version of go-testing-interface only supports Go 1.14+ so use an older version
	git clone -q git@github.com:mitchellh/go-testing-interface.git $(GO_TESTING_INTERFACE_DIR); true
	cd $(GO_TESTING_INTERFACE_DIR) && git checkout v1.0.0
	git clone -q git@github.com:btcsuite/btcd.git $(BTCD_DIR); true
	cd $(BTCD_DIR) && git checkout $(BTCD_GIT_REV)
	cd $(YUBIHSM_DIR) && git checkout master && git pull && git checkout $(YUBIHSM_REV)
	# fetch vendored packages
	dep ensure -vendor-only

#TODO we should turn back vet on, it broke when we upgraded go versions
test: proto
	go test  -failfast -timeout $(E2E_TESTS_TIMEOUT) -v -vet=off $(GOFLAGS) $(PKG)/...

test-race: proto
	go test -race -failfast -timeout $(E2E_TESTS_TIMEOUT) -v -vet=off $(GOFLAGS) $(PKG)/...

test-no-evm: proto
	go test -failfast -timeout $(E2E_TESTS_TIMEOUT) -v -vet=off $(GOFLAGS_NOEVM) $(PKG)/...

# Only builds the tests with the EVM disabled, but doesn't actually run them.
no-evm-tests: proto
	go test -failfast -v -vet=off $(GOFLAGS_NOEVM) -run nothing $(PKG)/...

test-e2e:
	go test -failfast -timeout $(E2E_TESTS_TIMEOUT) -v -vet=off $(PKG)/e2e

test-e2e-race:
	go test -race -failfast -timeout $(E2E_TESTS_TIMEOUT) -v -vet=off $(PKG)/e2e


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
		contracts/dpos.so.3.0.0 \
		contracts/plasmacash.so.1.0.0 \
		pcoracle
