// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sampling

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/elastic/apm-server/model"
)

var errTooManyTraceGroups = errors.New("too many trace groups")

type traceGroups struct {
	// defaultSamplingFraction holds the default sampling fraction to
	// set for new trace groups. See traceGroup.samplingFraction.
	defaultSamplingFraction float64

	mu sync.RWMutex

	// TODO(axw) we should enable the user to define specific trace groups,
	// along with tail sampling rates. The "groups" field would then track
	// other trace groups, up to a configured maximum.
	//
	// NOTE(axw) we currently do not ever delete groups, as we want to keep
	// a record of the ingest rate. We should eventually have something like
	// an LRU for the non-user defined trace groups, so we can keep track of
	// ingest rate as a best effort, while allowing new trace groups to
	// displace old ones.

	groups      map[uint64][]*traceGroup
	groupsCount int
	groupsSpace []traceGroup
}

func newTraceGroups(maxGroups int, defaultSamplingFraction float64) *traceGroups {
	return &traceGroups{
		defaultSamplingFraction: defaultSamplingFraction,
		groups:                  make(map[uint64][]*traceGroup),
		groupsSpace:             make([]traceGroup, maxGroups),
	}
}

type traceGroup struct {
	traceGroupKey

	// samplingFraction holds the configured fraction of traces in this
	// trace group to sample, as a fraction in the range (0,1).
	samplingFraction float64

	mu sync.Mutex
	// reservoir holds a random sample of root transactions observed
	// for this trace group, weighted by duration.
	reservoir *weightedRandomSample
	// total holds the total number of root transactions observed for
	// this trace group, including those that are not added to the
	// reservoir. This is used to update ingestRate.
	total int
	// ingestRate holds the exponentially weighted moving average number
	// of root transactions observed for this trace group per tail
	// sampling interval. This is read and written only by tailSampler.
	ingestRate float64
}

type traceGroupKey struct {
	// TODO(axw) review grouping attributes
	serviceName       string
	transactionName   string
	transactionResult string
}

func (k *traceGroupKey) hash() uint64 {
	var h xxhash.Digest
	h.WriteString(k.serviceName)
	h.WriteString(k.transactionName)
	h.WriteString(k.transactionResult)
	return h.Sum64()
}

// sampleTrace will return true if the root transaction is admitted to
// the in-memory sampling reservoir, and false otherwise.
//
// If the transaction is not admitted due to the transaction group limit
// having been reached, sampleTrace will return errTooManyTraceGroups.
func (g *traceGroups) sampleTrace(tx *model.Transaction) (bool, error) {
	key := traceGroupKey{
		serviceName:       tx.Metadata.Service.Name,
		transactionName:   tx.Name,
		transactionResult: tx.Result,
	}
	hash := key.hash()

	g.mu.RLock()
	entries, ok := g.groups[hash]
	g.mu.RUnlock()
	var offset int
	if ok {
		for offset = range entries {
			if entries[offset].traceGroupKey == key {
				group := entries[offset]
				group.mu.Lock()
				defer group.mu.Unlock()
				group.total++
				return group.reservoir.Sample(tx.Duration, tx.TraceID), nil
			}
		}
		offset++ // where to start searching withe the write lock below
	}

	g.mu.Lock()
	entries, ok = g.groups[hash]
	if ok {
		for _, group := range entries[offset:] {
			if group.traceGroupKey == key {
				g.mu.Unlock()
				group.mu.Lock()
				defer group.mu.Unlock()
				group.total++
				return group.reservoir.Sample(tx.Duration, tx.TraceID), nil
			}
		}
	}
	if g.groupsCount == len(g.groupsSpace) {
		g.mu.Unlock()
		return false, errTooManyTraceGroups
	}
	group := &g.groupsSpace[g.groupsCount]
	group.traceGroupKey = key
	group.samplingFraction = g.defaultSamplingFraction
	group.total = 1
	if group.reservoir == nil {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		group.reservoir = newWeightedRandomSample(rng, minReservoirSize)
	}
	group.reservoir.Sample(tx.Duration, tx.TraceID)
	g.groups[hash] = append(entries, group)
	g.groupsCount++
	g.mu.Unlock()
	return true, nil
}
