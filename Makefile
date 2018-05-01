PKG = github.com/loomnetwork/loomchain
GIT_SHA = `git rev-parse --verify HEAD`
GOFLAGS = -tags "evm" -ldflags "-X $(PKG).Build=$(BUILD_NUMBER) -X $(PKG).GitSHA=$(GIT_SHA)"
PROTOC = protoc --plugin=./protoc-gen-gogo -Ivendor -I$(GOPATH)/src -I/usr/local/include
PLUGIN_DIR = $(GOPATH)/src/github.com/loomnetwork/go-loom

.PHONY: all clean test install deps proto builtin

all: loom builtin

builtin: contracts/coin.so.1.0.0

contracts/coin.so.1.0.0:
	go build -buildmode=plugin -o $@ $(PKG)/builtin/plugins/coin

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
		github.com/ethereum/go-ethereum
	dep ensure -vendor-only

test: proto
	go test -v $(GOFLAGS) $(PKG)/...

clean:
	go clean
	rm -f \
		loom \
		protoc-gen-gogo \
		contracts/coin.so.1.0.0
