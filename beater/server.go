// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beater

import (
	"context"
	"net"
	"net/http"
	"net/url"

	"go.elastic.co/apm"
	"golang.org/x/net/netutil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/pkg/errors"

	"github.com/elastic/apm-server/beater/config"
	"github.com/elastic/apm-server/beater/internal/h2mux"
	"github.com/elastic/apm-server/beater/jaeger"
	"github.com/elastic/apm-server/publish"
)

// RunServerFunc is a function which runs the APM Server until a
// fatal error occurs, or the context is cancelled.
type RunServerFunc func(context.Context, ServerParams) error

// ServerParams holds parameters for running the APM Server.
type ServerParams struct {
	// Config is the configuration used for running the APM Server.
	Config *config.Config

	// Logger is the logger for the beater component.
	Logger *logp.Logger

	// Tracer is an apm.Tracer that the APM Server may use
	// for self-instrumentation.
	Tracer *apm.Tracer

	// Reporter is the publish.Reporter that the APM Server
	// should use for reporting events.
	Reporter publish.Reporter
}

// runServer runs the APM Server until a fatal error occurs, or ctx is cancelled.
func runServer(ctx context.Context, args ServerParams) error {
	srv, err := newServer(args.Logger, args.Config, args.Tracer, args.Reporter)
	if err != nil {
		return err
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			srv.stop()
		case <-done:
		}
	}()
	return srv.run()
}

type server struct {
	logger *logp.Logger
	cfg    *config.Config

	httpServer   *httpServer
	grpcServer   *grpc.Server
	jaegerServer *jaeger.Server
	reporter     publish.Reporter
}

func newServer(logger *logp.Logger, cfg *config.Config, tracer *apm.Tracer, reporter publish.Reporter) (server, error) {
	httpServer, err := newHTTPServer(logger, cfg, tracer, reporter)
	if err != nil {
		return server{}, err
	}
	grpcServer, err := newGRPCServer(logger, cfg, tracer)
	if err != nil {
		return server{}, err
	}
	jaegerServer, err := jaeger.NewServer(logger, cfg, grpcServer, reporter)
	if err != nil {
		return server{}, err
	}
	return server{
		logger:       logger,
		cfg:          cfg,
		httpServer:   httpServer,
		grpcServer:   grpcServer,
		jaegerServer: jaegerServer,
		reporter:     reporter,
	}, nil
}

func (s server) run() error {
	s.logger.Infof("Starting apm-server [%s built %s]. Hit CTRL-C to stop it.", version.Commit(), version.BuildTime())

	httpListener, grpcListener, err := s.listen()
	if err != nil {
		return err
	}

	s.logger.Infof("http: %s, grpc: %s", httpListener.Addr(), grpcListener.Addr())

	var g errgroup.Group
	if s.grpcServer != nil {
		g.Go(func() error { return s.grpcServer.Serve(grpcListener) })
	}
	if s.httpServer != nil {
		g.Go(func() error { return s.httpServer.start(httpListener) })
	}
	if err := g.Wait(); err != http.ErrServerClosed {
		return err
	}
	s.logger.Infof("Server stopped")
	return nil
}

func (s server) stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		s.httpServer.stop()
	}
}

func (s server) listen() (httpListener, grpcListener net.Listener, _ error) {
	httpListener, err := s.listenHTTP()
	if err != nil {
		return nil, nil, err
	}
	if s.cfg.MaxConnections > 0 {
		s.logger.Infof("Connection limit set to: %d", s.cfg.MaxConnections)
		httpListener = netutil.LimitListener(httpListener, s.cfg.MaxConnections)
	}

	addr := httpListener.Addr()
	if addr.Network() == "tcp" {
		s.logger.Infof("Listening on: %s", addr)
	} else {
		s.logger.Infof("Listening on: %s:%s", addr.Network(), addr.String())
	}

	// TODO(axw) h2mux.ConfigureServer assumes we're using TLS, as it
	// relies on ALPN upgrade. We also should support muxing when TLS is
	// disabled by sniffing the content. We don't do this in general as
	// it breaks TLS-related features in net/http.
	grpcListener, err = h2mux.ConfigureServer(s.httpServer.Server, nil)
	if err != nil {
		httpListener.Close()
		return nil, nil, errors.Wrap(err, "failed to configure h2mux")
	}
	return httpListener, grpcListener, nil
}

func (s server) listenHTTP() (net.Listener, error) {
	if url, err := url.Parse(s.cfg.Host); err == nil && url.Scheme == "unix" {
		return net.Listen("unix", url.Path)
	}

	const network = "tcp"
	addr := s.cfg.Host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		// Tack on a port if SplitHostPort fails on what should be a
		// tcp network address. If splitting failed because there were
		// already too many colons, one more won't change that.
		addr = net.JoinHostPort(addr, config.DefaultPort)
	}
	return net.Listen(network, addr)
}
