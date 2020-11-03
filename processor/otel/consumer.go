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

package otel

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/utility"
)

const (
	AgentNameJaeger = "Jaeger"

	sourceFormatJaeger = "jaeger"
	keywordLength      = 1024
	dot                = "."
	underscore         = "_"

	outcomeSuccess = "success"
	outcomeFailure = "failure"
	outcomeUnknown = "unknown"
)

// Consumer transforms open-telemetry data to be compatible with elastic APM data
type Consumer struct {
	Reporter publish.Reporter
}

// TODO(axw) remove this.
func (c *Consumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	traces := internaldata.OCToTraceData(td)
	return c.ConsumeTraces(ctx, traces)
}

// ConsumeTraces consumes OpenTelemetry trace data,
// converting into Elastic APM events and reporting to the Elastic APM schema.
func (c *Consumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	batch := c.convert(td)
	return c.Reporter(ctx, publish.PendingReq{
		Transformables: batch.Transformables(),
		Trace:          true,
	})
}

func (c *Consumer) convert(td pdata.Traces) *model.Batch {
	batch := model.Batch{}
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		c.convertResourceSpans(resourceSpans.At(i), &batch)
	}
	return &batch
}

func (c *Consumer) convertResourceSpans(resourceSpans pdata.ResourceSpans, out *model.Batch) {
	if resourceSpans.IsNil() {
		return
	}
	var metadata model.Metadata
	translateResourceMetadata(resourceSpans.Resource(), &metadata)
	instrumentationLibrarySpans := resourceSpans.InstrumentationLibrarySpans()
	for i := 0; i < instrumentationLibrarySpans.Len(); i++ {
		c.convertInstrumentationLibrarySpans(instrumentationLibrarySpans.At(i), metadata, out)
	}
}

func (c *Consumer) convertInstrumentationLibrarySpans(in pdata.InstrumentationLibrarySpans, metadata model.Metadata, out *model.Batch) {
	if in.IsNil() {
		return
	}
	//logger := logp.NewLogger(logs.Otel)
	otelSpans := in.Spans()
	for i := 0; i < otelSpans.Len(); i++ {
		otelSpan := otelSpans.At(i)
		if otelSpan.IsNil() {
			continue
		}
		root := len(otelSpan.ParentSpanID().Bytes()) == 0

		const sourceFormat = sourceFormatJaeger // TODO(axw)

		// TODO(axw) special formatting for Jaeger span IDs, for log correlation.
		var parentID, spanID, traceID string
		if sourceFormat == sourceFormatJaeger {
			if !root {
				parentID = formatJaegerSpanID(otelSpan.ParentSpanID().Bytes())
			}
			traceID = formatJaegerTraceID(otelSpan.TraceID().Bytes())
			spanID = formatJaegerSpanID(otelSpan.SpanID().Bytes())
		} else {
			if !root {
				parentID = otelSpan.ParentSpanID().HexString()
			}
			traceID = otelSpan.TraceID().HexString()
			spanID = otelSpan.SpanID().HexString()
		}

		startTime := pdata.UnixNanoToTime(otelSpan.StartTime())
		endTime := pdata.UnixNanoToTime(otelSpan.EndTime())
		var duration float64
		if endTime.After(startTime) {
			duration = endTime.Sub(startTime).Seconds() * 1000
		}

		name := otelSpan.Name()
		if root || otelSpan.Kind() == pdata.SpanKindSERVER {
			transaction := model.Transaction{
				Metadata: metadata,
				ID:       spanID,
				ParentID: parentID,
				TraceID:  traceID,
				Duration: duration,
				Name:     name,
			}
			translateTransaction(otelSpan, &transaction)
			out.Transactions = append(out.Transactions, &transaction)
			//for _, err := range parseErrors(logger, td.SourceFormat, otelSpan) {
			//	addTransactionCtxToErr(transaction, err)
			//	batch.Errors = append(batch.Errors, err)
			//}
		} else {
			span := model.Span{
				Metadata:  metadata,
				ID:        spanID,
				ParentID:  parentID,
				TraceID:   traceID,
				Timestamp: startTime,
				Duration:  duration,
				Name:      name,
				Outcome:   outcomeUnknown,
			}
			translateSpan(otelSpan, &span)
			out.Spans = append(out.Spans, &span)
			//for _, err := range parseErrors(logger, td.SourceFormat, otelSpan) {
			//	addSpanCtxToErr(span, hostname, err)
			//	batch.Errors = append(batch.Errors, err)
			//}
		}
	}
}

/*
func parseMetadata(td pdata.Traces, md *model.Metadata) {
	if ident := td.Node.GetIdentifier(); ident != nil {
		md.Process.Pid = int(ident.Pid)
	}

	switch td.SourceFormat {
	case sourceFormatJaeger:
		// version is of format `Jaeger-<agentlanguage>-<version>`, e.g. `Jaeger-Go-2.20.0`
		nVersionParts := 3
		versionParts := strings.SplitN(td.Node.GetLibraryInfo().GetExporterVersion(), "-", nVersionParts)
		if md.Service.Language.Name == "" && len(versionParts) == nVersionParts {
			md.Service.Language.Name = versionParts[1]
		}
		if v := versionParts[len(versionParts)-1]; v != "" {
			md.Service.Agent.Version = v
		} else {
			md.Service.Agent.Version = "unknown"
		}
		agentName := AgentNameJaeger
		if md.Service.Language.Name != "" {
			agentName = truncate(agentName + "/" + md.Service.Language.Name)
		}
		md.Service.Agent.Name = agentName

		if attributes := td.Node.GetAttributes(); attributes != nil {
			if clientUUID, ok := attributes["client-uuid"]; ok {
				md.Service.Agent.EphemeralID = truncate(clientUUID)
				delete(td.Node.Attributes, "client-uuid")
			}
			if ip, ok := attributes["ip"]; ok {
				md.System.IP = utility.ParseIP(ip)
				delete(td.Node.Attributes, "ip")
			}
		}
	default:
		md.Service.Agent.Name = strings.Title(td.SourceFormat)
		md.Service.Agent.Version = "unknown"
	}

	if md.Service.Language.Name == "" {
		md.Service.Language.Name = "unknown"
	}

	md.Labels = make(common.MapStr)
	for key, val := range td.Node.GetAttributes() {
		md.Labels[replaceDots(key)] = truncate(val)
	}
	if t := td.Resource.GetType(); t != "" {
		md.Labels["resource"] = truncate(t)
	}
	for key, val := range td.Resource.GetLabels() {
		md.Labels[replaceDots(key)] = truncate(val)
	}
}
*/

func translateTransaction(span pdata.Span, out *model.Transaction) {
	const sourceFormat = sourceFormatJaeger // TODO(axw)

	labels := make(common.MapStr)
	var http model.Http
	var httpStatusCode int
	var message model.Message
	var component string
	var outcome, result string
	var hasFailed bool
	var isHTTP, isMessaging bool
	var samplerType, samplerParam pdata.AttributeValue
	span.Attributes().ForEach(func(kDots string, v pdata.AttributeValue) {
		if sourceFormat == sourceFormatJaeger {
			switch kDots {
			case "sampler.type":
				samplerType = v
				return
			case "sampler.param":
				samplerParam = v
				return
			}
		}

		k := replaceDots(kDots)
		switch v.Type() {
		case pdata.AttributeValueBOOL:
			utility.DeepUpdate(labels, k, v.BoolVal())
			if k == "error" {
				hasFailed = v.BoolVal()
			}
		case pdata.AttributeValueDOUBLE:
			utility.DeepUpdate(labels, k, v.DoubleVal())
		case pdata.AttributeValueINT:
			switch kDots {
			case "http.status_code":
				httpStatusCode = int(v.IntVal())
				isHTTP = true
			default:
				utility.DeepUpdate(labels, k, v.IntVal())
			}
		case pdata.AttributeValueSTRING:
			stringval := truncate(v.StringVal())
			switch kDots {
			case "span.kind": // filter out
			case "http.method":
				http.Request = &model.Req{Method: stringval}
				isHTTP = true
			case "http.url", "http.path":
				out.URL = model.ParseURL(stringval, out.Metadata.System.Hostname())
				isHTTP = true
			case "http.status_code":
				if intv, err := strconv.Atoi(stringval); err == nil {
					httpStatusCode = intv
				}
				isHTTP = true
			case "http.protocol":
				if strings.HasPrefix(stringval, "HTTP/") {
					version := strings.TrimPrefix(stringval, "HTTP/")
					http.Version = &version
				} else {
					utility.DeepUpdate(labels, k, stringval)
				}
				isHTTP = true
			case "message_bus.destination":
				message.QueueName = &stringval
				isMessaging = true
			case "type":
				out.Type = stringval
			case "service.version":
				out.Metadata.Service.Version = stringval
			case "component":
				component = stringval
				fallthrough
			default:
				utility.DeepUpdate(labels, k, stringval)
			}
		}
	})

	if out.Type == "" {
		if isHTTP {
			out.Type = "request"
		} else if isMessaging {
			out.Type = "messaging"
		} else if component != "" {
			out.Type = component
		} else {
			out.Type = "custom"
		}
	}

	if isHTTP {
		if httpStatusCode == 0 {
			// TODO(axw) check this
			status := span.Status()
			if !status.IsNil() {
				httpStatusCode = int(status.Code())
			}
		}
		if httpStatusCode > 0 {
			http.Response = &model.Resp{MinimalResp: model.MinimalResp{StatusCode: &httpStatusCode}}
			result = statusCodeResult(httpStatusCode)
			outcome = serverStatusCodeOutcome(httpStatusCode)
		}
		out.HTTP = &http
	} else if isMessaging {
		out.Message = &message
	}

	if result == "" {
		if hasFailed {
			result = "Error"
			outcome = outcomeFailure
		} else {
			result = "Success"
			outcome = outcomeSuccess
		}
	}
	out.Result = result
	out.Outcome = outcome

	if samplerType != (pdata.AttributeValue{}) && samplerParam != (pdata.AttributeValue{}) {
		// The client has reported its sampling rate, so
		// we can use it to extrapolate span metrics.
		parseSamplerAttributes(samplerType, samplerParam, &out.RepresentativeCount, labels)
	}

	if len(labels) == 0 {
		return
	}
	l := model.Labels(labels)
	out.Labels = &l
}

func translateSpan(span pdata.Span, out *model.Span) {
	labels := make(common.MapStr)

	const sourceFormat = sourceFormatJaeger // TODO(axw)

	var http model.HTTP
	var message model.Message
	var db model.DB
	var destination model.Destination
	var destinationService model.DestinationService
	var isDBSpan, isHTTPSpan, isMessagingSpan bool
	var component string
	var samplerType, samplerParam pdata.AttributeValue
	span.Attributes().ForEach(func(kDots string, v pdata.AttributeValue) {
		if sourceFormat == sourceFormatJaeger {
			switch kDots {
			case "sampler.type":
				samplerType = v
				return
			case "sampler.param":
				samplerParam = v
				return
			}
		}

		k := replaceDots(kDots)
		switch v.Type() {
		case pdata.AttributeValueBOOL:
			utility.DeepUpdate(labels, k, v.BoolVal())
		case pdata.AttributeValueDOUBLE:
			utility.DeepUpdate(labels, k, v.DoubleVal())
		case pdata.AttributeValueINT:
			switch kDots {
			case "http.status_code":
				code := int(v.IntVal())
				http.StatusCode = &code
				isHTTPSpan = true
			case "peer.port":
				port := int(v.IntVal())
				destination.Port = &port
			default:
				utility.DeepUpdate(labels, k, v.IntVal())
			}
		case pdata.AttributeValueSTRING:
			stringval := truncate(v.StringVal())
			switch kDots {
			case "span.kind": // filter out
			case "http.url":
				http.URL = &stringval
				isHTTPSpan = true
			case "http.method":
				http.Method = &stringval
				isHTTPSpan = true
			case "sql.query":
				db.Statement = &stringval
				if db.Type == nil {
					dbType := "sql"
					db.Type = &dbType
				}
				isDBSpan = true
			case "db.statement":
				db.Statement = &stringval
				isDBSpan = true
			case "db.instance":
				db.Instance = &stringval
				isDBSpan = true
			case "db.type":
				db.Type = &stringval
				isDBSpan = true
			case "db.user":
				db.UserName = &stringval
				isDBSpan = true
			case "peer.address":
				destinationService.Resource = &stringval
				if !strings.ContainsRune(stringval, ':') || net.ParseIP(stringval) != nil {
					// peer.address is not necessarily a hostname
					// or IP address; it could be something like
					// a JDBC connection string or ip:port. Ignore
					// values containing colons, except for IPv6.
					destination.Address = &stringval
				}
			case "peer.hostname", "peer.ipv4", "peer.ipv6":
				destination.Address = &stringval
			case "peer.service":
				destinationService.Name = &stringval
				if destinationService.Resource == nil {
					// Prefer using peer.address for resource.
					destinationService.Resource = &stringval
				}
			case "message_bus.destination":
				message.QueueName = &stringval
				isMessagingSpan = true
			case "component":
				component = stringval
				fallthrough
			default:
				utility.DeepUpdate(labels, k, stringval)
			}
		}
	})

	if http.URL != nil {
		if fullURL, err := url.Parse(*http.URL); err == nil {
			url := url.URL{Scheme: fullURL.Scheme, Host: fullURL.Host}
			hostname := truncate(url.Hostname())
			var port int
			portString := url.Port()
			if portString != "" {
				port, _ = strconv.Atoi(portString)
			} else {
				port = schemeDefaultPort(url.Scheme)
			}

			// Set destination.{address,port} from the HTTP URL,
			// replacing peer.* based values to ensure consistency.
			destination = model.Destination{Address: &hostname}
			if port > 0 {
				destination.Port = &port
			}

			// Set destination.service.* from the HTTP URL,
			// unless peer.service was specified.
			if destinationService.Name == nil {
				resource := url.Host
				if port > 0 && port == schemeDefaultPort(url.Scheme) {
					hasDefaultPort := portString != ""
					if hasDefaultPort {
						// Remove the default port from destination.service.name.
						url.Host = hostname
					} else {
						// Add the default port to destination.service.resource.
						resource = fmt.Sprintf("%s:%d", resource, port)
					}
				}
				name := url.String()
				destinationService.Name = &name
				destinationService.Resource = &resource
			}
		}
	}

	if destination != (model.Destination{}) {
		out.Destination = &destination
	}

	switch {
	case isHTTPSpan:
		if http.StatusCode == nil {
			// TODO(axw) check this
			status := span.Status()
			if !status.IsNil() {
				if code := int(status.Code()); code != 0 {
					http.StatusCode = &code
				}
			}
		}
		if http.StatusCode != nil {
			out.Outcome = clientStatusCodeOutcome(*http.StatusCode)
		}
		out.Type = "external"
		subtype := "http"
		out.Subtype = &subtype
		out.HTTP = &http
	case isDBSpan:
		out.Type = "db"
		if db.Type != nil && *db.Type != "" {
			out.Subtype = db.Type
		}
		out.DB = &db
	case isMessagingSpan:
		out.Type = "messaging"
		out.Message = &message
	default:
		out.Type = "custom"
		if component != "" {
			out.Subtype = &component
		}
	}

	if destinationService != (model.DestinationService{}) {
		if destinationService.Type == nil {
			// Copy span type to destination.service.type.
			destinationService.Type = &out.Type
		}
		out.DestinationService = &destinationService
	}

	if samplerType != (pdata.AttributeValue{}) && samplerParam != (pdata.AttributeValue{}) {
		// The client has reported its sampling rate, so
		// we can use it to extrapolate transaction metrics.
		parseSamplerAttributes(samplerType, samplerParam, &out.RepresentativeCount, labels)
	}

	if len(labels) == 0 {
		return
	}
	out.Labels = labels
}

func parseSamplerAttributes(samplerType, samplerParam pdata.AttributeValue, representativeCount *float64, labels common.MapStr) {
	switch samplerType.StringVal() {
	case "probabilistic":
		probability := samplerParam.DoubleVal()
		if probability > 0 && probability <= 1 {
			*representativeCount = 1 / probability
		}
	default:
		utility.DeepUpdate(labels, "sampler_type", samplerType.StringVal())
		switch samplerParam.Type() {
		case pdata.AttributeValueBOOL:
			utility.DeepUpdate(labels, "sampler_param", samplerParam.BoolVal())
		case pdata.AttributeValueDOUBLE:
			utility.DeepUpdate(labels, "sampler_param", samplerParam.DoubleVal())
		}
	}
}

func parseErrors(logger *logp.Logger, source string, otelSpan *tracepb.Span) []*model.Error {
	var errors []*model.Error
	for _, log := range otelSpan.GetTimeEvents().GetTimeEvent() {
		var isError, hasMinimalInfo bool
		var err model.Error
		var logMessage, exMessage, exType string
		for k, v := range log.GetAnnotation().GetAttributes().GetAttributeMap() {
			if source == sourceFormatJaeger {
				switch v := v.Value.(type) {
				case *tracepb.AttributeValue_StringValue:
					vStr := v.StringValue.Value
					switch k {
					case "error", "error.object":
						exMessage = vStr
						hasMinimalInfo = true
						isError = true
					case "event":
						if vStr == "error" { // according to opentracing spec
							isError = true
						} else if logMessage == "" {
							// jaeger seems to send the message in the 'event' field
							// in case 'event' and 'message' are sent, 'message' is used
							logMessage = vStr
							hasMinimalInfo = true
						}
					case "message":
						logMessage = vStr
						hasMinimalInfo = true
					case "error.kind":
						exType = vStr
						hasMinimalInfo = true
						isError = true
					case "level":
						isError = vStr == "error"
					}
				}
			}
		}
		if !isError {
			continue
		}
		if !hasMinimalInfo {
			if logger.IsDebug() {
				logger.Debugf("Cannot convert %s event into elastic apm error: %v", source, log)
			}
			continue
		}

		if logMessage != "" {
			err.Log = &model.Log{Message: logMessage}
		}
		if exMessage != "" || exType != "" {
			err.Exception = &model.Exception{}
			if exMessage != "" {
				err.Exception.Message = &exMessage
			}
			if exType != "" {
				err.Exception.Type = &exType
			}
		}
		err.Timestamp = parseTimestamp(log.GetTime())
		errors = append(errors, &err)
	}
	return errors
}

func addTransactionCtxToErr(transaction model.Transaction, err *model.Error) {
	err.Metadata = transaction.Metadata
	err.TransactionID = transaction.ID
	err.TraceID = transaction.TraceID
	err.ParentID = transaction.ID
	err.HTTP = transaction.HTTP
	err.URL = transaction.URL
	err.TransactionType = &transaction.Type
}

func addSpanCtxToErr(span model.Span, hostname string, err *model.Error) {
	err.Metadata = span.Metadata
	err.TransactionID = span.TransactionID
	err.TraceID = span.TraceID
	err.ParentID = span.ID
	if span.HTTP != nil {
		err.HTTP = &model.Http{}
		if span.HTTP.StatusCode != nil {
			err.HTTP.Response = &model.Resp{MinimalResp: model.MinimalResp{StatusCode: span.HTTP.StatusCode}}
		}
		if span.HTTP.Method != nil {
			err.HTTP.Request = &model.Req{Method: *span.HTTP.Method}
		}
		if span.HTTP.URL != nil {
			err.URL = model.ParseURL(*span.HTTP.URL, hostname)
		}
	}
}

func replaceDots(s string) string {
	return strings.ReplaceAll(s, dot, underscore)
}

func parseTimestamp(timestampT *timestamp.Timestamp) time.Time {
	if timestampT == nil {
		return time.Time{}
	}
	return time.Unix(timestampT.Seconds, int64(timestampT.Nanos)).UTC()
}

var languageName = map[commonpb.LibraryInfo_Language]string{
	1:  "C++",
	2:  "CSharp",
	3:  "Erlang",
	4:  "Go",
	5:  "Java",
	6:  "Node",
	7:  "PHP",
	8:  "Python",
	9:  "Ruby",
	10: "JavaScript",
}

// copied from elastic go-apm agent

var standardStatusCodeResults = [...]string{
	"HTTP 1xx",
	"HTTP 2xx",
	"HTTP 3xx",
	"HTTP 4xx",
	"HTTP 5xx",
}

// statusCodeResult returns the transaction result value to use for the given status code.
func statusCodeResult(statusCode int) string {
	switch i := statusCode / 100; i {
	case 1, 2, 3, 4, 5:
		return standardStatusCodeResults[i-1]
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

// serverStatusCodeOutcome returns the transaction outcome value to use for the given status code.
func serverStatusCodeOutcome(statusCode int) string {
	if statusCode >= 500 {
		return outcomeFailure
	}
	return outcomeSuccess
}

// clientStatusCodeOutcome returns the span outcome value to use for the given status code.
func clientStatusCodeOutcome(statusCode int) string {
	if statusCode >= 400 {
		return outcomeFailure
	}
	return outcomeSuccess
}

// truncate returns s truncated at n runes, and the number of runes in the resulting string (<= n).
func truncate(s string) string {
	var j int
	for i := range s {
		if j == keywordLength {
			return s[:i]
		}
		j++
	}
	return s
}

// formatJaegerTraceID returns the traceID as string in Jaeger format (hexadecimal without leading zeros)
func formatJaegerTraceID(traceID []byte) string {
	jaegerTraceIDHigh, jaegerTraceIDLow, err := tracetranslator.BytesToUInt64TraceID(traceID)
	if err != nil {
		return fmt.Sprintf("%x", traceID)
	}

	if jaegerTraceIDHigh == 0 {
		return fmt.Sprintf("%x", jaegerTraceIDLow)
	}

	return fmt.Sprintf("%x%016x", jaegerTraceIDHigh, jaegerTraceIDLow)
}

// formatJaegerSpanID returns the spanID as string in Jaeger format (hexadecimal without leading zeros)
func formatJaegerSpanID(spanID []byte) string {
	jaegerSpanID, err := tracetranslator.BytesToUInt64SpanID(spanID)
	if err != nil {
		return fmt.Sprintf("%x", spanID)
	}

	return fmt.Sprintf("%x", jaegerSpanID)
}

func schemeDefaultPort(scheme string) int {
	switch scheme {
	case "http":
		return 80
	case "https":
		return 443
	}
	return 0
}
