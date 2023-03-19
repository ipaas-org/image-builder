d2:

```
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


TODO: la pull deve solo fare la pull e prendere last commit, nome, e cose senza richieste
