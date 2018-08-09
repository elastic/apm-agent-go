package elasticapm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go"
)

func TestTraceID(t *testing.T) {
	var id elasticapm.TraceID
	assert.EqualError(t, id.Validate(), "zero trace-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestSpanID(t *testing.T) {
	var id elasticapm.SpanID
	assert.EqualError(t, id.Validate(), "zero span-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestTraceOptions(t *testing.T) {
	opts := elasticapm.TraceOptions(0xFE)
	assert.False(t, opts.Requested())
	assert.True(t, opts.MaybeRecorded())

	opts = opts.WithRequested(true)
	assert.True(t, opts.Requested())
	assert.True(t, opts.MaybeRecorded())
	assert.Equal(t, elasticapm.TraceOptions(0xFF), opts)

	opts = opts.WithRequested(false)
	assert.False(t, opts.Requested())
	assert.True(t, opts.MaybeRecorded())
	assert.Equal(t, elasticapm.TraceOptions(0xFE), opts)

	opts = opts.WithMaybeRecorded(false)
	assert.False(t, opts.Requested())
	assert.False(t, opts.MaybeRecorded())
	assert.Equal(t, elasticapm.TraceOptions(0xFC), opts)
}
