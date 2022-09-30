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

package apmserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.elastic.co/apm/module/apmgrpc/v2"
	"go.elastic.co/apm/module/apmhttp/v2"
	"go.elastic.co/apm/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/instrumentation"
	"github.com/elastic/beats/v7/libbeat/licenser"
	"github.com/elastic/beats/v7/libbeat/outputs"
	esoutput "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	agentconfig "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/go-ucfg"

	"github.com/elastic/apm-server/internal/agentcfg"
	"github.com/elastic/apm-server/internal/apmserver/auth"
	"github.com/elastic/apm-server/internal/apmserver/config"
	"github.com/elastic/apm-server/internal/apmserver/interceptors"
	javaattacher "github.com/elastic/apm-server/internal/apmserver/java_attacher"
	"github.com/elastic/apm-server/internal/apmserver/ratelimit"
	"github.com/elastic/apm-server/internal/beatcmd"
	"github.com/elastic/apm-server/internal/elasticsearch"
	"github.com/elastic/apm-server/internal/kibana"
	"github.com/elastic/apm-server/internal/model"
	"github.com/elastic/apm-server/internal/model/modelindexer"
	"github.com/elastic/apm-server/internal/model/modelprocessor"
	"github.com/elastic/apm-server/internal/publish"
	"github.com/elastic/apm-server/internal/sourcemap"
)

var libbeatMonitoringRegistry = monitoring.Default.GetRegistry("libbeat")

// CreatorParams holds parameters for creating beat.Beaters.
type CreatorParams struct {
	// Logger is a logger to use in Beaters created by the beat.Creator.
	//
	// If Logger is nil, logp.NewLogger will be used to create a new one.
	Logger *logp.Logger

	// WrapServer is optional, and if provided, will be called to wrap
	// the ServerParams and RunServerFunc used to run the APM Server.
	//
	// The WrapServer function may modify ServerParams, for example by
	// wrapping the BatchProcessor with additional processors. Similarly,
	// WrapServer may wrap the RunServerFunc to run additional goroutines
	// along with the server.
	//
	// WrapServer may keep a reference to the provided ServerParams's
	// BatchProcessor for asynchronous event publication, such as for
	// aggregated metrics. All other events (i.e. those decoded from
	// agent payloads) should be sent to the BatchProcessor in the
	// ServerParams provided to RunServerFunc; this BatchProcessor will
	// have rate-limiting, authorization, and data preprocessing applied.
	WrapServer WrapServerFunc
}

// TODO(axw) move RunnerParams & Runner to beater package?
func NewRunnerFunc(wrapServer WrapServerFunc) beatcmd.NewRunnerFunc {
	return func(args beatcmd.RunnerParams) (beatcmd.Runner, error) {
		var unpackedConfig struct {
			APMServer  *agentconfig.C        `config:"apm-server"`
			Output     agentconfig.Namespace `config:"output"`
			Fleet      *config.Fleet         `config:"fleet"`
			DataStream struct {
				Namespace string `config:"namespace"`
			} `config:"data_stream"`
		}
		if err := args.Config.Unpack(&unpackedConfig); err != nil {
			return nil, err
		}

		var elasticsearchOutputConfig *agentconfig.C
		if unpackedConfig.Output.Name() == "elasticsearch" {
			elasticsearchOutputConfig = unpackedConfig.Output.Config()
		}
		cfg, err := config.NewConfig(unpackedConfig.APMServer, elasticsearchOutputConfig)
		if err != nil {
			return nil, err
		}
		if unpackedConfig.DataStream.Namespace != "" {
			cfg.DataStreams.Namespace = unpackedConfig.DataStream.Namespace
		}

		// We start the listener in the constructor, before Run is invoked,
		// to ensure zero downtime while any existing Runner is stopped.
		listener, err := listen(cfg, args.Logger)
		if err != nil {
			return nil, err
		}
		return &serverRunner{
			wrapServer: wrapServer,
			info:       args.Info,
			logger:     args.Logger,
			rawConfig:  args.Config,

			config:       cfg,
			fleetConfig:  unpackedConfig.Fleet,
			outputConfig: unpackedConfig.Output,

			listener: listener,
		}, nil
	}
}

type serverRunner struct {
	wrapServer WrapServerFunc
	info       beat.Info
	logger     *logp.Logger
	rawConfig  *agentconfig.C

	config                    *config.Config
	fleetConfig               *config.Fleet
	outputConfig              agentconfig.Namespace
	elasticsearchOutputConfig *agentconfig.C

	listener net.Listener
}

func (r *serverRunner) Run(ctx context.Context) error {
	defer r.listener.Close()

	// backgroundContext is a context to use in operations that should
	// block until shutdown, and will be cancelled after the shutdown
	// timeout.
	backgroundContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-ctx.Done()
		r.logger.Infof(
			"stopping apm-server... waiting maximum of %s for queues to drain",
			r.config.ShutdownTimeout,
		)
		time.AfterFunc(r.config.ShutdownTimeout, cancel)
	}()

	if r.config.Pprof.Enabled {
		// Profiling rates should be set once, early on in the program.
		runtime.SetBlockProfileRate(r.config.Pprof.BlockProfileRate)
		runtime.SetMutexProfileFraction(r.config.Pprof.MutexProfileRate)
		if r.config.Pprof.MemProfileRate > 0 {
			runtime.MemProfileRate = r.config.Pprof.MemProfileRate
		}
	}

	tracer, tracerServer, err := r.initTracing()
	if err != nil {
		return err
	}
	if tracerServer != nil {
		defer tracerServer.Close()
	}
	defer tracer.Close()

	// Send config to telemetry.
	recordAPMServerConfig(r.config)

	var kibanaClient kibana.Client
	if r.config.Kibana.Enabled {
		kibanaClient = kibana.NewConnectingClient(r.config.Kibana.ClientConfig)
	}

	// ELASTIC_AGENT_CLOUD is set when runningi n Elastic Cloud.
	isElasticCloud := os.Getenv("ELASTIC_AGENT_CLOUD") != ""
	if isElasticCloud && r.config.Kibana.Enabled {
		go func() {
			if err := kibana.SendConfig(ctx, kibanaClient, (*ucfg.Config)(r.rawConfig)); err != nil {
				r.logger.Infof("failed to upload config to kibana: %v", err)
			}
		}()
	}

	if r.config.JavaAttacherConfig.Enabled {
		if !isElasticCloud {
			go func() {
				attacher, err := javaattacher.New(r.config.JavaAttacherConfig)
				if err != nil {
					r.logger.Errorf("failed to start java attacher: %v", err)
					return
				}
				if err := attacher.Run(ctx); err != nil {
					r.logger.Errorf("failed to run java attacher: %v", err)
				}
			}()
		} else {
			r.logger.Error("java attacher not supported in cloud environments")
		}
	}

	g, ctx := errgroup.WithContext(ctx)

	// Ensure the libbeat output and go-elasticsearch clients do not index
	// any events to Elasticsearch before the integration is ready.
	publishReady := make(chan struct{})
	drain := make(chan struct{})
	g.Go(func() error {
		if err := r.waitReady(ctx, kibanaClient, tracer); err != nil {
			// One or more preconditions failed; drop events.
			close(drain)
			return errors.Wrap(err, "error waiting for server to be ready")
		}
		// All preconditions have been met; start indexing documents
		// into elasticsearch.
		close(publishReady)
		return nil
	})
	callbackUUID, err := esoutput.RegisterConnectCallback(func(*eslegclient.Connection) error {
		select {
		case <-publishReady:
			return nil
		default:
		}
		return errors.New("not ready for publishing events")
	})
	if err != nil {
		return err
	}
	defer esoutput.DeregisterConnectCallback(callbackUUID)
	newElasticsearchClient := func(cfg *elasticsearch.Config) (elasticsearch.Client, error) {
		httpTransport, err := elasticsearch.NewHTTPTransport(cfg)
		if err != nil {
			return nil, err
		}
		transport := &waitReadyRoundTripper{Transport: httpTransport, ready: publishReady, drain: drain}
		return elasticsearch.NewClientParams(elasticsearch.ClientParams{
			Config:    cfg,
			Transport: transport,
			RetryOnError: func(_ *http.Request, err error) bool {
				return !errors.Is(err, errServerShuttingDown)
			},
		})
	}

	var sourcemapFetcher sourcemap.Fetcher
	if r.config.RumConfig.Enabled && r.config.RumConfig.SourceMapping.Enabled {
		fetcher, err := newSourcemapFetcher(
			r.info, r.config.RumConfig.SourceMapping, r.fleetConfig,
			kibanaClient, newElasticsearchClient,
		)
		if err != nil {
			return err
		}
		cachingFetcher, err := sourcemap.NewCachingFetcher(
			fetcher, r.config.RumConfig.SourceMapping.Cache.Expiration,
		)
		if err != nil {
			return err
		}
		sourcemapFetcher = cachingFetcher
	}

	// Create the runServer function. We start with newBaseRunServer, and then
	// wrap depending on the configuration in order to inject behaviour.
	runServer := newBaseRunServer(r.listener)
	if tracerServer != nil {
		runServer = runServerWithTracerServer(runServer, tracerServer, tracer)
	}

	authenticator, err := auth.NewAuthenticator(r.config.AgentAuth)
	if err != nil {
		return err
	}

	ratelimitStore, err := ratelimit.NewStore(
		r.config.AgentAuth.Anonymous.RateLimit.IPLimit,
		r.config.AgentAuth.Anonymous.RateLimit.EventLimit,
		3, // burst mulitiplier
	)
	if err != nil {
		return err
	}

	// Note that we intentionally do not use a grpc.Creds ServerOption
	// even if TLS is enabled, as TLS is handled by the net/http server.
	gRPCLogger := r.logger.Named("grpc")
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery(), apmgrpc.WithTracer(tracer)),
		interceptors.ClientMetadata(),
		interceptors.Logging(gRPCLogger),
		interceptors.Metrics(gRPCLogger),
		interceptors.Timeout(),
		interceptors.Auth(authenticator),
		interceptors.AnonymousRateLimit(ratelimitStore),
	))

	// Create the BatchProcessor chain that is used to process all events,
	// including the metrics aggregated by APM Server.
	finalBatchProcessor, closeFinalBatchProcessor, err := r.newFinalBatchProcessor(tracer, newElasticsearchClient)
	if err != nil {
		return err
	}
	batchProcessor := modelprocessor.Chained{
		// Ensure all events have observer.*, ecs.*, and data_stream.* fields added,
		// and are counted in metrics. This is done in the final processors to ensure
		// aggregated metrics are also processed.
		newObserverBatchProcessor(r.info),
		&modelprocessor.SetDataStream{Namespace: r.config.DataStreams.Namespace},
		modelprocessor.NewEventCounter(monitoring.Default.GetRegistry("apm-server")),

		// The server always drops non-RUM unsampled transactions. We store RUM unsampled
		// transactions as they are needed by the User Experience app, which performs
		// aggregations over dimensions that are not available in transaction metrics.
		//
		// It is important that this is done just before calling the publisher to
		// avoid affecting aggregations.
		modelprocessor.NewDropUnsampled(false /* don't drop RUM unsampled transactions*/),
		modelprocessor.DroppedSpansStatsDiscarder{},
		finalBatchProcessor,
	}

	agentConfigReporter := agentcfg.NewReporter(
		newAgentConfigFetcher(r.config, kibanaClient),
		batchProcessor, 30*time.Second,
	)
	g.Go(func() error {
		return agentConfigReporter.Run(ctx)
	})

	serverParams := ServerParams{
		UUID:                   r.info.ID,
		Config:                 r.config,
		Managed:                r.fleetConfig != nil,
		Namespace:              r.config.DataStreams.Namespace, // TODO(axw) remove and use config field?
		Logger:                 r.logger,
		Tracer:                 tracer,
		Authenticator:          authenticator,
		RateLimitStore:         ratelimitStore,
		BatchProcessor:         batchProcessor,
		AgentConfig:            agentConfigReporter,
		SourcemapFetcher:       sourcemapFetcher,
		PublishReady:           publishReady,
		KibanaClient:           kibanaClient,
		NewElasticsearchClient: newElasticsearchClient,
		GRPCServer:             grpcServer,
	}
	if r.wrapServer != nil {
		// Wrap the serverParams and runServer function, enabling
		// injection of behaviour into the processing chain.
		serverParams, runServer, err = r.wrapServer(serverParams, runServer)
		if err != nil {
			return err
		}
	}

	// Add pre-processing batch processors to the beginning of the chain,
	// applying only to the events that are decoded from agent/client payloads.
	preBatchProcessors := modelprocessor.Chained{
		// Add a model processor that rate limits, and checks authorization for the
		// agent and service for each event. These must come at the beginning of the
		// processor chain.
		model.ProcessBatchFunc(rateLimitBatchProcessor),
		model.ProcessBatchFunc(authorizeEventIngestProcessor),

		// Pre-process events before they are sent to the final processors for
		// aggregation, sampling, and indexing.
		modelprocessor.SetHostHostname{},
		modelprocessor.SetServiceNodeName{},
		modelprocessor.SetMetricsetName{},
		modelprocessor.SetGroupingKey{},
		modelprocessor.SetErrorMessage{},
		modelprocessor.SetUnknownSpanType{},
	}
	if r.config.DefaultServiceEnvironment != "" {
		preBatchProcessors = append(preBatchProcessors, &modelprocessor.SetDefaultServiceEnvironment{
			DefaultServiceEnvironment: r.config.DefaultServiceEnvironment,
		})
	}
	serverParams.BatchProcessor = append(preBatchProcessors, serverParams.BatchProcessor)

	g.Go(func() error {
		return runServer(ctx, serverParams)
	})

	result := g.Wait()
	if err := closeFinalBatchProcessor(backgroundContext); err != nil {
		result = multierror.Append(result, err)
	}
	return result
}

// waitReady waits until the server is ready to index events.
func (r *serverRunner) waitReady(ctx context.Context, kibanaClient kibana.Client, tracer *apm.Tracer) error {
	var preconditions []func(context.Context) error
	var esOutputClient elasticsearch.Client
	if r.elasticsearchOutputConfig != nil {
		esConfig := elasticsearch.DefaultConfig()
		err := r.elasticsearchOutputConfig.Unpack(&esConfig)
		if err != nil {
			return err
		}
		esOutputClient, err = elasticsearch.NewClient(esConfig)
		if err != nil {
			return err
		}
	}

	// libbeat and go-elasticsearch both ensure a minimum level of Basic.
	//
	// If any configured features require a higher license level, add a
	// precondition which checks this.
	if esOutputClient != nil {
		requiredLicenseLevel := licenser.Basic
		licensedFeature := ""
		if r.config.Sampling.Tail.Enabled {
			requiredLicenseLevel = licenser.Platinum
			licensedFeature = "tail-based sampling"
		}
		if requiredLicenseLevel > licenser.Basic {
			preconditions = append(preconditions, func(ctx context.Context) error {
				license, err := elasticsearch.GetLicense(ctx, esOutputClient)
				if err != nil {
					return errors.Wrap(err, "error getting Elasticsearch licensing information")
				}
				if licenser.IsExpired(license) {
					return errors.New("Elasticsearch license is expired")
				}
				if license.Type == licenser.Trial || license.Cover(requiredLicenseLevel) {
					return nil
				}
				return fmt.Errorf(
					"invalid license level %s: %s requires license level %s",
					license.Type, licensedFeature, requiredLicenseLevel,
				)
			})
		}
		preconditions = append(preconditions, func(ctx context.Context) error {
			return queryClusterUUID(ctx, esOutputClient)
		})
	}

	// When running standalone with data streams enabled, by default we will add
	// a precondition that ensures the integration is installed.
	fleetManaged := r.fleetConfig != nil
	if !fleetManaged && r.config.DataStreams.WaitForIntegration {
		if kibanaClient == nil && esOutputClient == nil {
			return errors.New("cannot wait for integration without either Kibana or Elasticsearch config")
		}
		preconditions = append(preconditions, func(ctx context.Context) error {
			return checkIntegrationInstalled(ctx, kibanaClient, esOutputClient, r.logger)
		})
	}

	if len(preconditions) == 0 {
		return nil
	}
	check := func(ctx context.Context) error {
		for _, pre := range preconditions {
			if err := pre(ctx); err != nil {
				return err
			}
		}
		return nil
	}
	return waitReady(ctx, r.config.WaitReadyInterval, tracer, r.logger, check)
}

// newFinalBatchProcessor returns the final model.BatchProcessor that publishes events,
// and a cleanup function which should be called on server shutdown. If the output is
// "elasticsearch", then we use modelindexer; otherwise we use the libbeat publisher.
func (r *serverRunner) newFinalBatchProcessor(
	tracer *apm.Tracer,
	newElasticsearchClient func(cfg *elasticsearch.Config) (elasticsearch.Client, error),
) (model.BatchProcessor, func(context.Context) error, error) {
	if r.elasticsearchOutputConfig == nil {
		return r.newLibbeatFinalBatchProcessor(tracer)
	}

	var esConfig struct {
		*elasticsearch.Config `config:",inline"`
		FlushBytes            string        `config:"flush_bytes"`
		FlushInterval         time.Duration `config:"flush_interval"`
		MaxRequests           int           `config:"max_requests"`
	}
	esConfig.FlushInterval = time.Second
	esConfig.Config = elasticsearch.DefaultConfig()
	if err := r.elasticsearchOutputConfig.Unpack(&esConfig); err != nil {
		return nil, nil, err
	}

	var flushBytes int
	if esConfig.FlushBytes != "" {
		b, err := humanize.ParseBytes(esConfig.FlushBytes)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse flush_bytes")
		}
		flushBytes = int(b)
	}
	client, err := newElasticsearchClient(esConfig.Config)
	if err != nil {
		return nil, nil, err
	}
	indexer, err := modelindexer.New(client, modelindexer.Config{
		CompressionLevel: esConfig.CompressionLevel,
		FlushBytes:       flushBytes,
		FlushInterval:    esConfig.FlushInterval,
		Tracer:           tracer,
		MaxRequests:      esConfig.MaxRequests,
	})
	if err != nil {
		return nil, nil, err
	}

	// Install our own libbeat-compatible metrics callback which uses the modelindexer stats.
	// All the metrics below are required to be reported to be able to display all relevant
	// fields in the Stack Monitoring UI.
	monitoring.Default.Remove("libbeat")
	monitoring.NewFunc(monitoring.Default, "libbeat.output.write", func(_ monitoring.Mode, v monitoring.Visitor) {
		v.OnRegistryStart()
		defer v.OnRegistryFinished()
		v.OnKey("bytes")
		v.OnInt(indexer.Stats().BytesTotal)
	})
	outputType := monitoring.NewString(monitoring.Default.GetRegistry("libbeat.output"), "type")
	outputType.Set("elasticsearch")
	monitoring.NewFunc(monitoring.Default, "libbeat.output.events", func(_ monitoring.Mode, v monitoring.Visitor) {
		v.OnRegistryStart()
		defer v.OnRegistryFinished()
		stats := indexer.Stats()
		v.OnKey("acked")
		v.OnInt(stats.Indexed)
		v.OnKey("active")
		v.OnInt(stats.Active)
		v.OnKey("batches")
		v.OnInt(stats.BulkRequests)
		v.OnKey("failed")
		v.OnInt(stats.Failed)
		v.OnKey("toomany")
		v.OnInt(stats.TooManyRequests)
		v.OnKey("total")
		v.OnInt(stats.Added)
	})
	monitoring.NewFunc(monitoring.Default, "libbeat.pipeline.events", func(_ monitoring.Mode, v monitoring.Visitor) {
		v.OnRegistryStart()
		defer v.OnRegistryFinished()
		v.OnKey("total")
		v.OnInt(indexer.Stats().Added)
	})
	monitoring.Default.Remove("output")
	monitoring.NewFunc(monitoring.Default, "output.elasticsearch.bulk_requests", func(_ monitoring.Mode, v monitoring.Visitor) {
		v.OnRegistryStart()
		defer v.OnRegistryFinished()
		stats := indexer.Stats()
		v.OnKey("available")
		v.OnInt(stats.AvailableBulkRequests)
		v.OnKey("completed")
		v.OnInt(stats.BulkRequests)
	})
	return indexer, indexer.Close, nil
}

func (r *serverRunner) newLibbeatFinalBatchProcessor(tracer *apm.Tracer) (model.BatchProcessor, func(context.Context) error, error) {
	monitors := pipeline.Monitors{
		Metrics:   libbeatMonitoringRegistry,
		Telemetry: monitoring.GetNamespace("state").GetRegistry(),
		Logger:    logp.L().Named("publisher"),
		Tracer:    tracer,
	}

	outputFactory := func(stats outputs.Observer) (string, outputs.Group, error) {
		indexSupporter := newSupporter(nil, r.info, r.rawConfig)
		group, err := outputs.Load(indexSupporter, r.info, stats, r.outputConfig.Name(), r.outputConfig.Config())
		return r.outputConfig.Name(), group, err
	}
	pipeline, err := pipeline.Load(r.info, monitors, pipeline.Config{}, nopProcessingSupporter{}, outputFactory)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create libbeat publisher pipeline: %w", err)
	}

	// When the publisher stops cleanly it will close its pipeline client,
	// calling the acker's Close method. We need to call Open for each new
	// publisher to ensure we wait for all clients and enqueued events to
	// be closed at shutdown time.
	acker := publish.NewWaitPublishedAcker()
	acker.Open() // TODO(axw) move to constructor
	pipelineConnector := pipetool.WithACKer(pipeline, acker)
	publisher, err := publish.NewPublisher(pipelineConnector, tracer)
	if err != nil {
		return nil, nil, err
	}
	stop := func(ctx context.Context) error {
		if err := publisher.Stop(ctx); err != nil {
			return err
		}
		return acker.Wait(ctx)
	}

	// Restore the original libbeat monitoring registry.
	monitoring.Default.Remove("libbeat")
	monitoring.Default.Add("libbeat", libbeatMonitoringRegistry, monitoring.Full)
	return publisher, stop, nil
}

func (r *serverRunner) initTracing() (*apm.Tracer, *tracerServer, error) {
	instrumentation, err := instrumentation.New(r.rawConfig, r.info.Beat, r.info.Version)
	if err != nil {
		return nil, nil, err
	}
	var tracerServer *tracerServer
	if listener := instrumentation.Listener(); listener != nil {
		tracerServer, err = newTracerServer(listener, r.logger)
		if err != nil {
			return nil, nil, err
		}
	}
	return instrumentation.Tracer(), tracerServer, nil
}

// runServerWithTracerServer wraps runServer such that it also runs
// tracerServer, stopping it and the tracer when the server shuts down.
func runServerWithTracerServer(runServer RunServerFunc, tracerServer *tracerServer, tracer *apm.Tracer) RunServerFunc {
	return func(ctx context.Context, args ServerParams) error {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return tracerServer.serve(ctx, args.BatchProcessor)
		})
		g.Go(func() error {
			return runServer(ctx, args)
		})
		return g.Wait()
	}
}

func newSourcemapFetcher(
	beatInfo beat.Info,
	cfg config.SourceMapping,
	fleetCfg *config.Fleet,
	kibanaClient kibana.Client,
	newElasticsearchClient func(*elasticsearch.Config) (elasticsearch.Client, error),
) (sourcemap.Fetcher, error) {
	// When running under Fleet we only fetch via Fleet Server.
	if fleetCfg != nil {
		var tlsConfig *tlscommon.TLSConfig
		var err error
		if fleetCfg.TLS.IsEnabled() {
			if tlsConfig, err = tlscommon.LoadTLSConfig(fleetCfg.TLS); err != nil {
				return nil, err
			}
		}

		timeout := 30 * time.Second
		dialer := transport.NetDialer(timeout)
		tlsDialer := transport.TLSDialer(dialer, tlsConfig, timeout)

		client := *http.DefaultClient
		client.Transport = apmhttp.WrapRoundTripper(&http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			Dial:            dialer.Dial,
			DialTLS:         tlsDialer.Dial,
			TLSClientConfig: tlsConfig.ToConfig(),
		})

		fleetServerURLs := make([]*url.URL, len(fleetCfg.Hosts))
		for i, host := range fleetCfg.Hosts {
			urlString, err := common.MakeURL(fleetCfg.Protocol, "", host, 8220)
			if err != nil {
				return nil, err
			}
			u, err := url.Parse(urlString)
			if err != nil {
				return nil, err
			}
			fleetServerURLs[i] = u
		}

		artifactRefs := make([]sourcemap.FleetArtifactReference, len(cfg.Metadata))
		for i, meta := range cfg.Metadata {
			artifactRefs[i] = sourcemap.FleetArtifactReference{
				ServiceName:        meta.ServiceName,
				ServiceVersion:     meta.ServiceVersion,
				BundleFilepath:     meta.BundleFilepath,
				FleetServerURLPath: meta.SourceMapURL,
			}
		}

		return sourcemap.NewFleetFetcher(
			&client,
			fleetCfg.AccessAPIKey,
			fleetServerURLs,
			artifactRefs,
		)
	}

	// For standalone, we query both Kibana and Elasticsearch for backwards compatibility.
	var chained sourcemap.ChainedFetcher
	if kibanaClient != nil {
		chained = append(chained, sourcemap.NewKibanaFetcher(kibanaClient))
	}
	esClient, err := newElasticsearchClient(cfg.ESConfig)
	if err != nil {
		return nil, err
	}
	index := strings.ReplaceAll(cfg.IndexPattern, "%{[observer.version]}", beatInfo.Version)
	esFetcher := sourcemap.NewElasticsearchFetcher(esClient, index)
	chained = append(chained, esFetcher)
	return chained, nil
}

// TODO: This is copying behavior from libbeat:
// https://github.com/elastic/beats/blob/b9ced47dba8bb55faa3b2b834fd6529d3c4d0919/libbeat/cmd/instance/beat.go#L927-L950
// Remove this when cluster_uuid no longer needs to be queried from ES.
func queryClusterUUID(ctx context.Context, esClient elasticsearch.Client) error {
	stateRegistry := monitoring.GetNamespace("state").GetRegistry()
	outputES := "outputs.elasticsearch"
	// Running under elastic-agent, the callback linked above is not
	// registered until later, meaning we need to check and instantiate the
	// registries if they don't exist.
	elasticsearchRegistry := stateRegistry.GetRegistry(outputES)
	if elasticsearchRegistry == nil {
		elasticsearchRegistry = stateRegistry.NewRegistry(outputES)
	}

	var (
		s  *monitoring.String
		ok bool
	)

	clusterUUID := "cluster_uuid"
	clusterUUIDRegVar := elasticsearchRegistry.Get(clusterUUID)
	if clusterUUIDRegVar != nil {
		s, ok = clusterUUIDRegVar.(*monitoring.String)
		if !ok {
			return fmt.Errorf("couldn't cast to String")
		}
	} else {
		s = monitoring.NewString(elasticsearchRegistry, clusterUUID)
	}

	var response struct {
		ClusterUUID string `json:"cluster_uuid"`
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		return err
	}
	resp, err := esClient.Perform(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("error querying cluster_uuid: status_code=%d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return err
	}

	s.Set(response.ClusterUUID)
	return nil
}

type nopProcessingSupporter struct{}

func (nopProcessingSupporter) Close() error {
	return nil
}

func (nopProcessingSupporter) Create(cfg beat.ProcessingConfig, _ bool) (beat.Processor, error) {
	return cfg.Processor, nil
}
