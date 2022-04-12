package kafkaproducer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Shopify/sarama"

	"github.com/elastic/apm-server/model"
)

type Config struct {
	Broker string
	Topic  string

	// Key, if non-nil, is used to set the key for an event message.
	//
	// Producer uses murmur2 partitioning: consistent hashing over the
	// message key, if it is non-nil; random partition assignment if the
	// message has no key.
	//
	// If Key is nil, the message key will be set to the event's trace ID
	// if there is one, and nil otherwise.
	Key func(model.APMEvent) []byte
}

type Producer struct {
	cfg      Config
	producer sarama.AsyncProducer
}

func New(cfg Config) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	//saramaConfig.Producer.Partitioner // TODO
	producer, err := sarama.NewAsyncProducer([]string{cfg.Broker}, saramaConfig)
	if err != nil {
		return nil, err
	}
	// TODO(axw) use errgroup
	go func() {
		for err := range producer.Errors() {
			log.Printf("producer error: %s (%#v)", err, err.Err)
		}
	}()
	return &Producer{cfg: cfg, producer: producer}, nil
}

func (p *Producer) Close() error {
	return p.producer.Close()
}

func (p *Producer) ProcessBatch(ctx context.Context, batch *model.Batch) error {
	for _, event := range *batch {
		message := &sarama.ProducerMessage{
			Topic:     p.cfg.Topic,
			Timestamp: event.Timestamp,
		}

		encoded, err := json.Marshal(&event)
		if err != nil {
			return err
		}
		message.Value = sarama.ByteEncoder(encoded)

		if p.cfg.Key != nil {
			message.Key = sarama.ByteEncoder(p.cfg.Key(event))
		} else if event.Trace.ID != "" {
			// Set Key to Trace.ID if there is one. If there isn't,
			// the message will be sent to a random partition.
			message.Key = sarama.StringEncoder(event.Trace.ID)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.producer.Input() <- message:
		}
	}
	return nil
}
