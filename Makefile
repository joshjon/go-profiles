MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
CURRENT_DIR := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
CONFIG_PATH=$(CURRENT_DIR)/config
CERT_PATH=$(CURRENT_DIR)/certs
TEST_CONFIG_PATH=$(CURRENT_DIR)/test
TEST_CERT_PATH=$(TEST_CONFIG_PATH)/certs

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--gogo_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto,plugins=grpc:. \
		--proto_path=${GOPATH}/src \
		--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
		--proto_path=. \
		--govalidators_out=gogoimport=true:.

.PHONY: gen-ca-cert
gen-ca-cert:
	cfssl gencert -initca $(CONFIG_PATH)/ca-csr.json | cfssljson -bare ca
	mkdir -p $(CERT_PATH)
	mv *.pem *.csr $(CERT_PATH)

.PHONY: gen-server-cert
gen-server-cert:
	cfssl gencert \
		-ca=$(CERT_PATH)/ca.pem \
		-ca-key=$(CERT_PATH)/ca-key.pem \
		-config=$(CONFIG_PATH)/ca-config.json \
		-profile=server \
		$(CONFIG_PATH)/server-csr.json | cfssljson -bare server

	mkdir -p $(CERT_PATH)
	mv *.pem *.csr $(CERT_PATH)

.PHONY: gen-client-cert
gen-client-cert:
	cfssl gencert \
		-ca=$(CERT_PATH)/ca.pem \
		-ca-key=$(CERT_PATH)/ca-key.pem \
		-config=$(CONFIG_PATH)/ca-config.json \
		-profile=client \
		-cn="root" \
		$(CONFIG_PATH)/client-csr.json | cfssljson -bare root-client

	mkdir -p $(CERT_PATH)
	mv *.pem *.csr $(CERT_PATH)

.PHONY: gen-test-cert
gen-test-certs:
	cfssl gencert \
		-initca $(TEST_CONFIG_PATH)/test-ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=$(TEST_CONFIG_PATH)/test-ca-config.json \
		-profile=server \
		$(TEST_CONFIG_PATH)/test-server-csr.json | cfssljson -bare server

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=$(TEST_CONFIG_PATH)/test-ca-config.json \
		-profile=client \
		-cn="root" \
		$(TEST_CONFIG_PATH)/test-client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=$(TEST_CONFIG_PATH)/test-ca-config.json \
		-profile=client \
		-cn="nobody" \
		$(TEST_CONFIG_PATH)/test-client-csr.json | cfssljson -bare nobody-client

	mkdir -p $(TEST_CERT_PATH)
	mv *.pem *.csr $(TEST_CERT_PATH)
	cp $(TEST_CONFIG_PATH)/test-model.conf $(TEST_CERT_PATH)/model.conf
	cp $(TEST_CONFIG_PATH)/test-policy.csv $(TEST_CERT_PATH)/policy.csv

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	# CGO_ENABLED=0 in order to statically compile the binary as opposed to dynamically linked.
	CGO_ENABLED=0 go build -o ./build/go-profiles ./cmd/go-profiles

.PHONY: run
run:
	./build/go-profiles --server-tls-ca-file ./certs/ca.pem  \
						--server-tls-cert-file ./certs/server.pem \
						--server-tls-key-file ./certs/server-key.pem \
						--acl-model-file ./config/model.conf \
						--acl-policy-file ./config/policy.csv

.PHONY: run-client
run-client:
	go run cmd/client/main.go

TAG ?= 0.0.1
REPO = github.com/joshjon/go-profiles
CONTAINER = go-profiles

.PHONY: build-docker
build-docker:
	docker build -t $(REPO):$(TAG) .

.PHONY: run-docker
run-docker:
	docker run -p 8400:8400 -d --name $(CONTAINER) $(REPO):$(TAG)
	@echo Serving gRPC on localhost:8400

.PHONY: stop-docker
stop-docker:
	docker stop $(CONTAINER) | xargs docker rm
