package apmlogrus_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmlogrus"
)

func TestTraceContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf)

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		logger.WithTime(time.Unix(0, 0).UTC()).WithFields(apmlogrus.TraceContext(ctx)).Debug("beep")
	})
	require.Len(t, spans, 1)

	assert.Equal(t,
		fmt.Sprintf(
			`{"level":"debug","msg":"beep","span.id":"%x","time":"1970-01-01T00:00:00Z","trace.id":"%x","transaction.id":"%x"}`+"\n",
			spans[0].ID[:], tx.TraceID[:], tx.ID[:],
		),
		buf.String(),
	)
}

func TestTraceContextTextFormatter(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf)
	logger.Formatter = &logrus.TextFormatter{}

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		logger.WithTime(time.Unix(0, 0).UTC()).WithFields(apmlogrus.TraceContext(ctx)).Debug("beep")
	})
	require.Len(t, spans, 1)

	assert.Equal(t,
		fmt.Sprintf(
			"time=\"1970-01-01T00:00:00Z\" level=debug msg=beep span.id=%x trace.id=%x transaction.id=%x\n",
			spans[0].ID[:], tx.TraceID[:], tx.ID[:],
		),
		buf.String(),
	)
}

func TestTraceContextNoSpan(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf)
	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		logger.WithTime(time.Unix(0, 0).UTC()).WithFields(apmlogrus.TraceContext(ctx)).Debug("beep")
	})

	assert.Equal(t,
		fmt.Sprintf(
			`{"level":"debug","msg":"beep","time":"1970-01-01T00:00:00Z","trace.id":"%x","transaction.id":"%x"}`+"\n",
			tx.TraceID[:], tx.ID[:],
		),
		buf.String(),
	)
}

func TestTraceContextEmpty(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf)

	// apmlogrus.TraceContext will return nil if the context does not contain a transaction.
	ctx := context.Background()
	logger.WithTime(time.Unix(0, 0).UTC()).WithFields(apmlogrus.TraceContext(ctx)).Debug("beep")
	assert.Equal(t, `{"level":"debug","msg":"beep","time":"1970-01-01T00:00:00Z"}`+"\n", buf.String())
}
