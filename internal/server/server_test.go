package server

import (
	"context"
	"github.com/joshjon/go-profiles/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"net"
	"os"
	"testing"

	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/auth"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestServer(t *testing.T) {
	tests := map[string]func(t *testing.T, rootClient api.ProfileServiceClient, nobodyClient api.ProfileServiceClient, config *Config){
		"create/read a profile succeeds":  testCreateReadProfile,
		"consume past log boundary fails": testProfileNotFound,
		"unauthorized fails":              testUnauthorized,
	}

	for scenario, fn := range tests {
		t.Run(scenario, func(t *testing.T) {
			rootClient,
			nobodyClient,
			config,
			teardown := setupTest(t, nil)
			defer teardown()
			fn(t, rootClient, nobodyClient, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (rootClient api.ProfileServiceClient,
	nobodyClient api.ProfileServiceClient, cfg *Config, teardown func()) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Helper used to create new clients (see below)
	newClient := func(crtPath, keyPath string) (
		*grpc.ClientConn,
		api.ProfileServiceClient,
		[]grpc.DialOption,
	) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})
		require.NoError(t, err)
		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := api.NewProfileServiceClient(conn)
		return conn, client, opts
	}

	// Superuser permitted to produce and consume
	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(
		config.RootClientCertFile,
		config.RootClientKeyFile,
	)

	// Client who is not permitted to do anything
	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		config.NobodyClientCertFile,
		config.NobodyClientKeyFile,
	)

	// Parse the server's cert and key, which is used to configure the
	// server's TLS credentials. Those are then passed as a gRPC server
	// option to the NewGRPCServer function so it can create the gRPC
	// server with that option.
	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: config.ServerCertFile,
		KeyFile:  config.ServerKeyFile,
		CAFile:   config.CAFile,
		Server:   true,
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	require.NoError(t, err)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)

	cfg = &Config{
		Authorizer: authorizer,
	}
	if fn != nil {
		fn(cfg)
	}

	server := NewGRPCServer(cfg, grpc.Creds(serverCreds))

	go func() {
		server.Serve(l)
	}()

	return rootClient, nobodyClient, cfg, func() {
		server.Stop()
		rootConn.Close()
		nobodyConn.Close()
		l.Close()
	}
}

func testCreateReadProfile(t *testing.T, client, _ api.ProfileServiceClient, config *Config) {
	ctx := context.Background()
	payload := &api.CreateProfileReq{FirstName: "Foo", LastName: "Bar"}

	createResponse, err := client.CreateProfile(ctx, payload)
	require.NoError(t, err)
	require.NotEmpty(t, createResponse.Id)

	id := createResponse.Id

	readResponse, err := client.ReadProfile(ctx, &api.ReadProfileReq{Id: id})
	require.NoError(t, err)
	require.Equal(t, id, readResponse.Id)
}

func testProfileNotFound(t *testing.T, client, _ api.ProfileServiceClient, config *Config) {
	ctx := context.Background()
	response, err := client.ReadProfile(ctx, &api.ReadProfileReq{Id: "Foo"})
	require.Nil(t, response)
	code, expected := status.Code(err), status.Code(api.ErrProfileNotFound{}.GRPCStatus().Err())
	require.Equal(t, code, expected)
}

func testUnauthorized(t *testing.T, _, nobodyClient api.ProfileServiceClient, config *Config) {
	ctx := context.Background()
	payload := &api.CreateProfileReq{FirstName: "Foo", LastName: "Bar"}

	createResponse, err := nobodyClient.CreateProfile(ctx, payload)
	require.Nil(t, createResponse)
	code, expectedCode := status.Code(err), codes.PermissionDenied
	require.Equal(t, code, expectedCode)

	readResponse, err := nobodyClient.ReadProfile(ctx, &api.ReadProfileReq{Id: "foo"})
	require.Nil(t, readResponse)
	code, expectedCode = status.Code(err), codes.PermissionDenied
	require.Equal(t, code, expectedCode)
}
