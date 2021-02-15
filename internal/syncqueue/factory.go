package syncqueue

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

func init() {
	queue.RegisterQueueType(
		"sync", newQueue,
		feature.MakeDetails("Sync queue", "Sends events directly to the output.", feature.Stable),
	)
}

func newQueue(
	ackListener queue.ACKListener, logger *logp.Logger, cfg *common.Config, inQueueSize int,
) (queue.Queue, error) {
	/*
	   config := defaultConfig
	   if err := cfg.Unpack(&config); err != nil {
	           return nil, err
	   }

	   if logger == nil {
	           logger = logp.L()
	   }

	   return NewQueue(logger, Settings{
	           ACKListener:    ackListener,
	           Events:         config.Events,
	           FlushMinEvents: config.FlushMinEvents,
	           FlushTimeout:   config.FlushTimeout,
	           InputQueueSize: inQueueSize,
	   }), nil
	*/
	return newSyncQueue(logger), nil
}
