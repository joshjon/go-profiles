# Go Profiles

A project I created while learning Go ✌️

## Prerequisites
- [Go](https://golang.org/doc/install)
  
        brew install go
  
- [Docker community edition](https://hub.docker.com/search/?type=edition&offering=community)
- [CloudFlare CLIs](https://github.com/cloudflare/cfssl)
        
        go get github.com/cloudflare/cfssl/cmd/cfssl
        go get github.com/cloudflare/cfssl/cmd/cfssljson

- [Protobuf](https://developers.google.com/protocol-buffers/docs/downloads) and [GoGo Protobuf](https://github.com/gogo/protobuf)

        brew install protobuf
        go get github.com/gogo/protobuf/proto...@v1.3.2

## Makefile

### Protobuf

- Compile protobuf
  
      make compile

### Testing
- Generate test certificates
  
      make gen-test-cert

- Run tests

      make certs

### Certificates

- Generate CA cert
  
      make gen-ca-cert

- Generate server cert
  
      make gen-server-cert

- Generate client cert

      make gen-client-cert

### Binary

- Build and run binary

      make build
      make run

### Docker

- Build and run using Docker

      make build-docker
      make run-docker
      make stop-docker

## Consuming

For testing purposes use an RPC client such as BloomRPC. Import `/api/v1/profile.proto` and make requests to 
localhost:8400 using TLS configured with the generated certs from your generated CA and client certs.

We have both a client and server in the same code base so for this example we do not need to re-generate the code from
proto files. However, in real world usage, the proto file must be shared with the client, which will then generate its
code files in the programming language of its choice.