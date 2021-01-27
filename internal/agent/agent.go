package agent

import (
	"crypto/tls"
	"fmt"
	"github.com/joshjon/go-profiles/internal/auth"
	"github.com/joshjon/go-profiles/internal/config"
	"github.com/joshjon/go-profiles/internal/server"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"sync"
)

type Config struct {
	ServerTLSConfig *tls.Config
	PeerTLSConfig   *tls.Config
	// DataDir stores the log and raft data.
	DataDir string
	// BindAddr is the address serf runs on.
	BindAddr string
	// RPCPort is the port for client (and Raft) connections.
	RPCPort int
	// Raft server id.
	NodeName string
	// Bootstrap should be set to true when starting the first node of the cluster.
	StartJoinAddrs []string
	ACLModelFile   string
	ACLPolicyFile  string
	Bootstrap      bool
}

type Agent struct {
	Config Config

	mux    cmux.CMux
	server *grpc.Server

	shutdown     bool
	shutdowns    chan struct{}
	shutdownLock sync.Mutex
}

func (c Config) RPCAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.RPCPort), nil
}

func New(config Config) (*Agent, error) {
	agent := &Agent{
		Config:    config,
		shutdowns: make(chan struct{}),
	}
	setup := []func() error{
		agent.setupMux,
		agent.setupServer,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	go agent.serve()
	return agent, nil
}

func (a *Agent) setupMux() error {
	rpcAddr := fmt.Sprintf(
		":%d",
		a.Config.RPCPort,
	)
	ln, err := net.Listen("tcp", rpcAddr)
	fmt.Printf("localhost%s\n", rpcAddr)
	if err != nil {
		return err
	}
	a.mux = cmux.New(ln)
	return nil
}

func (a *Agent) serve() error {
	if err := a.mux.Serve(); err != nil {
		_ = a.Shutdown()
		return err
	}
	return nil
}

func (a *Agent) setupServer() error {
	// Hardcoded files, refer to tutorial for CLI params
	authorizer := auth.New(
		config.ACLModelFile,
		config.ACLPolicyFile,
	)
	serverConfig := &server.Config{
		Authorizer: authorizer,
	}
	var opts []grpc.ServerOption

	// Hardcoded files, refer to tutorial for CLI params
	conf := config.TLSConfig{
		CertFile: config.ServerCertFile,
		KeyFile:  config.ServerKeyFile,
		CAFile:   config.CAFile,
		Server:   true,
	}
	serverTLSConfig, _ := config.SetupTLSConfig(conf)
	creds := credentials.NewTLS(serverTLSConfig)
	opts = append(opts, grpc.Creds(creds))

	var err error
	a.server = server.NewGRPCServer(serverConfig, opts...)

	grpcLn := a.mux.Match(cmux.Any())
	go func() {
		if err := a.server.Serve(grpcLn); err != nil {
			_ = a.Shutdown()
		}
	}()
	return err
}

func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()
	if a.shutdown {
		return nil
	}
	a.shutdown = true
	close(a.shutdowns)

	shutdown := []func() error{
		func() error {
			a.server.GracefulStop()
			return nil
		},
	}
	for _, fn := range shutdown {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}
