package main

import (
	"context"
	"flag"
	"fmt"
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/config"
	"log"
	"net"
	"path/filepath"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr       = flag.String("addr", "localhost", "The address of the server to connect to")
	port       = flag.String("port", "8400", "The port to connect to")
	caCert     = certFile("ca.pem")
	clientCert = certFile("root-client.pem")
	clientKey  = certFile("root-client-key.pem")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	target := net.JoinHostPort(*addr, *port)

	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: clientCert,
		KeyFile:  clientKey,
		CAFile:   caCert,
		Server:   false,
	})

	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	conn, err := grpc.DialContext(ctx, target, opts...)

	if err != nil {
		log.Fatalf("Failed to dial server: %v", err)
	}

	defer conn.Close()
	client := api.NewProfileServiceClient(conn)

	created := createProfile(client)
	fmt.Printf("Created profile: %+v\n", created)
	fmt.Printf("Retrieved profile: %+v\n", readProfile(client, created.Id))
}

func createProfile(client api.ProfileServiceClient) *api.Profile {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &api.ProfileDto{
		FirstName: "foo",
		LastName:  "bar",
	}
	response, err := client.CreateProfile(ctx, req)
	if err != nil {
		log.Fatalf("Error occured: %v", err)
	}
	return response
}

func readProfile(client api.ProfileServiceClient, id string) *api.Profile {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &api.ReadProfileReq{Id: id}
	response, err := client.ReadProfile(ctx, req)
	if err != nil {
		log.Fatalf("Error occured: %v", err)
	}
	return response
}

func certFile(filename string) string {
	_, f, _, _ := runtime.Caller(0)
	projectPath := filepath.Join(filepath.Dir(f), "../..")
	return filepath.Join(projectPath, "certs", filename)
}
