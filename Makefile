GOFILES := $(shell find . -name "*.go" -not -path "./vendor/*")
BIN_DIR := bin

.PHONY: all
all: fmt lint test build-all

.PHONY: fmt
fmt:
	@echo "Formatting..."
	@gofmt -s -w .
	@if command -v goimports >/dev/null; then goimports -w $(GOFILES); else echo "Skipping goimports (not installed)"; fi

.PHONY: lint
lint:
	@echo "Linting..."
	@if command -v golangci-lint >/dev/null; then golangci-lint run ./...; else echo "Skipping lint (golangci-lint not installed)"; fi

.PHONY: test
test:
	@echo "Testing..."
	@go test ./...

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/camsnap ./cmd/camsnap

.PHONY: build-linux-amd64
build-linux-amd64:
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/camsnap-linux-amd64 ./cmd/camsnap

.PHONY: build-linux-arm64
build-linux-arm64:
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/camsnap-linux-arm64 ./cmd/camsnap

.PHONY: build-linux-arm
build-linux-arm:
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=arm GOARM=7 go build -o $(BIN_DIR)/camsnap-linux-arm ./cmd/camsnap

.PHONY: build-macos-arm64
build-macos-arm64:
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/camsnap-macos-arm64 ./cmd/camsnap

.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-linux-arm build-macos-arm64

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -f camsnap-bin

.PHONY: install
install:
	@mkdir -p $$HOME/bin
	BINARY=""; \
	BUILD_TARGET=""; \
	UNAME_S=$$(uname -s); \
	UNAME_M=$$(uname -m); \
	if [ "$$UNAME_S" = "Darwin" ]; then \
		BINARY="camsnap-macos-arm64"; \
		BUILD_TARGET="build-macos-arm64"; \
	elif [ "$$UNAME_M" = "aarch64" ]; then \
		BINARY="camsnap-linux-arm64"; \
		BUILD_TARGET="build-linux-arm64"; \
	elif [ "$$UNAME_M" = "armv7l" ]; then \
		BINARY="camsnap-linux-arm"; \
		BUILD_TARGET="build-linux-arm"; \
	else \
		BINARY="camsnap-linux-amd64"; \
		BUILD_TARGET="build-linux-amd64"; \
	fi; \
	if [ ! -f $(BIN_DIR)/$$BINARY ]; then \
		echo "Building $$BINARY (target: $$BUILD_TARGET)..."; \
		make $$BUILD_TARGET; \
	fi; \
	cp $(BIN_DIR)/$$BINARY $$HOME/bin/camsnap; \
	echo "Installed camsnap ($$BINARY) to $$HOME/bin/"
