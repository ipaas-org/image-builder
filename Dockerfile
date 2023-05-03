# Step 1: Compile nixpacks
FROM alpine:3.17 as nixpacks
RUN apk --no-cache add ca-certificates curl tar bash
RUN curl -sSL https://nixpacks.com/install.sh | bash

# Setp 2: install docker
FROM docker as docker

# Step 3: Modules caching
FROM golang:1.20.1-alpine3.17 as modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

# Step 4: Builder
FROM golang:1.20.1-alpine3.17 as builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build  -o /bin/app .

# Step 5: Final
FROM scratch

# use these 3 lines for debugging purposes
# FROM ubuntu 
# RUN apt update
# RUN apt install -y git curl iputils-ping
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
COPY --from=docker /usr/local/libexec/docker/cli-plugins/docker-buildx /usr/local/libexec/docker/cli-plugins/docker-buildx
COPY --from=nixpacks /usr/local/bin/nixpacks /usr/local/bin/nixpacks
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/config /config
COPY --from=builder /bin/app /app
CMD ["/app"]

# Metadata
LABEL org.opencontainers.image.vendor="IPaaS" \
    org.opencontainers.image.source="https://github.com/ipaas-org/image-builder" \
    org.opencontainers.image.title="image-builder" \
    org.opencontainers.image.description="A microservice to handle image building operations" \
    org.opencontainers.image.version="v0.0.1"