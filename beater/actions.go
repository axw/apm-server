package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/elastic/apm-server/datastreams"
)

const (
	actionKeyBPFTraceProgram  = "bpftrace_program"
	actionKeyBPFTraceDuration = "bpftrace_duration"
)

type apmActionHandler struct {
	logger   *logp.Logger
	pipeline beat.Pipeline
}

func (*apmActionHandler) Name() string {
	return "apm"
}

func (h *apmActionHandler) Execute(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	// TODO(axw) add tracing to Fleet Actions?
	h.logger.Infof("Execute: %+v", args)

	actionID := args["id"].(string)
	data := args["data"].(map[string]interface{})
	program, ok := data[actionKeyBPFTraceProgram].(string)
	if !ok {
		return nil, fmt.Errorf("missing arg %q", actionKeyBPFTraceProgram)
	}
	durationString, ok := data[actionKeyBPFTraceDuration].(string)
	if !ok {
		return nil, fmt.Errorf("missing arg %q", actionKeyBPFTraceDuration)
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return nil, err
	}

	// Run the program, sending SIGTERM after the specified duration elapses.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bpftrace", "-f", "json", "-e", program)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		defer cmd.Wait()
		select {
		case <-ctx.Done():
			return
		case <-time.After(duration):
			cmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	acker := newWaitPublishedAcker()
	acker.Open()
	client, err := h.pipeline.ConnectWith(beat.ClientConfig{
		CloseRef:   ctx,
		ACKHandler: acker,
	})
	if err != nil {
		return nil, err
	}
	defer client.Close()

	decoder := json.NewDecoder(stdout)
	for {
		line := make(map[string]interface{})
		if err := decoder.Decode(&line); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// TODO(axw) ideally we would be able to pick out well known variables
		// from print(f) statements and associative arrays and add them as ECS
		// fields. We could do this by preprocessing the bpftrace program and
		// transforming printfs and associative array keys to something we will
		// understand, or by creating an entirely new language.

		switch t := line["type"]; t {
		case "attached_probes":
		case "printf":
			client.Publish(beat.Event{
				Timestamp: time.Now(),
				Fields: common.MapStr{
					datastreams.TypeField:      "logs",
					datastreams.DatasetField:   "bpftrace",
					datastreams.NamespaceField: "default",
					"action_id":                actionID,
					"message":                  line["data"],
				},
			})

		// count(), sum(...), @foo[bar] = ...
		case "map":
			for key, mapData := range line["data"].(map[string]interface{}) {
				key := strings.TrimPrefix(key, "@")
				client.Publish(beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"map." + key:               mapData,
					},
				})
			}

		// hist(...) and lhist(...)
		case "hist":
			for key, histData := range line["data"].(map[string]interface{}) {
				key := strings.TrimPrefix(key, "@")
				var counts []int64
				var values []float64
				for _, histBucketData := range histData.([]interface{}) {
					count := histBucketData.(map[string]interface{})["count"].(float64)
					value := histBucketData.(map[string]interface{})["min"].(float64)
					counts = append(counts, int64(count))
					values = append(values, float64(value))
				}
				client.Publish(beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"hist." + key: common.MapStr{
							"values": values,
							"counts": counts,
						},
					},
				})
			}

		// stats(...)
		case "stats":
			for key, statsData := range line["data"].(map[string]interface{}) {
				key := strings.TrimPrefix(key, "@")
				count := statsData.(map[string]interface{})["count"].(float64)
				total := statsData.(map[string]interface{})["total"].(float64)
				client.Publish(beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"stats." + key: common.MapStr{
							"value_count": count,
							"sum":         total,
						},
					},
				})
			}
		default:
			h.logger.Debug("unknown type %q", t)
		}
	}
	if err := client.Close(); err != nil {
		return nil, err
	}
	// Wait for published events to be acked before returning, so the
	// initiator can be sure logs and metrics have been indexed before
	// the action result.
	//
	// TODO(axw) make this optional?
	if err := acker.Wait(ctx); err != nil {
		return nil, err
	}
	return make(map[string]interface{}), nil
}
