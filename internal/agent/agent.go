package agent

import (
	"crypto/tls"
	"fmt"
	"github.com/joshjon/go-profiles/internal/auth"
	"github.com/joshjon/go-profiles/internal/server"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"sync"
)

type Config struct {
	ServerTLSConfig *tls.Config
	//PeerTLSConfig   *tls.Config
	RPCPort int
	// server id
	NodeName      string
	ACLModelFile  string
	ACLPolicyFile string
}

type Agent struct {
	Config       Config
	mux          cmux.CMux
	server       *grpc.Server
	shutdown     bool
	shutdownLock sync.Mutex
}

func (c Config) RPCAddr() (string, error) {
	return fmt.Sprintf("%s:%d", "localhost", c.RPCPort), nil
}

func New(config Config) (*Agent, error) {
	agent := &Agent{
		Config:    config,
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
	rpcAddr := fmt.Sprintf(":%d", a.Config.RPCPort)
	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return err
	}
	a.mux = cmux.New(ln)
	return nil
}

func (a *Agent) serve() error {
	if err := a.mux.Serve(); err != nil {
		a.Shutdown()
		return err
	}
	return nil
}

func (a *Agent) setupServer() error {
	authorizer := auth.New(a.Config.ACLModelFile, a.Config.ACLPolicyFile)
	serverConfig := &server.Config{Authorizer: authorizer}
	var opts []grpc.ServerOption

	if a.Config.ServerTLSConfig != nil {
		creds := credentials.NewTLS(a.Config.ServerTLSConfig)
		opts = append(opts, grpc.Creds(creds))
	}

	a.server = server.NewGRPCServer(serverConfig, opts...)
	grpcLn := a.mux.Match(cmux.Any())

	var err error
	go func() {
		if err = a.server.Serve(grpcLn); err != nil {
			a.Shutdown()
		}
	}()
	return err
}

func (a *Agent) Shutdown() {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()
	if a.shutdown {
		return
	}
	a.shutdown = true
	a.server.GracefulStop()
}
