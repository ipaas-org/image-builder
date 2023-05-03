# Image Builder

## Description

Image builder is a service to build docker images from github repositories.
it listens to a build queue and builds images when a new build request is received.
It also listens to a fail queue to retry failed builds.
If a build is successful, it pushes the image to a registry and notifies a container manager to update the image.
if a build fails, it sends the build request to the fail queue and retries later, if it fails too many times, it sends a notification to lesser lord kusanali (connector manager) to notify the user and discard the build request.

## Requirements

install devcontainer cli if you are not gonna use the vscode terminal

```npm
npm install -g @devcontainers/cli
```

after the installation run `make devc` and it will start the container and attach the tty to it.

if you are gonna use vscode make sure to have the dev containers extension installed and reload the window, it should prompt you to open the folder in the container.

## How to start the service

before starting the development make sure you have a valid .env file
in the config/ folder, you can check how to stucture one in the .env.example file.
(the default usename and password for rabbitmq are guest:guest)

in the development environment, run `make services` and wait for rabbitmq and the registry to start.
then run `make run` to start the service.
if you need to check if the service can handle multiple replicas then run `make up` and it will build an image
and run 3 replicas of itself listening to the same queue.
you can check the logs by running `make logs` and stop everything by running `make down`.

you can also check the commands with `make help`.
remember to use `make prep` before committing to format the code and run the tests.

## How to use the service

check the `BuildRequest` struct in `model/buildInfo.go` to see what the service accept as a request.

here is an example of a request

```json
{
  //github token to authenticate request, you can generate one [here](https://github.com/settings/tokens) if you need to
  "token": "gho_xxx",
  "userID": "18008",
  "type": "repo",
  "connector": "github",
  "repo": "vano2903/testing"
}
```

you can open the rabbitmq management console [localhost:15672](http://localhost:15672/) and check the queues and messages.
also `make services` will start a docker registry with a frontend on [localhost:8081](http://localhost:8081/).

d2:

```d2
direction: right

build queue
fail queue
github
container manager
registry
image builder: {
  listener
  builder
  pusher
  connector
  container service
}

build queue -> image builder.listener

image builder.listener -> image builder.connector

image builder.connector -> github
image builder.connector -> image builder.builder

image builder.builder -> nixpacks: builds image
image builder.builder -> fail queue: build fails
image builder.builder -> image builder.pusher: successful build

image builder.pusher -> registry
image builder.pusher -> image builder.container service

image builder.container service -> container manager

```
