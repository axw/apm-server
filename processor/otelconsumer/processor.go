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

package otelconsumer

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
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"

	"github.com/elastic/apm-server/model"
	"github.com/elastic/apm-server/model/metadata"
	model_span "github.com/elastic/apm-server/model/span"
	"github.com/elastic/apm-server/model/transaction"
	"github.com/elastic/apm-server/publish"
	"github.com/elastic/apm-server/transform"
	"github.com/elastic/apm-server/utility"
	"github.com/elastic/beats/libbeat/common"
)

// Consumer implements OpenTelemetry Collector's consumer interfaces.
type Consumer struct {
	TransformConfig transform.Config
	ModelConfig     model.Config
	Reporter        publish.Reporter
}

// TODO(axw) ConsumeMetricsData

// ConsumeTraceData consumes OpenTelemetry trace data, transforming and reporting in the Elastic APM schema.
func (c *Consumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	requestTime := time.Now()
	metadata, transformables := c.transform(td)
	transformContext := &transform.Context{
		RequestTime: requestTime,
		Config:      c.TransformConfig,
		Metadata:    metadata,
	}
	return c.Reporter(ctx, publish.PendingReq{
		Transformables: transformables,
		Tcontext:       transformContext,
		Trace:          true,
	})
}

func (c *Consumer) transform(td consumerdata.TraceData) (metadata.Metadata, []transform.Transformable) {
	md := metadata.Metadata{
		Service: &metadata.Service{Name: &td.Node.ServiceInfo.Name},
		// User? Labels?
	}
	var hostport string
	if ident := td.Node.GetIdentifier(); ident != nil {
		md.Process = &metadata.Process{Pid: int(ident.GetPid())}
		if ident.HostName != "" {
			hostport = ident.HostName
			md.System = &metadata.System{DetectedHostname: &ident.HostName}
		}
	}
	if languageName, ok := languageName[td.Node.LibraryInfo.GetLanguage()]; ok {
		md.Service.Language.Name = &languageName
	}
	switch td.SourceFormat {
	case "jaeger":
		jaegerVersion := strings.TrimPrefix(td.Node.LibraryInfo.ExporterVersion, "Jaeger-")
		agentName := "Jaeger"
		if md.Service.Language.Name != nil {
			agentName += "/" + *md.Service.Language.Name
		}
		md.Service.Agent.Name = &agentName
		md.Service.Agent.Version = &jaegerVersion
		if clientUUID, ok := td.Node.Attributes["client-uuid"]; ok {
			md.Service.Agent.EphemeralId = &clientUUID
		}
	case "zipkin":
		agentName := "Zipkin"
		md.Service.Agent.Name = &agentName

		// ipv4, ipv6, and port are populated based on the Zipkin endpoint.
		host := td.Node.Attributes["ipv4"]
		if host == "" {
			host = td.Node.Attributes["ipv6"]
		}
		hostport = net.JoinHostPort(host, td.Node.Attributes["port"])
	}

	transformables := make([]transform.Transformable, 0, len(td.Spans))
	for _, span := range td.Spans {
		var parentId *string
		root := len(span.ParentSpanId) == 0
		if !root {
			str := fmt.Sprintf("%x", span.ParentSpanId)
			parentId = &str
		}
		traceId := fmt.Sprintf("%x", span.TraceId)
		spanId := fmt.Sprintf("%x", span.SpanId)
		startTime := time.Unix(span.StartTime.Seconds, int64(span.StartTime.Nanos))
		endTime := time.Unix(span.EndTime.Seconds, int64(span.EndTime.Nanos))

		const logTraceData = true
		if logTraceData {
			fmt.Printf("%s: %s (%s)\n", td.Node.ServiceInfo.Name, span.Name.Value, span.Kind)
			fmt.Printf(".. TraceId: %x\n", span.TraceId)
			fmt.Printf(".. SpanId: %x\n", span.SpanId)
			fmt.Printf(".. ParentId: %x\n", span.ParentSpanId)
			if attrs := span.Attributes.GetAttributeMap(); len(attrs) > 0 {
				fmt.Println(".. span attributes:")
				for k, v := range attrs {
					fmt.Printf(".... %s: %v\n", k, v.Value)
				}
			}
		}

		// TODO(axw) span.StackTrace
		// TODO(axw) span.TimeEvents -> marks? More like events, as they hold more data.
		// TODO(axw) span.Resource, span.Attributes -> context
		// TODO(axw) span.Status -> result

		//pretty.Println("StackTrace:", span.StackTrace)
		//pretty.Println("TimeEvents:", span.TimeEvents)
		//pretty.Println("Resource:", span.Resource)
		//pretty.Println("Status:", span.Status)
		//pretty.Println("Attributes:", span.Attributes)

		if root || span.Kind == tracepb.Span_SERVER {
			labels := make(common.MapStr)
			event := &transaction.Event{
				Id:        spanId,
				ParentId:  parentId,
				TraceId:   traceId,
				Timestamp: startTime,
				Duration:  endTime.Sub(startTime).Seconds() * 1000,
				Type:      "request", // TODO(axw) based on attributes
				Name:      &span.Name.Value,
			}

			// NOTE(axw) this doesn't quite match our meaning, as ChildSpanCount
			// is only the direct children, whereas we're counting all spans
			// started within a transaction. Should we even report it?
			if count := int(span.GetChildSpanCount().GetValue()); count > 0 {
				event.SpanCount.Started = &count
			}

			var httpReq model.Req
			var httpResp model.Resp
			var http model.Http
			for k, v := range span.Attributes.GetAttributeMap() {
				switch v := v.Value.(type) {
				case *tracepb.AttributeValue_BoolValue:
					utility.DeepUpdate(labels, k, v.BoolValue)
				case *tracepb.AttributeValue_DoubleValue:
					utility.DeepUpdate(labels, k, v.DoubleValue)
				case *tracepb.AttributeValue_IntValue:
					switch k {
					case "http.status_code":
						intv := int(v.IntValue)
						httpResp.StatusCode = &intv
						http.Response = &httpResp
					default:
						utility.DeepUpdate(labels, k, v.IntValue)
					}
				case *tracepb.AttributeValue_StringValue:
					switch k {
					case "span.kind": // filter out
					case "http.method":
						httpReq.Method = v.StringValue.Value
						http.Request = &httpReq
					case "http.url", "http.path":
						event.Url = parseURL(v.StringValue.Value, hostport)
					case "http.status_code":
						// Zipkin sends http.status_code as a string.
						if intv, err := strconv.Atoi(v.StringValue.Value); err == nil {
							httpResp.StatusCode = &intv
							http.Response = &httpResp
						}
					case "http.protocol":
						switch v.StringValue.Value {
						case "HTTP/2":
							version := "2.0"
							http.Version = &version
						default:
							utility.DeepUpdate(labels, k, v.StringValue.Value)
						}
					default:
						utility.DeepUpdate(labels, k, v.StringValue.Value)
					}
				}
			}
			if http != (model.Http{}) {
				event.Http = &http
			}
			if false && len(labels) != 0 { // TODO(axw)
				labels := model.Labels(labels)
				event.Labels = &labels
			}
			transformables = append(transformables, event)
		} else {
			// NOTE(axw) we do not set TransactionId, as this
			// concept does not exist in OpenTelemetry. We could
			// look through the ancestry in td.Spans, but there's
			// no guarantee they're sent in the same batch.
			labels := make(common.MapStr)
			event := &model_span.Event{
				Id:        spanId,
				ParentId:  *parentId,
				TraceId:   traceId,
				Timestamp: startTime,
				Duration:  endTime.Sub(startTime).Seconds() * 1000,
				Type:      "request", // TODO(axw) based on attributes
				Name:      span.Name.Value,
			}

			httpAttrs := make(map[string]interface{})
			dbAttrs := make(map[string]interface{})
			for k, v := range span.Attributes.GetAttributeMap() {
				switch v := v.Value.(type) {
				case *tracepb.AttributeValue_BoolValue:
					utility.DeepUpdate(labels, k, v.BoolValue)
				case *tracepb.AttributeValue_DoubleValue:
					utility.DeepUpdate(labels, k, v.DoubleValue)
				case *tracepb.AttributeValue_IntValue:
					switch k {
					case "http.status_code":
						httpAttrs["status_code"] = float64(v.IntValue)
					default:
						utility.DeepUpdate(labels, k, v.IntValue)
					}
				case *tracepb.AttributeValue_StringValue:
					switch k {
					case "span.kind": // filter out
					case "http.url":
						httpAttrs["url"] = v.StringValue.Value
					case "sql.query":
						// NOTE(axw) in theory we could override
						// the span name here with our own method
						// based on the SQL query.
						dbAttrs["statement"] = v.StringValue.Value
					default:
						utility.DeepUpdate(labels, k, v.StringValue.Value)
					}
				}
				if len(httpAttrs) != 0 {
					event.Http, _ = model_span.DecodeHttp(map[string]interface{}{"http": httpAttrs}, nil)
				}
				if len(dbAttrs) != 0 {
					dbAttrs["type"] = "sql"
					event.Db, _ = model_span.DecodeDb(map[string]interface{}{"db": dbAttrs}, nil)
				}
			}
			if false && len(labels) != 0 { // TODO(axw)
				event.Labels = labels
			}
			transformables = append(transformables, event)
		}
	}
	return md, transformables
}

func parseURL(original, hostname string) *model.Url {
	url, err := url.Parse(original)
	if err != nil {
		return &model.Url{Original: &original}
	}
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	if url.Host == "" {
		url.Host = hostname
	}
	full := url.String()
	out := &model.Url{
		Original: &original,
		Scheme:   &url.Scheme,
		Full:     &full,
	}
	if url.Path != "" {
		out.Path = &url.Path
	}
	if url.RawQuery != "" {
		out.Query = &url.RawQuery
	}
	if url.Fragment != "" {
		out.Fragment = &url.Fragment
	}
	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		host = url.Host
		port = ""
	}
	if host != "" {
		out.Domain = &host
	}
	if port != "" {
		if intv, err := strconv.Atoi(port); err == nil {
			out.Port = &intv
		}
	}
	return out
}

var languageName = map[commonpb.LibraryInfo_Language]string{
	1:  "C++",
	2:  "C#",
	3:  "Erlang",
	4:  "Go",
	5:  "Java",
	6:  "Node.js",
	7:  "PHP",
	8:  "Python",
	9:  "Ruby",
	10: "JavaScript",
}
