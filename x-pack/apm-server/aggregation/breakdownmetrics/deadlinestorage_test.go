package breakdownmetrics_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-server/x-pack/apm-server/aggregation/breakdownmetrics"
	"github.com/elastic/apm-server/x-pack/apm-server/sampling/eventstorage"
)

func TestTraceDeadlines(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "breakdownmetrics")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempdir) })

	badgerDB, err := eventstorage.OpenBadger(tempdir, 0)
	require.NoError(t, err)
	t.Cleanup(func() { badgerDB.Close() })

	storage := breakdownmetrics.NewDeadlineStorage(badgerDB)
	defer func() {
		err := storage.Flush()
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	deadlines := make(chan breakdownmetrics.TraceDeadline)
	go func() {
		defer close(deadlines)
		err = storage.ReadTraceDeadlines(ctx, time.Millisecond, deadlines)
		assert.Equal(t, context.DeadlineExceeded, err)
	}()

	err = storage.WriteTraceDeadline("trace_3", time.Now().Add(time.Second))
	require.NoError(t, err)
	err = storage.WriteTraceDeadline("trace_2", time.Now().Add(2*time.Second))
	require.NoError(t, err)
	err = storage.WriteTraceDeadline("trace_1", time.Now().Add(3*time.Second))
	require.NoError(t, err)

	for deadline := range deadlines {
		fmt.Println(deadline)
	}
}
