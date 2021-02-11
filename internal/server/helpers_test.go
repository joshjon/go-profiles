package server

import (
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/auth"
	"github.com/joshjon/go-profiles/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const ServerAddress string = "127.0.0.1:50051"

type Client struct {
	Conn    *grpc.ClientConn
	Client  api.ProfileServiceClient
	options []grpc.DialOption
}

func NewProfileServiceClient(address string, crtPath string, keyPath string, CAFile string) (*Client, error) {
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: crtPath,
		KeyFile:  keyPath,
		CAFile:   CAFile,
		Server:   false,
	})
	if err != nil {
		return nil, err
	}
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, err
	}
	client := api.NewProfileServiceClient(conn)
	return &Client{Conn: conn, Client: client, options: opts}, nil
}

func NewTestGRPCServer(certFile string, keyFile string, CAFile string, ACLModelFile string, ACLPolicyFile string) (*grpc.Server, error) {
	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   CAFile,
		Server:   true,
	})
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Authorizer: auth.New(ACLModelFile, ACLPolicyFile),
	}
	serverCreds := credentials.NewTLS(serverTLSConfig)
	return NewGRPCServer(cfg, grpc.Creds(serverCreds)), nil
}
