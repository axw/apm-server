// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"

	"github.com/elastic/apm-server/beater"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/spanmetrics"
	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/txmetrics"
	"github.com/elastic/apm-server/x-pack/apm-server/cmd"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling"
)

type namedProcessor struct {
	name string
	processor
}

type processor interface {
	ProcessTransformables(context.Context, []transform.Transformable) ([]transform.Transformable, error)
	Run() error
	Stop(context.Context) error
}

// newProcessors returns a list of processors which will process
// events in sequential order, prior to the events being published.
func newProcessors(args beater.ServerParams) ([]namedProcessor, error) {
	var processors []namedProcessor
	if args.Config.Aggregation.Transactions.Enabled {
		const name = "transaction metrics aggregation"
		args.Logger.Infof("creating %s with config: %+v", name, args.Config.Aggregation.Transactions)
		agg, err := txmetrics.NewAggregator(txmetrics.AggregatorConfig{
			Report:                         args.Reporter,
			MaxTransactionGroups:           args.Config.Aggregation.Transactions.MaxTransactionGroups,
			MetricsInterval:                args.Config.Aggregation.Transactions.Interval,
			HDRHistogramSignificantFigures: args.Config.Aggregation.Transactions.HDRHistogramSignificantFigures,
			RUMUserAgentLRUSize:            args.Config.Aggregation.Transactions.RUMUserAgentLRUSize,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "error creating %s", name)
		}
		processors = append(processors, namedProcessor{name: name, processor: agg})
	}
	if args.Config.Aggregation.ServiceDestinations.Enabled {
		const name = "service destinations aggregation"
		args.Logger.Infof("creating %s with config: %+v", name, args.Config.Aggregation.ServiceDestinations)
		spanAggregator, err := spanmetrics.NewAggregator(spanmetrics.AggregatorConfig{
			Report:    args.Reporter,
			Interval:  args.Config.Aggregation.ServiceDestinations.Interval,
			MaxGroups: args.Config.Aggregation.ServiceDestinations.MaxGroups,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "error creating %s", name)
		}
		processors = append(processors, namedProcessor{name: name, processor: spanAggregator})
	}
	if args.Config.Sampling.Tail.Enabled {
		// runServerWithTailSampler wraps a RunServerFunc with a tail
		// sampler, if enabled. Transactions and spans will run through
		// the tail sampler, and will be buffered in local storage until
		// a tail-sampling decision is made (then published/dropped based
		// on the decision), or published/dropped immediately if the
		// decision has already been made.
		const name = "tail sampler"
		tailSamplingConfig := args.Config.Sampling.Tail
		storageDir := paths.Resolve(paths.Data, tailSamplingConfig.StorageDir)
		sampler, err := sampling.NewProcessor(sampling.Config{
			Reporter:   args.Reporter,
			StorageDir: storageDir,

			// TODO(axw) should these be configurable? Depends on whether
			// or not we want to keep the ability define a default
			// tail-sampling rate for "other" trace groups, or require
			// users to specify a tail-sampling rate for the high
			// priority trace groups.
			MaxTraceGroups:    1000,
			DefaultSampleRate: tailSamplingConfig.DefaultSampleRate,

			TTL:                   tailSamplingConfig.TTL,
			FlushInterval:         tailSamplingConfig.Interval,
			IngestRateCoefficient: tailSamplingConfig.IngestRateCoefficient,
			StorageGCInterval:     tailSamplingConfig.StorageGCInterval,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "error creating %s", name)
		}
		processors = append(processors, namedProcessor{name: name, processor: sampler})
	}
	return processors, nil
}

// runServerWithProcessors runs the APM Server and the given list of processors.
//
// newProcessors returns a list of processors which will process events in
// sequential order, prior to the events being published.
func runServerWithProcessors(ctx context.Context, runServer beater.RunServerFunc, args beater.ServerParams, processors ...namedProcessor) error {
	if len(processors) == 0 {
		return runServer(ctx, args)
	}

	origReport := args.Reporter
	args.Reporter = func(ctx context.Context, req publish.PendingReq) error {
		var err error
		for _, p := range processors {
			req.Transformables, err = p.ProcessTransformables(ctx, req.Transformables)
			if err != nil {
				return err
			}
		}
		return origReport(ctx, req)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, p := range processors {
		p := p // copy for closure
		g.Go(func() error {
			if err := p.Run(); err != nil {
				args.Logger.Errorf("%s aborted", p.name, logp.Error(err))
				return err
			}
			args.Logger.Infof("%s stopped", p.name)
			return nil
		})
		g.Go(func() error {
			<-ctx.Done()
			stopctx := context.Background()
			if args.Config.ShutdownTimeout > 0 {
				// On shutdown wait for the aggregator to stop
				// in order to flush any accumulated metrics.
				var cancel context.CancelFunc
				stopctx, cancel = context.WithTimeout(stopctx, args.Config.ShutdownTimeout)
				defer cancel()
			}
			return p.Stop(stopctx)
		})
	}
	g.Go(func() error {
		return runServer(ctx, args)
	})
	return g.Wait()
}

var rootCmd = cmd.NewXPackRootCommand(beater.NewCreator(beater.CreatorParams{
	WrapRunServer: func(runServer beater.RunServerFunc) beater.RunServerFunc {
		return func(ctx context.Context, args beater.ServerParams) error {
			processors, err := newProcessors(args)
			if err != nil {
				return err
			}
			return runServerWithProcessors(ctx, runServer, args, processors...)
		}
	},
}))

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
