# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFUMPT := gofumpt
GOLINT := golangci-lint

# Project parameters
BIN_DIR := bin
CMD_DIR := cmd
TARGETS := $(notdir $(wildcard $(CMD_DIR)/*))
PKG_LIST := $(shell $(GOCMD) list ./... | grep -v /vendor/)

# Build flags
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S')
LDFLAGS := -w -s -X 'master.Version=$(VERSION)' -X 'master.BuildTime=$(BUILD_TIME)'
CGO_FLAGS := CGO_ENABLED=1 # CGO_CXXFLAGS='-D_GLIBCXX_USE_CXX11_ABI=0'

# Colors for pretty printing
BLUE := \033[0;34m
NC := \033[0m # No Color

# Targets
.PHONY: all clean test lint tidy help $(TARGETS)

# Default target
all: build
build: clean tidy fumpt lint $(TARGETS)

# For GDP without golangci-lint
compile: tidy $(TARGETS)

# Build each target
define build_target
$(BIN_DIR)/$(1): $$(shell find $(CMD_DIR)/$(1) -name '*.go')
	@printf "$(BLUE)Building $$@...$(NC)\n"
	@mkdir -p $(BIN_DIR)
	@$(CGO_FLAGS) $(GOBUILD) -ldflags "$(LDFLAGS)" -o $$@ ./$(CMD_DIR)/$(1)
endef

# Generate build rules for each target
$(foreach target,$(TARGETS),$(eval $(call build_target,$(target))))

# Shortcut targets
$(TARGETS):
	@echo "Building with CGO_FLAGS=$(CGO_FLAGS)"
	@$(MAKE) $(BIN_DIR)/$@

test:
	@printf "$(BLUE)Running tests ...$(NC)\n"
	@$(GOTEST) -v $(PKG_LIST)

coverage:
	@printf "$(BLUE)Running tests with coverage ...$(NC)\n"
	@$(GOTEST) -v -coverprofile=coverage.out $(PKG_LIST)
	@$(GOCOVER) -html=coverage.out -o coverage.html

fumpt:
	@printf "$(BLUE)Running fumpt ...$(NC)\n"
	@$(GOFUMPT) -w -l $(shell find . -name '*.go')

lint:
	@printf "$(BLUE)Running linter ...$(NC)\n"
	@$(GOLINT) run ./...

tidy:
	@printf "$(BLUE)Tidying and verifying module dependencies ...$(NC)\n"
	@$(GOMOD) tidy
	@$(GOMOD) verify

clean:
	@printf "$(BLUE)Cleaning up ...$(NC)\n"
	@$(GOCLEAN)
	@rm -rf $(BIN_DIR) *.pid *.perf coverage.html coverage.out

help:
	@echo "Available targets:"
	@echo "  all (build) : Build the program (default)"
	@echo "  test        : Run tests"
	@echo "  fumpt       : Run gofumpt"
	@echo "  lint        : Run golangci-lint"
	@echo "  tidy        : Tidy and verify go modules"
	@echo "  clean       : Remove object files and binaries"
	@echo "  help        : Display this help message"
	@echo "  <target>    : Build specific target ($(TARGETS))"

# Debugging
print-%:
	@echo '$*=$($*)'
