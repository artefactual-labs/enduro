PROJECT := enduro
MAKEDIR := hack/make
SHELL   := /bin/bash

.DEFAULT_GOAL := help
.PHONY: *

DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

include hack/make/bootstrap.mk
include hack/make/dep_goa.mk
include hack/make/dep_golangci_lint.mk
include hack/make/dep_gomajor.mk
include hack/make/dep_goreleaser.mk
include hack/make/dep_gotestsum.mk
include hack/make/dep_hugo.mk
include hack/make/dep_jq.mk
include hack/make/dep_mockgen.mk
include hack/make/dep_temporal_cli.mk


define NEWLINE


endef

IGNORED_PACKAGES := \
	github.com/artefactual-labs/enduro/hack/gencols \
	github.com/artefactual-labs/enduro/hack/landfill/gencols \
	github.com/artefactual-labs/enduro/internal/api/design \
	github.com/artefactual-labs/enduro/internal/api/gen/batch \
	github.com/artefactual-labs/enduro/internal/api/gen/collection \
	github.com/artefactual-labs/enduro/internal/api/gen/collection/views \
	github.com/artefactual-labs/enduro/internal/api/gen/http/batch/client \
	github.com/artefactual-labs/enduro/internal/api/gen/http/batch/server \
	github.com/artefactual-labs/enduro/internal/api/gen/http/cli/enduro \
	github.com/artefactual-labs/enduro/internal/api/gen/http/collection/client \
	github.com/artefactual-labs/enduro/internal/api/gen/http/collection/server \
	github.com/artefactual-labs/enduro/internal/api/gen/http/pipeline/client \
	github.com/artefactual-labs/enduro/internal/api/gen/http/pipeline/server \
	github.com/artefactual-labs/enduro/internal/api/gen/http/swagger/client \
	github.com/artefactual-labs/enduro/internal/api/gen/http/swagger/server \
	github.com/artefactual-labs/enduro/internal/api/gen/pipeline \
	github.com/artefactual-labs/enduro/internal/api/gen/pipeline/views \
	github.com/artefactual-labs/enduro/internal/api/gen/swagger \
	github.com/artefactual-labs/enduro/internal/batch/fake \
	github.com/artefactual-labs/enduro/internal/collection/fake \
	github.com/artefactual-labs/enduro/internal/pipeline/fake \
	github.com/artefactual-labs/enduro/internal/watcher/fake
PACKAGES		:= $(shell go list ./...)
TEST_PACKAGES	:= $(filter-out $(IGNORED_PACKAGES),$(PACKAGES))

run: # @HELP Builds and run the enduro binary.
run: build
	./build/enduro

build: # @HELP Builds the enduro binary.
build: GO         ?= $(shell which go)
build: BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%T%z)
build: GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
build: LD_FLAGS   ?= '-X "main.buildTime=$(BUILD_TIME)" -X main.gitCommit=$(GIT_COMMIT)'
build: GO_FLAGS   ?= -ldflags=$(LD_FLAGS)
build:
	mkdir -p ./build
	$(GO) build -trimpath -o build/enduro $(GO_FLAGS) -v

deps: $(GOMAJOR) # @HELP Lists available module dependency updates.
	gomajor list

test: $(GOTESTSUM) # @HELP Tests using gotestsum.
	gotestsum $(TEST_PACKAGES)

test-race: $(GOTESTSUM) # @HELP Tests using gotestsum and the race detector.
	gotestsum $(TEST_PACKAGES) -- -race

ignored: # @HELP Prints ignored packages.
ignored:
	$(foreach PACKAGE,$(IGNORED_PACKAGES),@echo $(PACKAGE)$(NEWLINE))

lint: # @HELP Lints the code using golangci-lint.
lint: $(GOLANGCI_LINT)
	golangci-lint run -v --timeout=5m --fix

gen-goa: # @HELP Generates Goa assets.
gen-goa: $(GOA)
	goa gen github.com/artefactual-labs/enduro/internal/api/design -o internal/api
	@$(MAKE) gen-goa-json-pretty

gen-goa-json-pretty: goa_http_dir = "internal/api/gen/http"
gen-goa-json-pretty: json_files = $(shell find $(goa_http_dir) -type f -name "*.json" | sort -u)
gen-goa-json-pretty: $(JQ)
	@for f in $(json_files); \
		do (cat "$$f" | jq -S '.' >> "$$f".sorted && mv "$$f".sorted "$$f") \
			&& echo "Formatting $$f with jq" || exit 1; \
	done

clean: # @HELP Cleans temporary files.
clean:
	rm -rf ./build ./dist

release-test-config: # @HELP Tests the goreleaser config.
release-test-config: $(GORELEASER)
	goreleaser --snapshot --skip-publish --clean

release-test: # @HELP Tests the release with goreleaser.
release-test: $(GORELEASER)
	goreleaser --skip-publish

website: # @HELP Serves the website for development.
website: $(HUGO)
	hugo serve --source=website/

ui: # @HELP Builds the UI.
ui:
	npm --prefix=ui install
	npm --prefix=ui run build

ui-dev:
ui-dev: # @HELP Serves the UI for development.
	npm --prefix=ui run dev

ui-client: # @HELP Generates the UI client using openapi-generator-cli.
ui-client:
	rm -rf $(CURDIR)/ui/src/client
	docker container run --rm --user $(shell id -u):$(shell id -g) --volume $(CURDIR):/local openapitools/openapi-generator-cli:v6.6.0 \
		generate \
			--input-spec /local/internal/api/gen/http/openapi3.json \
			--generator-name typescript-fetch \
			--output /local/ui/src/openapi-generator/ \
			-p "generateAliasAsModel=false" \
			-p "withInterfaces=true" \
			-p "supportsES6=true"
	echo "@@@@ Please, review all warnings generated by openapi-generator-cli above!"

db: # @HELP Opens the MySQL CLI.
db:
	docker compose exec --user=root mysql mysql -hlocalhost -uroot -proot123

flush: # @HELP Flushes the enduro database.
flush:
	docker compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "drop database enduro"
	docker compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "create database enduro"

gen-mock: # @HELP Generates mocks with mockgen.
gen-mock: $(MOCKGEN)
	mockgen -typed -destination=./internal/batch/fake/mock_batch.go -package=fake github.com/artefactual-labs/enduro/internal/batch Service
	mockgen -typed -destination=./internal/collection/fake/mock_collection.go -package=fake github.com/artefactual-labs/enduro/internal/collection Service
	mockgen -typed -destination=./internal/pipeline/fake/mock_pipeline.go -package=fake github.com/artefactual-labs/enduro/internal/pipeline Service
	mockgen -typed -destination=./internal/watcher/fake/mock_watcher.go -package=fake github.com/artefactual-labs/enduro/internal/watcher Service

temporal: # @HELP Runs a development instance of Temporal.
temporal: PORT := 55555
temporal: LOG_LEVEL := warn
temporal: $(TEMPORAL_CLI)
	temporal server start-dev --namespace=default --port=$(PORT) --headless --log-format=pretty --log-level=$(LOG_LEVEL)

help: # @HELP Prints this message.
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' Makefile             \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-30s %s\n", $$1, $$2 };  \
	    '
