package beatcmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"golang.org/x/sync/errgroup"
)

// NewRunnerFunc is a function type that constructs a new Runner with the given
// parameters.
type NewRunnerFunc func(RunnerParams) (Runner, error)

type RunnerParams struct {
	// Config holds the full, raw, configuration, including apm-server.*
	// and output.* attributes.
	Config *config.C

	// Info holds information about the APM Server ("beat", for historical
	// reasons) process.
	Info beat.Info

	// Logger holds a logger to use for logging throughout the APM Server.
	Logger *logp.Logger
}

type Runner interface {
	Run(context.Context) error
}

// NewCreator returns a new Reloader which creates Runners using the provided
// beat.Info and NewRunnerFunc.
func NewReloader(info beat.Info, newRunner NewRunnerFunc) (*Reloader, error) {
	r := &Reloader{
		info:      info,
		logger:    logp.NewLogger(""),
		newRunner: newRunner,
		stopped:   make(chan struct{}),
	}
	if err := reload.Register.RegisterList("inputs", reloadableListFunc(r.reloadInputs)); err != nil {
		return nil, fmt.Errorf("failed to register inputs reloader: %w", err)
	}
	if err := reload.Register.Register("output", reload.ReloadableFunc(r.reloadOutput)); err != nil {
		return nil, fmt.Errorf("failed to register output reloader: %w", err)
	}
	return r, nil
}

type Reloader struct {
	info      beat.Info
	logger    *logp.Logger
	newRunner func(RunnerParams) (Runner, error)

	runner     Runner
	stopRunner func() error

	mu           sync.Mutex
	inputConfig  *config.C
	outputConfig config.Namespace
	stopped      chan struct{}
}

// Run runs the Reloader, blocking until ctx is cancelled or a fatal error occurs.
//
// Run must be called once and only once.
func (r *Reloader) Run(ctx context.Context) error {
	defer close(r.stopped)
	<-ctx.Done()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runner == nil {
		return nil
	}
	return r.stopRunner()
}

// reloadInput (re)loads input configuration.
//
// Note: reloadInputs may be called before the Reloader is running.
func (r *Reloader) reloadInputs(configs []*reload.ConfigWithMeta) error {
	if n := len(configs); n != 1 {
		return fmt.Errorf("only 1 input supported, got %d", n)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg := configs[0].Config
	if err := r.reload(cfg, r.outputConfig); err != nil {
		return fmt.Errorf("failed to load input config: %w", err)
	}
	r.inputConfig = cfg
	return nil
}

// reloadOutput (re)loads output configuration.
//
// Note: reloadOutput may be called before the Reloader is running.
func (r *Reloader) reloadOutput(cfg *reload.ConfigWithMeta) error {
	var outputConfig config.Namespace
	if err := cfg.Config.Unpack(&outputConfig); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.reload(r.inputConfig, outputConfig); err != nil {
		return fmt.Errorf("failed to load output config: %w", err)
	}
	r.outputConfig = outputConfig
	return nil
}

func (r *Reloader) reload(inputConfig *config.C, outputConfig config.Namespace) error {
	if inputConfig == nil || !outputConfig.IsSet() {
		// Wait until both input and output have been received.
		return nil
	}
	select {
	case <-r.stopped:
		// The process is shutting down: ignore reloads.
		return nil
	default:
	}
	mergedConfig, err := config.MergeConfigs(inputConfig, outputConfig.Config())
	if err != nil {
		return err
	}

	// Create a new runner. We separate creation from starting to
	// allow the runner to perform initialisations that must run
	// synchronously.
	newRunner, err := r.newRunner(RunnerParams{
		Config: mergedConfig,
		Info:   r.info,
		Logger: r.logger,
	})
	if err != nil {
		return err
	}

	// Start the new runner.
	var g errgroup.Group
	ctx, cancel := context.WithCancel(context.Background())
	g.Go(func() error { return newRunner.Run(ctx) })
	stopRunner := func() error {
		cancel()
		return g.Wait()
	}

	// Stop any existing runner.
	if r.runner != nil {
		if err := r.stopRunner(); err != nil {
			r.logger.Named("beater").With(logp.Error(err)).Error("on reload, old runner stopped with an error")
		}
	}
	r.runner = newRunner
	r.stopRunner = stopRunner

	return nil
}

type reloadableListFunc func(config []*reload.ConfigWithMeta) error

func (f reloadableListFunc) Reload(configs []*reload.ConfigWithMeta) error {
	return f(configs)
}
