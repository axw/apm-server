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

package stream

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"go.elastic.co/apm"

	"github.com/elastic/apm-server/beater/config"
	"github.com/elastic/apm-server/decoder"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/modeldecoder"
	"github.com/elastic/apm-server/model/modeldecoder/field"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/utility"
	"github.com/elastic/apm-server/validation"
)

var (
	ErrUnrecognizedObject = errors.New("did not recognize object type")
)

const (
	batchSize = 10
)

type decodeMetadataFunc func(interface{}, bool, *model.Metadata) error

// functions with the decodeEventFunc signature decode their input argument into their batch argument (output)
type decodeEventFunc func(modeldecoder.Input, *model.Batch) error

type Processor struct {
	Mconfig          modeldecoder.Config
	MaxEventSize     int
	streamReaderPool sync.Pool
	decodeMetadata   decodeMetadataFunc
	models           map[string]decodeEventFunc
}

func BackendProcessor(cfg *config.Config) *Processor {
	return &Processor{
		Mconfig:        modeldecoder.Config{Experimental: cfg.Mode == config.ModeExperimental},
		MaxEventSize:   cfg.MaxEventSize,
		decodeMetadata: modeldecoder.DecodeMetadata,
		models: map[string]decodeEventFunc{
			"transaction": modeldecoder.DecodeTransaction,
			"span":        modeldecoder.DecodeSpan,
			"metricset":   modeldecoder.DecodeMetricset,
			"error":       modeldecoder.DecodeError,
		},
	}
}

func RUMV2Processor(cfg *config.Config) *Processor {
	return &Processor{
		Mconfig:        modeldecoder.Config{Experimental: cfg.Mode == config.ModeExperimental},
		MaxEventSize:   cfg.MaxEventSize,
		decodeMetadata: modeldecoder.DecodeMetadata,
		models: map[string]decodeEventFunc{
			"transaction": modeldecoder.DecodeRUMV2Transaction,
			"span":        modeldecoder.DecodeRUMV2Span,
			"metricset":   modeldecoder.DecodeRUMV2Metricset,
			"error":       modeldecoder.DecodeRUMV2Error,
		},
	}
}

func RUMV3Processor(cfg *config.Config) *Processor {
	return &Processor{
		Mconfig:        modeldecoder.Config{Experimental: cfg.Mode == config.ModeExperimental, HasShortFieldNames: true},
		MaxEventSize:   cfg.MaxEventSize,
		decodeMetadata: modeldecoder.DecodeRUMV3Metadata,
		models: map[string]decodeEventFunc{
			"x":  modeldecoder.DecodeRUMV3Transaction,
			"e":  modeldecoder.DecodeRUMV3Error,
			"me": modeldecoder.DecodeRUMV3Metricset,
		},
	}
}

func (p *Processor) readMetadata(metadata *model.Metadata, reader *streamReader) error {
	var rawModel map[string]interface{}
	line, err := reader.Read(&rawModel)
	if err != nil {
		if err == io.EOF {
			return &Error{
				Type:     InvalidInputErrType,
				Message:  "EOF while reading metadata",
				Document: line,
			}
		}
		return err
	}

	fieldName := field.Mapper(p.Mconfig.HasShortFieldNames)
	rawMetadata, ok := rawModel[fieldName("metadata")].(map[string]interface{})
	if !ok {
		return &Error{
			Type:     InvalidInputErrType,
			Message:  ErrUnrecognizedObject.Error(),
			Document: line,
		}
	}

	if err := p.decodeMetadata(rawMetadata, p.Mconfig.HasShortFieldNames, metadata); err != nil {
		var ve *validation.Error
		if errors.As(err, &ve) {
			return &Error{
				Type:     InvalidInputErrType,
				Message:  err.Error(),
				Document: line,
			}
		}
		return err
	}
	return nil
}

// HandleRawModel validates and decodes a single json object into its struct form
func (p *Processor) HandleRawModel(rawModel map[string]interface{}, batch *model.Batch, requestTime time.Time, streamMetadata model.Metadata) error {
	for key, decodeEvent := range p.models {
		entry, ok := rawModel[key]
		if !ok {
			continue
		}
		err := decodeEvent(modeldecoder.Input{
			Raw:         entry,
			RequestTime: requestTime,
			Metadata:    streamMetadata,
			Config:      p.Mconfig,
		}, batch)
		if err != nil {
			return err
		}
		return nil
	}
	return ErrUnrecognizedObject
}

func (p *Processor) readBatches(
	ctx context.Context,
	ipRateLimiter *rate.Limiter,
	requestTime time.Time,
	streamMetadata *model.Metadata,
	batchSize int,
	reader *streamReader,
	response *Result,
	batches chan<- *model.Batch,
) {
	type decodedLine struct {
		m    map[string]interface{}
		line string
		err  error
	}
	mapPool := sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{})
		},
	}

	decodedLines := make(chan decodedLine, 1000)
	go func() {
		defer close(decodedLines)
		var n int
		for !reader.IsEOF() {
			if ipRateLimiter != nil {
				if n++; n == batchSize {
					n = 0
					// use provided rate limiter to throttle batch read
					ctxT, cancel := context.WithTimeout(ctx, time.Second)
					err := ipRateLimiter.WaitN(ctxT, batchSize)
					cancel()
					if err != nil {
						response.Add(&Error{
							Type:    RateLimitErrType,
							Message: "rate limit exceeded",
						})
						return
					}
				}
			}
			var decoded decodedLine
			decoded.m = mapPool.Get().(map[string]interface{})
			decoded.line, decoded.err = reader.Read(&decoded.m)
			if decoded.err == io.EOF {
				decoded.err = nil
			}
			select {
			case <-ctx.Done():
				return
			case decodedLines <- decoded:
			}
		}
	}()

	var out chan<- *model.Batch
	batch := getBatch()
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(time.Second)
			out = batches
		case out <- batch:
			batch = getBatch()
			out = nil
		case decoded, ok := <-decodedLines:
			if !ok {
				if batch.Len() > 0 {
					select {
					case <-ctx.Done():
					case batches <- batch:
					}
				}
				return
			}
			err := decoded.err
			if err == nil {
				if len(decoded.m) > 0 {
					err = p.HandleRawModel(decoded.m, batch, requestTime, *streamMetadata)
					if err != nil {
						err = &Error{
							Type:     InvalidInputErrType,
							Message:  err.Error(),
							Document: decoded.line,
						}
					}
					for k := range decoded.m {
						delete(decoded.m, k)
					}
				}
			}
			if err != nil && err != io.EOF {
				if err, ok := err.(*Error); ok {
					switch err.Type {
					case InvalidInputErrType, InputTooLargeErrType:
						response.LimitedAdd(err)
						continue
					}
				}
				// return early, we assume we can only recover from a input error types
				response.Add(err)
				return
			}
			mapPool.Put(decoded.m)
			if batch.Len() == batchSize {
				out = batches
			}
		}
	}
}

// HandleStream processes a stream of events
func (p *Processor) HandleStream(ctx context.Context, ipRateLimiter *rate.Limiter, meta *model.Metadata, reader io.Reader, report publish.Reporter) *Result {
	res := &Result{}

	sr := p.getStreamReader(reader)
	defer sr.release()

	// first item is the metadata object
	err := p.readMetadata(meta, sr)
	if err != nil {
		// no point in continuing if we couldn't read the metadata
		res.Add(err)
		return res
	}

	requestTime := utility.RequestTime(ctx)

	sp, ctx := apm.StartSpan(ctx, "Stream", "Reporter")
	defer sp.End()

	batches := make(chan *model.Batch)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer close(batches)
		p.readBatches(ctx, ipRateLimiter, requestTime, meta, batchSize, sr, res, batches)
	}()
	for batch := range batches {
		// NOTE(axw) `report` takes ownership of transformables, which
		// means we cannot reuse the slice memory. We should investigate
		// alternative interfaces between the processor and publisher
		// which would enable better memory reuse.
		if err := report(ctx, publish.PendingReq{
			Transformables: batch.Transformables(),
			Trace:          !sp.Dropped(),
		}); err != nil {
			switch err {
			case publish.ErrChannelClosed:
				res.Add(&Error{
					Type:    ShuttingDownErrType,
					Message: "server is shutting down",
				})
			case publish.ErrFull:
				res.Add(&Error{
					Type:    QueueFullErrType,
					Message: err.Error(),
				})
			default:
				res.Add(err)
			}
			return res
		}
		res.AddAccepted(batch.Len())
		releaseBatch(batch)
	}

	return res
}

// getStreamReader returns a streamReader that reads ND-JSON lines from r.
func (p *Processor) getStreamReader(r io.Reader) *streamReader {
	if sr, ok := p.streamReaderPool.Get().(*streamReader); ok {
		sr.d.Reset(r)
		return sr
	}
	return &streamReader{
		processor: p,
		d:         decoder.NewNDJSONStreamDecoder(r, p.MaxEventSize),
	}
}

// streamReader wraps NDJSONStreamReader, converting errors to stream errors.
type streamReader struct {
	processor *Processor
	d         *decoder.NDJSONStreamDecoder
}

// release releases the streamReader, adding it to its Processor's sync.Pool.
// The streamReader must not be used after release returns.
func (sr *streamReader) release() {
	sr.d.Reset(nil)
	sr.processor.streamReaderPool.Put(sr)
}

func (sr *streamReader) IsEOF() bool {
	return sr.d.IsEOF()
}

func (sr *streamReader) Read(v *map[string]interface{}) (string, error) {
	// TODO(axw) decode into a reused map, clearing out the
	// map between reads. We would require that decoders copy
	// any contents of rawModel that they wish to retain after
	// the call, in order to safely reuse the map.
	err := sr.d.Decode(v)
	line := string(sr.d.LatestLine())
	if err != nil {
		if _, ok := err.(decoder.JSONDecodeError); ok {
			return line, &Error{
				Type:     InvalidInputErrType,
				Message:  err.Error(),
				Document: line,
			}
		}
		if err == decoder.ErrLineTooLong {
			return line, &Error{
				Type:     InputTooLargeErrType,
				Message:  "event exceeded the permitted size.",
				Document: line,
			}
		}
	}
	return line, err
}

var batchPool = sync.Pool{
	New: func() interface{} {
		return new(model.Batch)
	},
}

func getBatch() *model.Batch {
	return batchPool.Get().(*model.Batch)
}

func releaseBatch(b *model.Batch) {
	b.Reset()
	batchPool.Put(b)
}
