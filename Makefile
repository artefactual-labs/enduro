SHELL=/bin/bash
BUILD_TIME=$(shell date -u +%Y-%m-%dT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LD_FLAGS= '-X "main.buildTime=$(BUILD_TIME)" -X main.gitCommit=$(GIT_COMMIT)'
GO_FLAGS= -ldflags=$(LD_FLAGS)
GOPATH=$(shell go env GOPATH)
GOBIN=$(GOPATH)/bin
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOTEST=$(GOCMD) test
GOGEN=$(GOCMD) generate

export PATH:=$(GOBIN):$(PATH)

run: enduro-dev
	./build/enduro

enduro-dev: generate
	mkdir -p ./build
	$(GOBUILD) -trimpath -o build/enduro $(GO_FLAGS) -v

test: generate
	$(GOTEST) -race -v ./...

lint:
	golangci-lint run

generate:
	cd / && env GO111MODULE=off go get -u github.com/myitcv/gobin
	gobin github.com/GeertJohan/go.rice/rice
	find . -name fake -type d | xargs rm -rf
	$(GOGEN) ./internal/...

goagen:
	goa gen github.com/artefactual-labs/enduro/internal/api/design -o internal/api

tools:
	cd / && env GO111MODULE=off go get -u github.com/myitcv/gobin
	gobin \
		github.com/minio/mc \
		github.com/golangci/golangci-lint/cmd/golangci-lint \
		github.com/GeertJohan/go.rice/rice \
		github.com/golang/mock/mockgen

clean:
	rm -rf ./build ./dist
	find . -name fake -type d | xargs rm -rf
	find . -name rice-box.go -delete

release-test-config:
	goreleaser --snapshot --skip-publish --rm-dist

release-test:
	goreleaser --skip-publish

website:
	hugo serve --source=website/

ui:
	yarn --cwd ui install
	yarn --cwd ui build
	make ui-gen

ui-dev:
	yarn --cwd ui serve

ui-gen:
	cd / && env GO111MODULE=off go get -u github.com/myitcv/gobin
	gobin github.com/GeertJohan/go.rice/rice
	$(GOGEN) -v ./ui

ui-client:
	rm -rf $(CURDIR)/ui/src/client
	docker container run --rm --user $(shell id -u):$(shell id -g) --volume $(CURDIR):/local openapitools/openapi-generator-cli:v4.2.1 \
		generate \
			--input-spec /local/internal/api/gen/http/openapi.json \
			--generator-name typescript-fetch \
			--output /local/ui/src/openapi-generator/ \
			-p "generateAliasAsModel=true" \
			-p "typescriptThreePlus=true" \
			-p "withInterfaces=true"

cadence-flush:
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "DROP DATABASE IF EXISTS cadence;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "DROP DATABASE IF EXISTS cadence_visibility;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "CREATE DATABASE IF NOT EXISTS cadence;"
	docker-compose exec mysql mysql -hlocalhost -uroot -proot123 -e "CREATE DATABASE IF NOT EXISTS cadence_visibility;"
	docker-compose run --rm cadence /seed.sh
	docker-compose restart cadence
	docker run -it --network=host --rm ubercadence/cli:master --address=127.0.0.1:7400 --domain=enduro domain register

cadence-seed:
	docker-compose run --rm cadence /seed.sh

cadence-domain:
	docker run -it --network=host --rm ubercadence/cli:master --address=127.0.0.1:7400 --domain=enduro domain register

db:
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123

flush:
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "drop database enduro"
	docker-compose exec --user=root mysql mysql -hlocalhost -uroot -proot123 -e "create database enduro"

.PHONY: *
