# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

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

build: fmt lint ### build docker image called image-builder
	docker build -t image-builder .
.PHONY: docker

services: ### start services needed
	docker-compose up --build --remove-orphans -d rabbitmq registry frontend
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

devc: ### automatically start and  connect to dev container (it works even if the container is already running)
	OUTPUT=$$(devcontainer up --workspace-folder . );\
	OUTCOME=$$(echo $$OUTPUT | jq -r .outcome);\
	echo $$OUTPUT;\
	if [ $$OUTCOME = "success" ];then\
	 COMMAND=$$(echo \'cd $$(echo $$OUTPUT | jq -r .remoteWorkspaceFolder));\
	 COMMAND+=";su $$(echo $$OUTPUT | jq -r .remoteUser)'";\
	 COMMAND=$$(echo docker exec -it $$(echo $$OUTPUT | jq -r .containerId) zsh -c $$COMMAND);\
	 echo $$COMMAND;\
	 eval $$COMMAND;\
	else\
	 echo "devcontainer output was not successful";\
	fi
.PHONY: devc