package server

import (
	"context"
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/auth"
	"github.com/joshjon/go-profiles/internal/config"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
	"os"
	"testing"
)

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type ServerTestSuite struct {
	suite.Suite
	server       *grpc.Server
	rootConn     *grpc.ClientConn
	rootClient   api.ProfileServiceClient
	nobodyConn   *grpc.ClientConn
	nobodyClient api.ProfileServiceClient
	listener     net.Listener
}

func (suite *ServerTestSuite) SetupTest() {
	t := suite.T()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	suite.NoError(err)
	suite.listener = l

	// Superuser permitted to produce and consume
	suite.rootConn, suite.rootClient, _ = newClient(t, config.RootClientCertFile, config.RootClientKeyFile, l)
	// Client who is not permitted to do anything
	suite.nobodyConn, suite.nobodyClient, _ = newClient(t, config.NobodyClientCertFile, config.NobodyClientKeyFile, l)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: config.ServerCertFile,
		KeyFile:  config.ServerKeyFile,
		CAFile:   config.CAFile,
		Server:   true,
	})
	suite.NoError(err)
	serverCreds := credentials.NewTLS(serverTLSConfig)
	dir, err := ioutil.TempDir("", "server-test")
	suite.NoError(err)
	defer os.RemoveAll(dir)
	suite.NoError(err)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)
	config := &Config{
		Authorizer: authorizer,
	}

	suite.server = NewGRPCServer(config, grpc.Creds(serverCreds))
	go func() {
		suite.server.Serve(l)
	}()
}

func (suite *ServerTestSuite) TearDownTest() {
	suite.server.Stop()
	suite.rootConn.Close()
	suite.nobodyConn.Close()
	suite.listener.Close()
}

func (suite *ServerTestSuite) TestCreateReadProfile() {
	client := suite.rootClient
	ctx := context.Background()
	payload := &api.CreateProfileReq{FirstName: "Foo", LastName: "Bar"}

	createResponse, err := client.CreateProfile(ctx, payload)
	suite.NoError(err)
	suite.NotEmpty(createResponse.Id)

	id := createResponse.Id
	readResponse, err := client.ReadProfile(ctx, &api.ReadProfileReq{Id: id})
	suite.NoError(err)
	suite.Equal(id, readResponse.Id)
}

func newClient(t *testing.T, crtPath, keyPath string, l net.Listener) (*grpc.ClientConn, api.ProfileServiceClient, []grpc.DialOption) {
	t.Helper()
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: crtPath,
		KeyFile:  keyPath,
		CAFile:   config.CAFile,
		Server:   false,
	})
	if err != nil {
		t.Fatal("Failed to setup TLS config: ", err)
	}
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	conn, err := grpc.Dial(l.Addr().String(), opts...)
	if err != nil {
		t.Fatal("Failed to create client connection to target: ", err)
	}
	client := api.NewProfileServiceClient(conn)
	return conn, client, opts
}
