package server

import (
	"context"
	"github.com/joshjon/go-profiles/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"math/rand"
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
	for scenario, fn := range map[string]func(
		t *testing.T,
		rootClient api.ProfileServiceClient,
		nobodyClient api.ProfileServiceClient,
		config *Config,
	){
		// ...
		//"create/read a profile succeeds": testCreateReadProfile,
		//"produce/consume stream succeeds":                     testProduceConsumeStream,
		//"consume past log boundary fails":                     testConsumePastBoundary,
		"unauthorized fails": testUnauthorized,
	} {
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

func setupTest(t *testing.T, fn func(*Config)) (
	rootClient api.ProfileServiceClient,
	nobodyClient api.ProfileServiceClient,
	cfg *Config,
	teardown func(),
) {
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

// Tests that producing and consuming works by using our client and
// server to produce a record to the log, consume it back, and then
// check that the record we sent is the same one we got back.
func testCreateReadProfile(t *testing.T, client, _ api.ProfileServiceClient, config *Config) {
	ctx := context.Background()

	want := &api.Profile{
		Id:        rand.Uint64(),
		FirstName: "Foo",
		LastName:  "Bar",
	}

	createProfile, err := client.CreateProfile(
		ctx,
		&api.CreateProfileReq{
			Profile: want,
		},
	)
	require.NoError(t, err)

	readProfile, err := client.ReadProfile(ctx, &api.ReadProfileReq{
		Id: createProfile.Profile.Id,
	})
	require.NoError(t, err)
	require.Equal(t, want, readProfile.Profile)
}

// TODO: use this to test profile id not found

//// Tests that our server responds with an api.ErrOffsetOutOfRange
//// error when a client tries to consume beyond the log’s boundaries.
//func testConsumePastBoundary(
//	t *testing.T,
//	client, _ api.LogClient,
//	config *Config,
//) {
//	ctx := context.Background()
//
//	produce, err := client.Produce(ctx, &api.ProduceRequest{
//		Record: &api.Record{
//			Value: []byte("hello world"),
//		},
//	})
//	require.NoError(t, err)
//
//	consume, err := client.Consume(ctx, &api.ConsumeRequest{
//		Offset: produce.Offset + 1,
//	})
//	if consume != nil {
//		t.Fatal("consume not nil")
//	}
//	got := grpc.Code(err)
//	want := grpc.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
//	if got != want {
//		t.Fatalf("got err: %v, want: %v", got, want)
//	}
//}

func testUnauthorized(
	t *testing.T,
	_,
	client api.ProfileServiceClient,
	config *Config,
) {
	profile := &api.Profile{
		Id:        rand.Uint64(),
		FirstName: "Foo",
		LastName:  "Bar",
	}

	ctx := context.Background()
	createProfile, err := client.CreateProfile(ctx,
		&api.CreateProfileReq{Profile: profile},
	)
	if createProfile != nil {
		t.Fatalf("createProfile response should be nil")
	}
	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
	readProfile, err := client.ReadProfile(ctx, &api.ReadProfileReq{
		Id: profile.Id,
	})
	if readProfile != nil {
		t.Fatalf("readProfile response should be nil")
	}
	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
}

func setupTest1(t *testing.T, fn func(*Config)) (
	client api.ProfileServiceClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: config.CAFile,
	})
	require.NoError(t, err)

	clientCreds := credentials.NewTLS(clientTLSConfig)
	cc, err := grpc.Dial(
		l.Addr().String(),
		grpc.WithTransportCredentials(clientCreds),
	)
	require.NoError(t, err)

	client = api.NewProfileServiceClient(cc)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	require.NoError(t, err)

	require.NoError(t, err)

	cfg = &Config{}
	if fn != nil {
		fn(cfg)
	}
	server := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
	}
}

func setupTest2(t *testing.T, fn func(*Config)) (
	client api.ProfileServiceClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: config.CAFile,
	})
	require.NoError(t, err)

	clientCreds := credentials.NewTLS(clientTLSConfig)
	cc, err := grpc.Dial(
		l.Addr().String(),
		grpc.WithTransportCredentials(clientCreds),
	)
	require.NoError(t, err)

	client = api.NewProfileServiceClient(cc)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	require.NoError(t, err)

	require.NoError(t, err)

	cfg = &Config{}
	if fn != nil {
		fn(cfg)
	}
	server := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
	}
}
