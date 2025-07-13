# Makefile for flarewrap

GO         ?= go
BINARY     := flarewrap
BIN_DIR    := dist
LDFLAGS    := -ldflags "-s -w"

.PHONY: build run clean install lint

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/flarewrap

run: build
	$(BIN_DIR)/$(BINARY)

clean:
	@rm -rf $(BIN_DIR)

install:
	$(GO) install $(LDFLAGS) ./cmd/flarewrap

lint:
	golangci-lint run
