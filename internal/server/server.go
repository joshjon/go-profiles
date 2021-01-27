package server

import (
	"context"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcAuth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	api "github.com/joshjon/go-profiles/api/v1"
	"sync"

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
	//CommitProfile CommitProfile
	Authorizer Authorizer
}

const (
	objectWildcard = "*"
	createAction   = "create"
	readAction     = "read"
	updateAction   = "update"
	deleteAction   = "delete"
)

type grpcServer struct {
	*Config
	mu       *sync.RWMutex
	profiles []*api.Profile
}

// Provides users a way to instantiate the service, create a
// gRPC server, and register the service to that server.
// This will give the user a server that just needs a listener
// for it to accept incoming connections.
func newgrpcServer(config *Config) *grpcServer {
	return &grpcServer{
		Config: config,
		mu:     &sync.RWMutex{},
	}
}

func NewGRPCServer(config *Config, grpcOpts ...grpc.ServerOption) *grpc.Server {
	grpcOpts = append(grpcOpts, grpc.StreamInterceptor(
		grpcMiddleware.ChainStreamServer(
			grpcAuth.StreamServerInterceptor(authenticate),
		)), grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(
		grpcAuth.UnaryServerInterceptor(authenticate),
	)))
	gsrv := grpc.NewServer(grpcOpts...)
	srv := newgrpcServer(config)
	api.RegisterProfileServiceServer(gsrv, srv)
	return gsrv
}

func (server *grpcServer) CreateProfile(ctx context.Context, req *api.CreateProfileReq) (*api.CreateProfileRes, error) {
	if err := server.Authorizer.Authorize(subject(ctx), objectWildcard, createAction); err != nil {
		return nil, err
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	for _, profile := range server.profiles {
		if profile.GetId() == req.Profile.GetId() {
			return nil, status.Error(codes.FailedPrecondition, "profile already exists")
		}
	}

	server.profiles = append(server.profiles, req.Profile)
	return &api.CreateProfileRes{Profile: req.Profile}, nil
}

func (server *grpcServer) ReadProfile(ctx context.Context, req *api.ReadProfileReq) (*api.ReadProfileRes, error) {
	if err := server.Authorizer.Authorize(subject(ctx), objectWildcard, readAction); err != nil {
		return nil, err
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	for _, profile := range server.profiles {
		if profile.GetId() == req.GetId() {
			return &api.ReadProfileRes{Profile: profile}, nil
		}
	}

	return nil, api.ErrProfileNotFound{Id: req.GetId()}
}

func (server *grpcServer) UpdateProfile(ctx context.Context, req *api.UpdateProfileReq) (*api.UpdateProfileRes, error) {
	if err := server.Authorizer.Authorize(subject(ctx), objectWildcard, updateAction); err != nil {
		return nil, err
	}
	panic("implement me")
}

func (server *grpcServer) DeleteProfile(ctx context.Context, req *api.DeleteProfileReq) (*api.DeleteProfileRes, error) {
	if err := server.Authorizer.Authorize(subject(ctx), objectWildcard, deleteAction); err != nil {
		return nil, err
	}
	panic("implement me")
}

func (server *grpcServer) ListProfiles(ctx context.Context, req *api.ListProfilesReq) (*api.ListProfilesRes, error) {
	panic("implement me")
	//if err := server.Authorizer.Authorize(subject(ctx), objectWildcard, readAction); err != nil {
	//	return nil, err
	//}
	//
	//server.mu.Lock()
	//defer server.mu.Unlock()
	//
	//if len(server.profiles) == 0 {
	//
	//}
}

// Interfaces

// Only need CommitProfile interface if following same pattern
// as the book e.g. profile passed to server -> log -> segment -> store

//type CommitProfile interface {
//	Create(*api.Profile) (*api.Profile, error)
//	Read(uint64) (*api.Profile, error)
//	Update(*api.Profile) (*api.Profile, error)
//	Delete(uint64) (bool, error)
//	List() (*api.Profile, error)
//}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

// Authentication

// Interceptor that reads the subject out of the client’s cert and
// writes it to the RPC’s context.
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

// Returns the client’s cert’s subject so we can identify a client
// and check their access.
func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}
