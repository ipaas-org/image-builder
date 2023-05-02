# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# compose-up: ### Run docker-compose
# 	docker-compose up --build -d postgres rabbitmq && docker-compose logs -f
# .PHONY: compose-up

# compose-up-integration-test: ### Run docker-compose with integration test
# 	docker-compose up --build --abort-on-container-exit --exit-code-from integration
# .PHONY: compose-up-integration-test

# compose-down: ### Down docker-compose
# 	docker-compose down --remove-orphans
# .PHONY: compose-down

init-dev: ### initialize commitizen (not mandatory)
	commitizen init cz-conventional-changelog -save-dev -save-exact
.PHONY: init-dev

run: fmt lint ### format, lint, check module and run go code
	go run .
.PHONY: run

fmt: ### format go mod and code
	go mod tidy 
	go fmt .
.PHONY: fmt

lint: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

test: ### run test
	go test ./controller/tests
.PHONY: test

testv: ### run verbose test
	go test -v ./controller/tests
.PHONY: test

update: ### update dependencies
	go mod tidy
	go get -u -v
.PHONY: update

build: prep ### build docker image called image-builder
	docker build -t image-builder .
.PHONY: docker

services: ### start services needed
	docker-compose up --build --remove-orphans -d rabbitmq
.PHONY: services

up: build ### start docker image following docker-compose
	docker-compose up --build --remove-orphans -d app
	make logs
.PHONY: up

logs: ### attach app's logs from docker-compose
	docker-compose logs -f app
.PHONY: logs

down: ### stop all container created by docker-compose
	docker-compose down --remove-orphans
.PHONY: down

prep: fmt lint test ### format, lint and test. to use before commit
.PHONY: prep