# Go Profiles

A project I created while learning Go ✌️

### Makefile

Compile protobuf
```shell
make compile
```
 
Generate test certificates
```shell
make gen-test-cert
```

Run tests
```shell
make certs
```

Generate CA cert
```shell
make gen-ca-cert
```

Generate Server cert
```shell
make gen-server-cert
```

Generate Client cert
```shell
make gen-client-cert
```

### Consuming the REST endpoints
We have both a client and server in the same code base so for this example we do not need to re-generate the code from 
proto files. However, in real world usage, the proto file must be shared with the client, which will then generate its 
code files in the programming language of its choice.

### Running the service
```shell
go run cmd/main.go --config-file args.yaml
```

### Build and run binary
`CGO_ENABLED=0` in order to statically compile the binary as opposed to dynamically linked.
```shell
CGO_ENABLED=0 go build -o ./build/go-profiles ./cmd
./build/go-profiles --config-file args.yaml
```