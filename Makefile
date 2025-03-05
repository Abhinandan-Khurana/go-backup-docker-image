# Build information
VERSION ?= 1.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
MODULE := $(shell go list -m)

# Go build flags
LDFLAGS := -ldflags "-X '$(MODULE)/version.Version=$(VERSION)' -X '$(MODULE)/version.Commit=$(COMMIT)' -X '$(MODULE)/version.BuildTime=$(BUILD_TIME)'"
GOBUILD := GO111MODULE=on CGO_ENABLED=0 go build $(LDFLAGS)

# Binary name
BINARY_NAME := go-backup-docker-image

# Output directory
OUTPUT_DIR := ./dist

# Supported platforms: GOOS/GOARCH pairs
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# Default target
.PHONY: all
all: clean test build-all

# Clean the build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@go clean

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./... -v

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go get -v ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed"; \
		go vet ./...; \
	fi

# Build for the current platform
.PHONY: build
build:
	@echo "Building for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME) .

# Build for all platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	$(foreach platform,$(PLATFORMS),$(call build-platform,$(platform)))

# Build for a specific platform (internal function)
define build-platform
	$(eval GOOS := $(word 1,$(subst /, ,$(1))))
	$(eval GOARCH := $(word 2,$(subst /, ,$(1))))
	$(eval SUFFIX := $(if $(filter windows,$(GOOS)),.exe,))
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_$(GOOS)_$(GOARCH)$(SUFFIX) .
endef

# Create compressed archives for each build
.PHONY: package
package: build-all
	@echo "Creating packages..."
	$(foreach platform,$(PLATFORMS),$(call package-platform,$(platform)))

# Package for a specific platform (internal function)
define package-platform
	$(eval GOOS := $(word 1,$(subst /, ,$(1))))
	$(eval GOARCH := $(word 2,$(subst /, ,$(1))))
	$(eval SUFFIX := $(if $(filter windows,$(GOOS)),.exe,))
	$(eval ARCHIVE_EXT := $(if $(filter windows,$(GOOS)),.zip,.tar.gz))
	@echo "Packaging for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)/tmp/$(BINARY_NAME)
	@cp $(OUTPUT_DIR)/$(BINARY_NAME)_$(GOOS)_$(GOARCH)$(SUFFIX) $(OUTPUT_DIR)/tmp/$(BINARY_NAME)/$(BINARY_NAME)$(SUFFIX)
	@cp README.md LICENSE $(OUTPUT_DIR)/tmp/$(BINARY_NAME)/ 2>/dev/null || true
	@if [ "$(ARCHIVE_EXT)" = ".zip" ]; then \
		(cd $(OUTPUT_DIR)/tmp && zip -r ../$(BINARY_NAME)_$(VERSION)_$(GOOS)_$(GOARCH).zip $(BINARY_NAME)); \
	else \
		tar -czf $(OUTPUT_DIR)/$(BINARY_NAME)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz -C $(OUTPUT_DIR)/tmp $(BINARY_NAME); \
	fi
	@rm -rf $(OUTPUT_DIR)/tmp
endef

# Install the binary locally
.PHONY: install
install: build
	@echo "Installing..."
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Build for specific platforms
.PHONY: linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64

linux-amd64:
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_linux_amd64 .

linux-arm64:
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_linux_arm64 .

darwin-amd64:
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_darwin_amd64 .

darwin-arm64:
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_darwin_arm64 .

windows-amd64:
	@GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_windows_amd64.exe .

windows-arm64:
	@GOOS=windows GOARCH=arm64 $(GOBUILD) -o $(OUTPUT_DIR)/$(BINARY_NAME)_windows_arm64.exe .

# Help target
.PHONY: help
help:
	@echo "Docker Image Backup Tool (go-backup-docker-image) Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all           Clean, test, and build for all platforms"
	@echo "  clean         Remove build artifacts"
	@echo "  test          Run tests"
	@echo "  deps          Install dependencies"
