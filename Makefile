GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: fmt
fmt:
	@gofmt -w $(GOFILES)
	@goimports -w $(GOFILES)

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: test
test:
	@go test ./...

.PHONY: build-linux-amd64
build-linux-amd64:
	@GOOS=linux GOARCH=amd64 go build -o bin/camsnap-linux-amd64 ./cmd/camsnap

.PHONY: build-linux-arm64
build-linux-arm64:
	@GOOS=linux GOARCH=arm64 go build -o bin/camsnap-linux-arm64 ./cmd/camsnap

.PHONY: build-linux-arm
build-linux-arm:
	@GOOS=linux GOARCH=arm GOARM=7 go build -o bin/camsnap-linux-arm ./cmd/camsnap

.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-linux-arm

.PHONY: all
all: fmt lint test build-all
