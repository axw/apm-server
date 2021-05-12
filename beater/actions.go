package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
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

var (
	// From https://github.com/iovisor/bpftrace/blob/master/src/lexer.l
	bpftraceIdentRegexp = regexp.MustCompile("[_a-zA-Z][_a-zA-Z0-9]*")
)

type apmActionHandler struct {
	logger   *logp.Logger
	pipeline beat.Pipeline
}

func (*apmActionHandler) Name() string {
	return "apm"
}

func (h *apmActionHandler) Execute(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	defer func() {
		if v := recover(); v != nil {
			h.logger.Errorf("recovered panic: %s: %s", v, debug.Stack())
		}
	}()

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

	// TODO(axw) ideally we would be able to pick out well known variables
	// from print(f) statements as well. We could do this by preprocessing
	// the bpftrace program and transforming printfs into something we will
	// understand, or by creating an entirely new language.
	program, mapKeys, err := preprocessBPFTraceProgram(program)
	if err != nil {
		return nil, err
	}
	foreachMapKey := func(mapName string, data interface{}, f func(data interface{}, fields common.MapStr)) {
		mapKeyNames, ok := mapKeys[mapName]
		if !ok {
			f(data, nil)
			return
		}
		mapData := data.(map[string]interface{})
		for k, data := range mapData {
			fields := make(common.MapStr)
			mapKeyValues := strings.Split(k, ",")
			for i, mapKeyName := range mapKeyNames {
				mapKeyValue := mapKeyValues[i]
				switch mapKeyName {
				case "pid":
					fields["process.pid"], _ = strconv.Atoi(mapKeyValue)
				case "tid":
					fields["process.thread.id"], _ = strconv.Atoi(mapKeyValue)
				case "uid":
					fields["user.id"] = mapKeyValue
				case "gid":
					fields["group.id"] = mapKeyValue
				case "comm":
					fields["process.name"] = mapKeyValue

				// builtins with no ECS mapping
				case "cgroup":
					fields["process.cgroup.id"] = mapKeyValue
				case "func":
					fields["function.name"] = mapKeyValue
				case "kstack", "ustack":
					frames := strings.Fields(mapKeyValue)
					fields[mapKeyName] = frames
				}
			}
			f(data, fields)
		}
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
			for name, mapData := range line["data"].(map[string]interface{}) {
				mapName := strings.TrimPrefix(name, "@")
				foreachMapKey(mapName, mapData, func(mapData interface{}, ecs common.MapStr) {
					fields := common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"map":                      common.MapStr{name: mapData},
					}
					fields.DeepUpdate(ecs)
					client.Publish(beat.Event{Timestamp: time.Now(), Fields: fields})
				})
			}

		// hist(...) and lhist(...)
		case "hist":
			for name, histData := range line["data"].(map[string]interface{}) {
				name := strings.TrimPrefix(name, "@")
				foreachMapKey(name, histData, func(histData interface{}, ecs common.MapStr) {
					var counts []int64
					var values []float64
					var lastMin, lastMax float64
					for _, histBucketData := range histData.([]interface{}) {
						count := histBucketData.(map[string]interface{})["count"].(float64)
						min, haveMin := histBucketData.(map[string]interface{})["min"].(float64)
						max, haveMax := histBucketData.(map[string]interface{})["max"].(float64)
						var value float64
						if haveMin {
							if haveMax {
								lastMax = max
							} else {
								// NOTE(axw) when using lhist, if there are values that
								// extend beyond the final bucket, the the final bucket
								// will have no "max". We report these as being within
								// the final bucket. Clearly wrong, not sure what else
								// we can do yet.
								max = min + (lastMax - lastMin)
							}
							value = (max - min) / 2
							lastMin = min
						} else {
							// For negative values, there may be no min.
							value = max
						}
						counts = append(counts, int64(count))
						values = append(values, float64(value))
					}
					fields := common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"hist." + name: common.MapStr{
							"values": values,
							"counts": counts,
						},
					}
					fields.DeepUpdate(ecs)
					client.Publish(beat.Event{Timestamp: time.Now(), Fields: fields})
				})
			}

		// stats(...)
		case "stats":
			for name, statsData := range line["data"].(map[string]interface{}) {
				name := strings.TrimPrefix(name, "@")
				foreachMapKey(name, statsData, func(statsData interface{}, ecs common.MapStr) {
					count := statsData.(map[string]interface{})["count"].(float64)
					total := statsData.(map[string]interface{})["total"].(float64)
					fields := common.MapStr{
						datastreams.TypeField:      "metrics",
						datastreams.DatasetField:   "bpftrace",
						datastreams.NamespaceField: "default",
						"action_id":                actionID,
						"stats." + name: common.MapStr{
							"value_count": count,
							"sum":         total,
						},
					}
					fields.DeepUpdate(ecs)
					client.Publish(beat.Event{Timestamp: time.Now(), Fields: fields})
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

func preprocessBPFTraceProgram(program string) (string, map[string][]string, error) {
	// TODO(axw) parse the program producing an AST, identify associative arrays
	// and keys correctly. This is just demoware.
	var mapKeys map[string][]string
	remaining := program
	for remaining != "" {
		i := strings.IndexRune(remaining, '@')
		if i == -1 {
			break
		}
		remaining = remaining[i+1:]
		lbracket := strings.IndexRune(remaining, '[')
		if lbracket == -1 {
			break
		}
		name := remaining[:lbracket]
		remaining = remaining[lbracket+1:]
		rbracket := strings.IndexRune(remaining, ']')
		if rbracket == -1 {
			break
		}
		if name != "" && !bpftraceIdentRegexp.MatchString(name) {
			continue
		}
		keys := remaining[:rbracket]
		remaining = remaining[rbracket+1:]
		if mapKeys == nil {
			mapKeys = make(map[string][]string)
		}
		mapKeys[name] = strings.Split(keys, ",")
	}
	return program, mapKeys, nil
}
