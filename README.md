# Go Profiles

## Prerequisites

- [Go](https://golang.org/doc/install)

        brew install go

- [Docker community edition](https://hub.docker.com/search/?type=edition&offering=community)
- [CloudFlare CLIs](https://github.com/cloudflare/cfssl)

        go get github.com/cloudflare/cfssl/cmd/cfssl
        go get github.com/cloudflare/cfssl/cmd/cfssljson

- [Protobuf](https://developers.google.com/protocol-buffers/docs/downloads)
  and [GoGo Protobuf](https://github.com/gogo/protobuf)

        brew install protobuf
        go get github.com/gogo/protobuf/proto...@v1.3.2

## Protobuf

- Compile protobuf

      make compile

## Testing

- Generate test certificates

      make gen-test-cert

- Run tests

      make test

## Build & Run

- Generate CA, server, and client certs

      make gen-ca-cert
      make gen-server-cert
      make gen-client-cert

- Build and run a binary

      make build
      make run

- Build and run using Docker

      make build-docker
      make run-docker
      make stop-docker

## Client

For local testing, start the server using the binary or docker, and select one of the client methods below:

- `make run-client` to use the client example at `./cmd/client` which executes hard coded requests to the server.
- Use an RPC client such as BloomRPC, import `/api/v1/profile.proto`, and make custom requests to localhost:8400 via TLS
  configured with the generated CA and client certs in `./certs`.
