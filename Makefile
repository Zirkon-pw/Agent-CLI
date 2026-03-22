APP_NAME    := agentctl
MODULE      := github.com/docup/agentctl
BUILD_DIR   := build
CMD_PATH    := ./cmd/agentctl
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

GOFLAGS     ?=
GOTEST      := go test
GOLINT      := golangci-lint
TEST_PKGS   := ./tests/...
COVER_PKGS  := ./internal/...
COVER_TEST_PKGS := ./tests/unit/... ./tests/integration/... ./tests/runtime/...
GO_TEST_ENV := GOCACHE=$(CURDIR)/.gocache

.PHONY: all build install clean test test-verbose lint fmt vet tidy run help

## —— Build ——————————————————————————————————————————

all: tidy fmt vet build ## Run tidy, fmt, vet, then build

build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(APP_NAME)"

install: ## Install to $GOPATH/bin
	go install $(GOFLAGS) -ldflags "$(LDFLAGS)" $(CMD_PATH)
	@echo "Installed $(APP_NAME)"

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	go clean -cache -testcache
	@echo "Cleaned"

## —— Quality ————————————————————————————————————————

test: ## Run tests
	$(GO_TEST_ENV) $(GOTEST) $(TEST_PKGS) -count=1

test-verbose: ## Run tests with verbose output
	$(GO_TEST_ENV) $(GOTEST) -v $(TEST_PKGS) -count=1

test-cover: ## Run tests with coverage
	@mkdir -p $(BUILD_DIR)
	$(GO_TEST_ENV) $(GOTEST) -coverpkg=$(COVER_PKGS) -coverprofile=$(BUILD_DIR)/coverage.out $(COVER_TEST_PKGS) -count=1
	go tool cover -func=$(BUILD_DIR)/coverage.out
	@echo "HTML report: go tool cover -html=$(BUILD_DIR)/coverage.out"

lint: ## Run linter (requires golangci-lint)
	$(GOLINT) run ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy and verify modules
	go mod tidy
	go mod verify

## —— Run ———————————————————————————————————————————

run: build ## Build and run
	$(BUILD_DIR)/$(APP_NAME) $(ARGS)

init: build ## Build and init workspace in current dir
	$(BUILD_DIR)/$(APP_NAME) init

## —— Cross-compile ——————————————————————————————————

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

release: ## Cross-compile for all platforms
	@mkdir -p $(BUILD_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval OS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT := $(if $(filter windows,$(OS)),.exe,)) \
		echo "Building $(OS)/$(ARCH)..." && \
		GOOS=$(OS) GOARCH=$(ARCH) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o $(BUILD_DIR)/$(APP_NAME)-$(OS)-$(ARCH)$(EXT) $(CMD_PATH) && \
	) true
	@echo "Release binaries in $(BUILD_DIR)/"

## —— Help ——————————————————————————————————————————

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
