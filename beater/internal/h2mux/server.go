package h2mux

import (
	"context"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	HTTP  *http.Server
	HTTP2 *http2.Server
	GRPC  *grpc.Server

	configureOnce sync.Once
	configureErr  error
	grpcListener  net.Listener
}

func NewServer(srv *http.Server, grpcOptions ...grpc.ServerOption) *Server {
	s := &Server{
		HTTP:  srv,
		HTTP2: &http2.Server{},
		GRPC:  grpc.NewServer(grpcOptions...),
	}
	s.HTTP.RegisterOnShutdown(s.GRPC.GracefulStop)
	return s
}

func (s *Server) Serve(lis net.Listener) error {
	s.configureOnce.Do(func() {
		// TODO(axw) handle TLS vs. non-TLS here
		s.grpcListener, s.configureErr = ConfigureServer(s.HTTP, s.HTTP2)
	})
	if s.configureErr != nil {
		return s.configureErr
	}
	var g errgroup.Group
	g.Go(func() error {
		return s.HTTP.Serve(lis)
	})
	g.Go(func() error {
		return s.GRPC.Serve(s.grpcListener)
	})
	return g.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.GRPC.GracefulStop()
	return s.HTTP.Shutdown(ctx)
}
