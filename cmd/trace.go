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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/elastic/go-elasticsearch/v7/esutil"

	"github.com/elastic/apm-server/elasticsearch"
)

func traceCmd(settings instance.Settings) *cobra.Command {
	cmd := &cobra.Command{Use: "bpftrace"}
	cmd.AddCommand(traceRunCmd(settings))
	cmd.AddCommand(traceResultCmd(settings))
	return cmd
}

func traceRunCmd(settings instance.Settings) *cobra.Command {
	var agents []string
	var duration, timeout time.Duration
	var program string
	short := "Run a bpftrace program on APM Server hosts"
	cmd := &cobra.Command{
		Use:   "run",
		Short: short,
		Long:  short,
		Run: func(cmd *cobra.Command, args []string) {
			esClient, _, err := bootstrap(settings)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), duration+timeout)
			defer cancel()
			actionID, err := runBPFTrace(ctx, esClient, program, duration, agents)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
			cmd.Printf("created action %q\n", actionID)
			result, err := getActionResult(ctx, esClient, actionID)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
			cmd.Printf("%s\n", result)
		},
	}
	flags := cmd.Flags()
	flags.StringSliceVar(&agents, "agents", nil, "agents to run bpfprogram")
	flags.DurationVar(&duration, "duration", time.Second, "duration to run the bpftrace program")
	flags.DurationVar(&timeout, "timeout", 10*time.Second, "timeout waiting for result after specified duration")
	flags.StringVar(&program, "program", "", "bpftrace program")
	return cmd
}

func traceResultCmd(settings instance.Settings) *cobra.Command {
	var timeout time.Duration
	short := "Fetch the results of a previous `bpftrace run`"
	cmd := &cobra.Command{
		Use:   "result",
		Short: short,
		Long:  short,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			esClient, _, err := bootstrap(settings)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			actionID := args[0]
			result, err := getActionResult(ctx, esClient, actionID)
			if err != nil {
				cmd.PrintErrln(err)
				os.Exit(1)
			}
			cmd.Printf("%s\n", result)
		},
	}
	flags := cmd.Flags()
	flags.DurationVar(&timeout, "timeout", 10*time.Second, "timeout waiting for result")
	return cmd
}

func runBPFTrace(
	ctx context.Context,
	esClient elasticsearch.Client,
	program string,
	duration time.Duration,
	agents []string,
) (string, error) {
	return createAction(ctx, esClient, "apm", agents, "nobody", time.Hour, map[string]interface{}{
		"bpftrace_program":  program,
		"bpftrace_duration": duration.String(),
	})
}

func createAction(
	ctx context.Context, esClient elasticsearch.Client,
	inputType string,
	agents []string,
	userID string,
	ttl time.Duration,
	data map[string]interface{},
) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("invalid TTL %s, must be >= 0", ttl)
	}
	if len(agents) == 0 {
		return "", fmt.Errorf("no agents specified")
	}

	timestamp := time.Now().UTC()
	expiration := timestamp.Add(ttl)
	actionID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	actionIDString := actionID.String()
	fields := map[string]interface{}{
		"@timestamp": timestamp,
		"expiration": expiration,
		"type":       "INPUT_ACTION",
		"input_type": inputType,

		"action_id": actionIDString,
		"agents":    agents,
		"data":      data,
		"user_id":   userID,
	}

	resp, err := esapi.IndexRequest{
		Index: ".fleet-actions",
		Body:  esutil.NewJSONReader(fields),
	}.Do(ctx, esClient)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed: %s", body)
	}

	return actionIDString, nil
}

func getActionResult(ctx context.Context, esClient elasticsearch.Client, actionID string) (*actionResult, error) {
	header := make(http.Header)
	header.Set("X-elastic-product-origin", "kibana") // pretend we're kibana so we can query system data stream

	type hit struct {
		Index  string          `json:"_index"`
		Source json.RawMessage `json:"_source"`
	}

	search := func(index ...string) ([]hit, error) {
		req := esapi.SearchRequest{
			Index:  index,
			Header: header,
			Pretty: true,
			Body: esutil.NewJSONReader(map[string]interface{}{
				"query": map[string]interface{}{
					"term": map[string]interface{}{
						"action_id": actionID,
					},
				},
			}),
		}
		resp, err := req.Do(ctx, esClient)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.IsError() {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("error searching results: %q", body)
		}

		var result struct {
			Hits struct {
				Hits []hit `json:"hits"`
			} `json:"hits"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		return result.Hits.Hits, nil
	}

	for {
		hits, err := search(".fleet-actions-results")
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		actionResult := actionResult{fleetActionResult: hits[0].Source}

		// Fleet action result should only be visible after the related
		// logs and metrics have been indexed. Refresh indices to make
		// sure they're visible in searches.
		indices := []string{"logs-bpftrace-*", "metrics-bpftrace-*"}
		req := esapi.IndicesRefreshRequest{Index: indices}
		resp, err := req.Do(ctx, esClient)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		hits, err = search(indices...)
		if err != nil {
			return nil, err
		}
		for _, hit := range hits {
			hit.Index = strings.TrimPrefix(hit.Index, ".ds-")
			switch {
			case strings.HasPrefix(hit.Index, "logs-"):
				actionResult.logs = append(actionResult.logs, hit.Source)
			case strings.HasPrefix(hit.Index, "metrics-"):
				actionResult.metrics = append(actionResult.metrics, hit.Source)
			}
		}
		return &actionResult, nil
	}
}

type actionResult struct {
	fleetActionResult []byte
	logs              [][]byte
	metrics           [][]byte
}
