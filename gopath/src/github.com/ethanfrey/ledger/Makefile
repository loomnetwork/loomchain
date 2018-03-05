.PHONY: install test vendor

install:
	go install ./cmd/ledger

test:
	@go test .

vendor:
	@go get github.com/Masterminds/glide
	@glide install
