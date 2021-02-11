FROM golang:1.15-alpine AS build

WORKDIR /go/src/go-profiles
COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/go-profiles ./cmd/go-profiles

RUN GRPC_HEALTH_PROBE_VERSION=v0.3.1 && \
    wget -qO/go/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /go/bin/grpc_health_probe

FROM scratch
COPY --from=build /go/bin/go-profiles /bin/go-profiles
COPY --from=build /go/src/go-profiles/args.yaml /bin/
COPY --from=build /go/src/go-profiles/certs /bin/certs
COPY --from=build /go/src/go-profiles/config /bin/config
COPY --from=build /go/bin/grpc_health_probe /bin/grpc_health_probe

ENTRYPOINT ["/bin/go-profiles", "--config-file", "/bin/args.yaml"]
