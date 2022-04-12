package kafkaconsumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/apm-server/beater/kafkaconsumer"
	"github.com/elastic/apm-server/model"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	consumer := kafkaconsumer.New(kafkaconsumer.Config{})
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := consumer.Consume(ctx, model.ProcessBatchFunc(func(ctx context.Context, batch *model.Batch) error {
		return nil
	}))
	require.NoError(t, err)
}
