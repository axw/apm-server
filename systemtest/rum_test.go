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

package systemtest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"

	"github.com/elastic/apm-server/systemtest"
	"github.com/elastic/apm-server/systemtest/apmservertest"
	"github.com/elastic/apm-server/systemtest/estest"
	"github.com/elastic/apm-server/systemtest/internal/sourcemap"
)

func TestRUMXForwardedFor(t *testing.T) {
	systemtest.CleanupElasticsearch(t)
	srv := apmservertest.NewUnstartedServer(t)
	srv.Config.RUM = &apmservertest.RUMConfig{Enabled: true}
	err := srv.Start()
	require.NoError(t, err)

	serverURL, err := url.Parse(srv.URL)
	require.NoError(t, err)
	serverURL.Path = "/intake/v2/rum/events"

	const body = `{"metadata":{"service":{"name":"rum-js-test","agent":{"name":"rum-js","version":"5.5.0"}}}}
{"transaction":{"trace_id":"611f4fa950f04631aaaaaaaaaaaaaaaa","id":"611f4fa950f04631","type":"page-load","duration":643,"span_count":{"started":0}}}`

	req, _ := http.NewRequest("POST", serverURL.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("X-Forwarded-For", "220.244.41.16")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	result := systemtest.Elasticsearch.ExpectDocs(t, "apm-*", estest.TermQuery{Field: "processor.event", Value: "transaction"})
	systemtest.ApproveEvents(
		t, t.Name(), result.Hits.Hits,
		// RUM timestamps are set by the server based on the time the payload is received.
		"@timestamp", "timestamp.us",
	)
}

func TestRUMErrorSourcemapping(t *testing.T) {
	systemtest.CleanupElasticsearch(t)
	srv := apmservertest.NewUnstartedServer(t)
	srv.Config.RUM = &apmservertest.RUMConfig{Enabled: true}
	err := srv.Start()
	require.NoError(t, err)

	uploadSourcemap(t, srv, "../testdata/sourcemap/bundle.js.map",
		"http://localhost:8000/test/e2e/../e2e/general-usecase/bundle.js.map", // bundle filepath
		"apm-agent-js", // service name
		"1.0.1",        // service version
	)
	systemtest.Elasticsearch.ExpectDocs(t, "apm-*-sourcemap", nil)

	sendRUMEventsPayload(t, srv, "../testdata/intake-v2/errors_rum.ndjson")
	result := systemtest.Elasticsearch.ExpectDocs(t, "apm-*-error", nil)

	systemtest.ApproveEvents(
		t, t.Name(), result.Hits.Hits,
		// RUM timestamps are set by the server based on the time the payload is received.
		"@timestamp", "timestamp.us",
	)
}

func TestRUMErrorSourcemapEnrichment(t *testing.T) {
	const (
		sourcemapIndex   = "apm-rum-sourcemaps"
		enrichPolicyName = "apm-rum-sourcemaps"
		matchField       = "rum.service.name_version"
	)
	systemtest.CleanupElasticsearch(t)
	systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.EnrichDeletePolicyRequest{Name: enrichPolicyName},
		nil,
	)
	systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.DeleteScriptRequest{ScriptID: "apm_apply_sourcemap"},
		nil,
	)

	// Create the stored script which will be executed by the
	// sourcemap pipeline to fuzzy-match mappings to stacktrace
	// frames.
	storeIngestScript(t, "apm_apply_sourcemap", `
boolean pred(Map mapping, Map frame) {
  int cmp = (int)mapping.get("gen_line") - (int)frame.line.number;
  if (cmp == 0) {
    return (int)mapping.get("gen_column") >= (int)frame.line.column;
  }
  return cmp > 0;
}

int binarySearch(List mappings, Map frame) {
    int i = 0;
    int j = mappings.size();
    while (i < j) {
        int h = (int)((i+j) / 2);
        if (!pred(mappings.get(h), frame)) {
            i = h + 1;
        } else {
            j = h;
        }
    }
    return i;
}

Map getMatch(List mappings, Map frame) {
  int i = binarySearch(mappings, frame);
  if (i == mappings.size()) {
    return null;
  }
  Map match = mappings.get(i);
  if ((int)match.get("gen_line") > (int)frame.line.number ||
      (int)match.get("gen_column") > (int)frame.line.column) {
    if (i == 0) {
      return null;
    }
    match = mappings.get(i-1);
  }
  return match;
}

void updateStacktrace(List stacktrace, Map sourcemaps) {
  if (stacktrace == null) {
    return;
  }
  def function = "<anonymous>";
  for (int i = stacktrace.size()-1; i >= 0; i--) {
    def frame = stacktrace[i];
    def bundle_filepath = frame.remove("rum.bundle_filepath");
    if (bundle_filepath == null) {
      continue;
    }
    Map sourcemap = sourcemaps.get(bundle_filepath);
    if (sourcemap == null) {
      continue;
    }
    def match = getMatch(sourcemap.mappings, frame);
    if (match != null) {
      // Record original source information.
      //
      // TODO(axw) set fields conditionally
      frame.original = new HashMap();
      frame.original.library_frame = frame.library_frame;
      frame.original.filename = frame.filename;
      frame.original.classname = frame.classname;
      frame.original.abs_path = frame.abs_path;
      frame.original.function = frame.function;
      frame.original.lineno = frame.line.number;
      frame.original.colno = frame.line.column;

      // Set new source information.
      frame.line.number = match.source_line;
      frame.line.column = match.source_column;
      frame.filename = match.source;
      frame.function = function;

      def name = match.get("name");
      if (name != null) {
        function = name;
      } else {
        function = "<unknown>";
      }

      def sourceContent = sourcemap.source_content.get(match.source);
      def numLines = sourceContent.size();
      if (match.source_line < numLines) {
          frame.line.context = sourceContent[match.source_line-1];

          // TODO(axw) make the number of pre/post context lines a parameter.
	  int preIndex = (int)Math.max(match.source_line-5-1, 0);
	  int postIndex = (int)Math.min(match.source_line+5, numLines);
	  frame.context = new HashMap();
	  frame.context.pre = sourceContent.subList(preIndex, match.source_line-1);
	  frame.context.post = sourceContent.subList(match.source_line, postIndex);
      }
    }
  }
}

// TODO(axw) sourcemap docs should have a unique key
// <service.name, service.version, bundle_filepath>
Map sourcemaps = new HashMap();
for (sourcemap in ctx.sourcemap) {
  sourcemaps[sourcemap.bundle_filepath] = sourcemap;
}

updateStacktrace(ctx?.span?.stacktrace, sourcemaps);
updateStacktrace(ctx?.error?.log?.stacktrace, sourcemaps);
for (exception in ctx?.error?.exception) {
  updateStacktrace(exception.stacktrace, sourcemaps);
}
`)

	// Create the stored script which will be executed by the
	// sourcemap pipeline to identify stacktrace library frames
	// after applying sourcemaps.
	storeIngestScript(t, "apm_set_library_frames", `
boolean match(String s, String[] parts) {
  return false;
}

boolean matchAny(String s, List any_parts) {
  for (parts in any_parts) {
    if (match(s, (String[])parts)) {
      return true;
    }
  }
  return false;
}

void setLibraryFrames(List stacktrace, List library_patterns_parts) {
  if (stacktrace == null) {
    return;
  }
  
  for (frame in stacktrace) {
    if (frame.filename != "") {
      frame.library_frame = matchAny(frame.filename, library_patterns_parts);
    }
  }
}

List library_patterns = ctx.rum?.library_patterns;
if (library_patterns != null) {
  List library_patterns_parts = new ArrayList();
  for (library_pattern in library_patterns) {
    library_patterns_parts.add(library_pattern.splitOnToken("*"));
  }

  setLibraryFrames(ctx?.span?.stacktrace, library_patterns_parts);
  setLibraryFrames(ctx?.error?.log?.stacktrace, library_patterns_parts);
  for (exception in ctx?.error?.exception) {
    setLibraryFrames(exception.stacktrace, library_patterns_parts);
  }
}
`)

	// Create the stored script which will be executed by the
	// sourcemap pipeline to set the error culprit after applying
	// sourcemaps.
	storeIngestScript(t, "apm_set_error_culprit", `
Map firstApplicationFrame(List stacktrace) {
  if (stacktrace == null) {
    return null;
  }
  for (frame in stacktrace) {
    if (!frame.library_frame) {
      return frame;
    }
  }
  return null;
}

def app_frame = firstApplicationFrame(ctx.error.log?.stacktrace);
if (app_frame == null) {
  for (exception in ctx.error?.exception) {
    app_frame = firstApplicationFrame(exception.stacktrace);
    if (app_frame != null) {
      break;
    }
  }
}
if (app_frame != null) {
  ctx.error.culprit = app_frame.filename;
  if (app_frame.function != null) {
    ctx.error.culprit += " in " + app_frame.function;
  }
}
`)

	// Create the raw sourcemap index.
	var indexDefinition struct {
		Mappings struct {
			Properties map[string]interface{} `json:"properties"`
		} `json:"mappings"`
	}
	indexDefinition.Mappings.Properties = map[string]interface{}{
		matchField: map[string]interface{}{"type": "keyword"},
	}
	_, err := systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.IndicesCreateRequest{
			Index: sourcemapIndex,
			Body:  esutil.NewJSONReader(indexDefinition),
		},
		nil,
	)
	require.NoError(t, err)

	// Create the enrich policy which will take documents from
	// the raw sourcemap index.
	var enrichPolicy struct {
		Match struct {
			Indices      []string `json:"indices"`
			MatchField   string   `json:"match_field"`
			EnrichFields []string `json:"enrich_fields"`
		} `json:"match"`
	}
	enrichPolicy.Match.Indices = []string{sourcemapIndex}
	enrichPolicy.Match.MatchField = matchField
	enrichPolicy.Match.EnrichFields = []string{"mappings", "source_content"}
	_, err = systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.EnrichPutPolicyRequest{
			Name: enrichPolicyName,
			Body: esutil.NewJSONReader(enrichPolicy),
		},
		nil,
	)
	require.NoError(t, err)

	// Index a sourcemap document, with the same service name/version
	// as in the RUM error event we send subsequently.
	f, err := os.Open("../testdata/sourcemap/bundle.js.map")
	require.NoError(t, err)
	defer f.Close()
	parsedSourcemap, err := sourcemap.Parse(f)
	require.NoError(t, err)
	var sourcemapWithService struct {
		RUMServiceNameVersion string `json:"rum.service.name_version"`
		BundleFilepath        string `json:"bundle_filepath"`
		*sourcemap.Sourcemap
	}
	sourcemapWithService.Sourcemap = parsedSourcemap
	sourcemapWithService.RUMServiceNameVersion = "apm-agent-js/1.0.1"
	sourcemapWithService.BundleFilepath = "/test/e2e/general-usecase/bundle.js.map"
	_, err = systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.IndexRequest{
			Index: sourcemapIndex,
			Body:  esutil.NewJSONReader(sourcemapWithService),
			// Refresh to ensure the enrichment policy execution
			// can see the document.
			Refresh: "true",
		},
		nil,
	)
	require.NoError(t, err)

	// Execute the enrich policy.
	_, err = systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.EnrichExecutePolicyRequest{Name: enrichPolicyName},
		nil,
	)
	require.NoError(t, err)

	// Send a RUM error, and check that the sourcemap is applied.
	srv := apmservertest.NewUnstartedServer(t)
	srv.Config.RUM = &apmservertest.RUMConfig{Enabled: true}
	require.NoError(t, srv.Start())
	sendRUMEventsPayload(t, srv, "../testdata/intake-v2/errors_rum.ndjson")
	result := systemtest.Elasticsearch.ExpectDocs(t, "apm-*-error", nil)
	_ = result

	/*
		systemtest.Elasticsearch.ExpectDocs(t, "apm-*-sourcemap", nil)
	*/
}

func sendRUMEventsPayload(t *testing.T, srv *apmservertest.Server, payloadFile string) {
	t.Helper()

	f, err := os.Open(payloadFile)
	require.NoError(t, err)
	defer f.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/intake/v2/rum/events", f)
	req.Header.Add("Content-Type", "application/x-ndjson")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode, string(respBody))
}

func storeIngestScript(t *testing.T, id, scriptSource string) {
	_, err := systemtest.Elasticsearch.Do(
		context.Background(),
		esapi.PutScriptRequest{
			ScriptID:      id,
			ScriptContext: "ingest",
			Body: esutil.NewJSONReader(map[string]interface{}{
				"script": map[string]interface{}{
					"lang":   "painless",
					"source": scriptSource,
				},
			}),
		},
		nil,
	)
	if estestError, ok := err.(*estest.Error); ok {
		var m map[string]interface{}
		fmt.Println(json.Unmarshal(estestError.Body, &m))
		t.Logf("%v", m)
	}
	require.NoError(t, err)
}

func uploadSourcemap(t *testing.T, srv *apmservertest.Server, sourcemapFile, bundleFilepath, serviceName, serviceVersion string) {
	t.Helper()

	var data bytes.Buffer
	mw := multipart.NewWriter(&data)
	require.NoError(t, mw.WriteField("service_name", serviceName))
	require.NoError(t, mw.WriteField("service_version", serviceVersion))
	require.NoError(t, mw.WriteField("bundle_filepath", bundleFilepath))

	f, err := os.Open(sourcemapFile)
	require.NoError(t, err)
	defer f.Close()
	sourcemapFileWriter, err := mw.CreateFormFile("sourcemap", filepath.Base(sourcemapFile))
	require.NoError(t, err)
	_, err = io.Copy(sourcemapFileWriter, f)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, _ := http.NewRequest("POST", srv.URL+"/assets/v1/sourcemaps", &data)
	req.Header.Add("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode, string(respBody))
}
