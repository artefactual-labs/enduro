# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.4.3. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for goa variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(GOA)
#	@echo "Running goa"
#	@$(GOA) <flags/args..>
#
GOA := $(GOBIN)/goa-v3.2.4
$(GOA): $(BINGO_DIR)/goa.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/goa-v3.2.4"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=goa.mod -o=$(GOBIN)/goa-v3.2.4 "goa.design/goa/v3/cmd/goa"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.40.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.40.1"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.40.1 "github.com/golangci/golangci-lint/cmd/golangci-lint"

MOCKGEN := $(GOBIN)/mockgen-v1.5.0
$(MOCKGEN): $(BINGO_DIR)/mockgen.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/mockgen-v1.5.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=mockgen.mod -o=$(GOBIN)/mockgen-v1.5.0 "github.com/golang/mock/mockgen"

RICE := $(GOBIN)/rice-v1.0.2
$(RICE): $(BINGO_DIR)/rice.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/rice-v1.0.2"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=rice.mod -o=$(GOBIN)/rice-v1.0.2 "github.com/GeertJohan/go.rice/rice"

