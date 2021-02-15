package syncqueue

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

type syncQueue struct {
	logger *logp.Logger
	events chan publisher.Event
}

func newSyncQueue(logger *logp.Logger) *syncQueue {
	return &syncQueue{logger: logger, events: make(chan publisher.Event)}
}

func (q *syncQueue) Close() error {
	// TODO
	return nil
}

func (q *syncQueue) BufferConfig() queue.BufferConfig {
	return queue.BufferConfig{}
}

func (q *syncQueue) Producer(cfg queue.ProducerConfig) queue.Producer {
	fmt.Println("Producer!")
	return &syncQueueProducer{q.events}
}

func (q *syncQueue) Consumer() queue.Consumer {
	fmt.Println("Consumer!")
	return &syncQueueConsumer{q.events}
}

type syncQueueProducer struct {
	events chan<- publisher.Event
}

func (p *syncQueueProducer) Publish(event publisher.Event) bool {
	p.events <- event
	return true
}

func (p *syncQueueProducer) TryPublish(event publisher.Event) bool {
	select {
	case p.events <- event:
		return true
	default:
		return false
	}
}

func (p *syncQueueProducer) Cancel() int {
	return 0
}

type syncQueueConsumer struct {
	events <-chan publisher.Event
}

func (q *syncQueueConsumer) Get(eventCount int) (queue.Batch, error) {
	event, ok := <-q.events
	if !ok {
		return nil, errors.New("channel closed")
	}
	return singleEventBatch{event: [1]publisher.Event{event}}, nil
}

func (q *syncQueueConsumer) Close() error {
	return nil
}

type singleEventBatch struct {
	event [1]publisher.Event
}

func (b singleEventBatch) Events() []publisher.Event {
	return b.event[:]
}

func (b singleEventBatch) ACK() {}
