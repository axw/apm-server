package kafkaconsumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Shopify/sarama"

	"github.com/elastic/apm-server/model"
)

type Config struct {
	Brokers         []string
	Topics          []string
	ConsumerGroupID string
}

type Consumer struct {
	cfg   Config
	group sarama.ConsumerGroup
}

func New(cfg Config) (*Consumer, error) {
	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroupID, nil)
	if err != nil {
		return nil, err
	}
	return &Consumer{cfg: cfg, group: group}, nil
}

func (c *Consumer) Close() error {
	return c.group.Close()
}

func (c *Consumer) Consume(ctx context.Context, process model.BatchProcessor) (err error) {
	return c.group.Consume(ctx, c.cfg.Topics, consumerGroupHandler{process})
}

type consumerGroupHandler struct {
	processor model.BatchProcessor
}

func (h consumerGroupHandler) Setup(s sarama.ConsumerGroupSession) error {
	//log.Printf("consuming claims: %+v", s.Claims())
	return nil
}

func (h consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		//log.Printf("consuming message from %q", message.Topic)

		// Mark the message as consumed first: at most once delivery.
		session.MarkMessage(message, "")

		// Decode and process the event.
		//
		// TODO(axw) we should either refactor BatchProcessor to process a single
		// event at a time, and perhaps only batch before libbeat outputs; or we
		// should consider consuming a batch of events before processing below.
		batch := make(model.Batch, 1)
		if err := json.Unmarshal(message.Value, &batch[0]); err != nil {
			log.Fatalf("failed to decode event: %w", err)
		}
		if err := h.processor.ProcessBatch(session.Context(), &batch); err != nil {
			return err
		}
	}
	return nil
}
