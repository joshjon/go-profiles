package main

import (
	"context"
	"flag"
	api "github.com/joshjon/go-profiles/api/v1"
	"github.com/joshjon/go-profiles/internal/config"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	serverAddr = flag.String("server_addr", "localhost:8400", "The server address in the format of host:port")
)

func createRead(client api.ProfileServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	profile := &api.Profile{
		Id:        rand.Uint64(),
		FirstName: "foo",
		LastName:  "bar",
	}
	response, err := client.CreateProfile(ctx, &api.CreateProfileReq{Profile: profile})
	if err != nil {
		log.Fatalf("Error occured: %v", err)
	}
	log.Println(response.Profile)
}

func main() {
	flag.Parse()

	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile: config.RootClientCertFile,
		KeyFile:  config.RootClientKeyFile,
		CAFile:   config.CAFile,
		Server:   false,
	})

	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	conn, err := grpc.Dial(*serverAddr, opts...)

	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	defer conn.Close()
	client := api.NewProfileServiceClient(conn)
	createRead(client)
}
