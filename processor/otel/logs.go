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

// Portions copied from OpenTelemetry Collector (contrib), from the
// elastic exporter.
//
// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"

	apmserverlogs "github.com/elastic/apm-server/log"
	"github.com/elastic/apm-server/model"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var jsonLogsMarshaler = otlp.NewJSONLogsMarshaler()

func (c *Consumer) ConsumeLogs(ctx context.Context, logs pdata.Logs) error {
	receiveTimestamp := time.Now()
	logger := logp.NewLogger(apmserverlogs.Otel)
	if logger.IsDebug() {
		data, err := jsonLogsMarshaler.MarshalLogs(logs)
		if err != nil {
			logger.Debug(err)
		} else {
			logger.Debug(string(data))
		}
	}
	batch := c.convertLogs(logs, receiveTimestamp)
	return c.Processor.ProcessBatch(ctx, batch)
}

func (c *Consumer) convertLogs(logs pdata.Logs, receiveTimestamp time.Time) *model.Batch {
	batch := model.Batch{}
	resourceLogs := logs.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		c.convertResourceLogs(resourceLogs.At(i), receiveTimestamp, &batch)
	}
	return &batch
}

func (c *Consumer) convertResourceLogs(resourceLogs pdata.ResourceLogs, receiveTimestamp time.Time, out *model.Batch) {
	var baseEvent model.APMEvent
	var timeDelta time.Duration
	resource := resourceLogs.Resource()
	translateResourceMetadata(resource, &baseEvent)
	if exportTimestamp, ok := exportTimestamp(resource); ok {
		timeDelta = receiveTimestamp.Sub(exportTimestamp)
	}
	instrumentationLibraryLogs := resourceLogs.InstrumentationLibraryLogs()
	for i := 0; i < instrumentationLibraryLogs.Len(); i++ {
		c.convertInstrumentationLibraryLogs(instrumentationLibraryLogs.At(i), baseEvent, timeDelta, out)
	}
}

func (c *Consumer) convertInstrumentationLibraryLogs(
	in pdata.InstrumentationLibraryLogs,
	baseEvent model.APMEvent,
	timeDelta time.Duration,
	out *model.Batch,
) {
	otelLogs := in.Logs()
	for i := 0; i < otelLogs.Len(); i++ {
		event := c.convertLogRecord(otelLogs.At(i), baseEvent, timeDelta)
		*out = append(*out, event)
	}
}

func (c *Consumer) convertLogRecord(
	record pdata.LogRecord,
	baseEvent model.APMEvent,
	timeDelta time.Duration,
) model.APMEvent {
	event := baseEvent
	event.Processor = model.LogProcessor
	event.Timestamp = record.Timestamp().AsTime().Add(timeDelta)
	if body := record.Body(); body.Type() != pdata.AttributeValueTypeNull {
		// TODO(axw) record map-type body as labels? Top-level fields?
		event.Message = body.AsString()
	}
	if traceID := record.TraceID(); !traceID.IsEmpty() {
		event.Trace.ID = traceID.HexString()
	}
	if spanID := record.SpanID(); !spanID.IsEmpty() {
		event.Span.ID = spanID.HexString()
	}
	if attrs := record.Attributes(); attrs.Len() > 0 {
		event.Labels = initEventLabels(event.Labels)
		attrs.Range(func(k string, v pdata.AttributeValue) bool {
			event.Labels[replaceDots(k)] = ifaceAttributeValue(v)
			return true
		})
	}
	// TODO(axw) log.level (record.SeverityText)
	// TODO(axw) event.severity (record.SeverityNumber)
	// TODO(axw) event.action (record.Name)
	return event
}
