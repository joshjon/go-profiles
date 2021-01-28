# Go Profiles

A project I created while learning Go ✌️.

### Makefile

Compile protobuf for Go
```shell
make compile
```

Create directory for test certificates
```shell
make init
```
 
Generate test certificates
```shell
make gencert
```

Run tests
```shell
make test
```


### Consuming the REST endpoints
We have both a client and server in the same code base so for this example we do not need to re-generate the code from 
proto files. However, in real world usage, the proto file must be shared with the client, which will then generate its 
code files in the programming language of its choice.