package server

import (
	"context"
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/config"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"testing"
)

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type ServerTestSuite struct {
	suite.Suite
	server       *grpc.Server
	rootClient   *Client
	nobodyClient *Client
	listener     net.Listener
}

func (suite *ServerTestSuite) SetupTest() {
	listener, err := net.Listen("tcp", ServerAddress)
	suite.NoError(err)
	suite.listener = listener
	// Superuser permitted to produce and consume
	suite.rootClient, err = NewProfileServiceClient(ServerAddress, config.RootClientCertFile, config.RootClientKeyFile, config.CAFile)
	suite.NoError(err)
	// Client who is not permitted to do anything
	suite.nobodyClient, err = NewProfileServiceClient(ServerAddress, config.NobodyClientCertFile, config.NobodyClientKeyFile, config.CAFile)
	suite.NoError(err)
	suite.server, err = NewTestGRPCServer(config.ServerCertFile, config.ServerKeyFile, config.CAFile, config.ACLModelFile, config.ACLPolicyFile)
	suite.NoError(err)
	go suite.server.Serve(listener)
}

func (suite *ServerTestSuite) TearDownTest() {
	suite.server.Stop()
	suite.rootClient.Conn.Close()
	suite.nobodyClient.Conn.Close()
	suite.listener.Close()
}

func (suite *ServerTestSuite) TestCreateReadProfile() {
	client := suite.rootClient.Client
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

func (suite *ServerTestSuite) TestProfileNotFound() {
	ctx := context.Background()
	client := suite.rootClient.Client
	response, err := client.ReadProfile(ctx, &api.ReadProfileReq{Id: "Foo"})
	suite.Nil(response)
	code, expected := status.Code(err), status.Code(api.ErrProfileNotFound{}.GRPCStatus().Err())
	suite.Equal(code, expected)
}

func (suite *ServerTestSuite) TestUnauthorized() {
	client := suite.nobodyClient.Client
	ctx := context.Background()
	payload := &api.CreateProfileReq{FirstName: "Foo", LastName: "Bar"}

	createResponse, err := client.CreateProfile(ctx, payload)
	suite.Nil(createResponse)
	code, expectedCode := status.Code(err), codes.PermissionDenied
	suite.Equal(code, expectedCode)

	readResponse, err := client.ReadProfile(ctx, &api.ReadProfileReq{Id: "foo"})
	suite.Nil(readResponse)
	code, expectedCode = status.Code(err), codes.PermissionDenied
	suite.Equal(code, expectedCode)
}
