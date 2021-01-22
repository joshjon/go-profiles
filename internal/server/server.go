package server

import (
	"context"
	"strings"
	"time"

	api "github.com/joshjon/go-profiles/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Guarantees *grpcServer satisfies api.LogServer interface.
// This is a trick to ensure all methods are implemented for
// the interface or else the compiler will complain.
// It acts like type-checked code documentation.
var _ api.ProfileServiceServer = (*grpcServer)(nil)

type Config struct {
	CommitLog  CommitProfile
	Authorizer Authorizer
}

const (
	objectWildcard = "*"
	produceAction  = "produce"
	consumeAction  = "consume"
)

type grpcServer struct {
	*Config
	Authorizer Authorizer
}

// Provides users a way to instantiate the service, create a
// gRPC server, and register the service to that server.
// This will give the user a server that just needs a listener
// for it to accept incoming connections.
func newgrpcServer(config *Config) (*grpcServer) {
	srv := &grpcServer{
		Config: config,
	}
	return srv
}

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) *grpc.Server {
	gsrv := grpc.NewServer(grpcOpts...)
	srv := newgrpcServer(config)
	api.RegisterProfileServiceServer(gsrv, srv)
	return gsrv
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		produceAction,
	); err != nil {
		return nil, err
	}

	offset, err := s.CommitLog.Create(req.Record)

	if err != nil {
		return nil, err
	}

	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (
	*api.ConsumeResponse, error) {

	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		consumeAction,
	); err != nil {
		return nil, err
	}

	record, err := s.CommitLog.Read(req.Offset)

	if err != nil {
		return nil, err
	}

	return &api.ConsumeResponse{Record: record}, nil
}

// Bidirectional streaming RPC so the client can stream data into
// the server’s log and the server can tell the client whether each
// request succeeded.
func (s *grpcServer) ProduceStream(stream api.Log_ProduceStreamServer) error {
	for {
		req, err := stream.Recv()

		if err != nil {
			return err
		}

		res, err := s.Produce(stream.Context(), req)

		if err != nil {
			return err
		}

		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

// Implements a server-side streaming RPC so the client can tell the
// server where in the log to read records, and then the server will
// stream every record that follows (even records that aren’t in the
// log yet).
// When the server reaches the end of the log, the server will wait
// until someone appends a record to the log and then continue streaming
// records to the client.
func (s *grpcServer) ConsumeStream(
	req *api.ConsumeRequest,
	stream api.Log_ConsumeStreamServer,
) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Consume(stream.Context(), req)
			switch err.(type) {
			case nil:
			default:
				return err
			}
			if err = stream.Send(res); err != nil {
				return err
			}
			req.Offset++
		}
	}
}

// Server interfaces

type CommitProfile interface {
	Create(*api.Profile) (*api.Profile, error)
	Read(uint64) (*api.Profile, error)
	Update(*api.Profile) (*api.Profile, error)
	Delete(uint64) (bool, error)
	List() (*api.Profile, error)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

// Authentication

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)

	if !ok {
		return ctx, status.New(codes.Unknown, "couldn't find peer info").Err()
	}

	if peer.AuthInfo == nil {
		return ctx, status.New(codes.Unauthenticated, "no transport security being used").Err()
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)
	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
