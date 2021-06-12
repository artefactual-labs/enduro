include .bingo/Variables.mk

SHELL=/bin/bash
BUILD_TIME=$(shell date -u +%Y-%m-%dT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LD_FLAGS= '-X "main.buildTime=$(BUILD_TIME)" -X main.gitCommit=$(GIT_COMMIT)'
GO_FLAGS= -ldflags=$(LD_FLAGS)

export PATH:=$(GOBIN):$(PATH)

tools:
	$(GO) get github.com/bwplotka/bingo

run: enduro-dev
	./build/enduro

enduro-dev:
	mkdir -p ./build
	$(GO) build -trimpath -o build/enduro $(GO_FLAGS) -v

test:
	$(GO) test -race -v ./...

lint:
	$(GOLANGCI_LINT) run

goagen:
	$(GOA) gen github.com/artefactual-labs/enduro/internal/api/design -o internal/api

clean:
	rm -rf ./build ./dist
	find . -name fake -type d | xargs rm -rf
	find . -name rice-box.go -delete

release-test-config:
	goreleaser --snapshot --skip-publish --rm-dist

release-test:
	goreleaser --skip-publish

website:
	$(HUGO) serve --source=website/

ui:
	yarn --cwd ui install
	yarn --cwd ui build
	make ui-gen

ui-dev:
	yarn --cwd ui serve

ui-gen:
	$(GO) generate -v ./ui

ui-client:
	@rm -rf $(CURDIR)/ui/src/client
	@docker container run --rm --user $(shell id -u):$(shell id -g) --volume $(CURDIR):/local openapitools/openapi-generator-cli:v4.2.3 \
		generate \
			--input-spec /local/internal/api/gen/http/openapi.json \
			--generator-name typescript-fetch \
			--output /local/ui/src/openapi-generator/ \
			--skip-validate-spec \
			-p "generateAliasAsModel=true" \
			-p "typescriptThreePlus=true" \
			-p "withInterfaces=true"
	@echo "@@@@ Please, review all warnings generated by openapi-generator-cli above!"
	@echo "@@@@ We're using \`--skip-validate-spec\` to deal with Goa spec generation issues."

cadence-flush:
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "DROP DATABASE IF EXISTS cadence;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "DROP DATABASE IF EXISTS cadence_visibility;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "CREATE DATABASE IF NOT EXISTS cadence;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "CREATE DATABASE IF NOT EXISTS cadence_visibility;"
	docker-compose run --rm cadence /seed.sh
	docker-compose restart cadence
	docker run -it --network=host --rm ubercadence/cli:master --address=127.0.0.1:7400 --domain=enduro domain register --active_cluster=active

cadence-seed:
	docker-compose run --rm cadence /seed.sh

cadence-domain:
	docker run -it --network=host --rm ubercadence/cli:master --address=127.0.0.1:7400 --domain=enduro domain register --active_cluster=active

db:
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123

flush:
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "drop database enduro"
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "create database enduro"

bingen: gen-ui gen-migrations

gen-mock:
	$(MOCKGEN) -destination=./internal/batch/fake/mock_batch.go -package=fake github.com/artefactual-labs/enduro/internal/batch Service
	$(MOCKGEN) -destination=./internal/collection/fake/mock_collection.go -package=fake github.com/artefactual-labs/enduro/internal/collection Service
	$(MOCKGEN) -destination=./internal/pipeline/fake/mock_pipeline.go -package=fake github.com/artefactual-labs/enduro/internal/pipeline Service
	$(MOCKGEN) -destination=./internal/watcher/fake/mock_watcher.go -package=fake github.com/artefactual-labs/enduro/internal/watcher Service
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_ingest.go -package=fake github.com/artefactual-labs/enduro/internal/amclient IngestService
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_processing_config.go -package=fake github.com/artefactual-labs/enduro/internal/amclient ProcessingConfigService
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_transfer.go -package=fake github.com/artefactual-labs/enduro/internal/amclient TransferService
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_v2_jobs.go -package=fake github.com/artefactual-labs/enduro/internal/amclient JobsService
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_v2_package.go -package=fake github.com/artefactual-labs/enduro/internal/amclient PackageService
	$(MOCKGEN) -destination=./internal/amclient/fake/mock_v2_task.go -package=fake github.com/artefactual-labs/enduro/internal/amclient TaskService

gen-ui:
	cd ui/ && $(RICE) embed-go

gen-migrations:
	cd internal/db && $(RICE) embed-go

.PHONY: *
