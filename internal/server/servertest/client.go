package servertest

import (
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
)

type profileClient struct {
	clientConn *grpc.ClientConn
	client     api.ProfileServiceClient
	options    []grpc.DialOption
}

func NewProfileServiceClient(crtPath, keyPath string, l net.Listener) (*profileClient, error) {
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: crtPath,
		KeyFile:  keyPath,
		CAFile:   config.CAFile,
		Server:   false,
	})
	if err != nil {
		return nil, err
	}
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	conn, err := grpc.Dial(l.Addr().String(), opts...)
	if err != nil {
		return nil, err
	}
	client := api.NewProfileServiceClient(conn)
	return &profileClient{clientConn: conn, client: client, options: opts}, nil
}
