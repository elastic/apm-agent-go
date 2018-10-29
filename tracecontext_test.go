package apm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm"
)

func TestTraceID(t *testing.T) {
	var id apm.TraceID
	assert.EqualError(t, id.Validate(), "zero trace-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestSpanID(t *testing.T) {
	var id apm.SpanID
	assert.EqualError(t, id.Validate(), "zero span-id is invalid")

	id[0] = 1
	assert.NoError(t, id.Validate())
}

func TestTraceOptions(t *testing.T) {
	opts := apm.TraceOptions(0xFE)
	assert.False(t, opts.Recorded())

	opts = opts.WithRecorded(true)
	assert.True(t, opts.Recorded())
	assert.Equal(t, apm.TraceOptions(0xFF), opts)

	opts = opts.WithRecorded(false)
	assert.False(t, opts.Recorded())
	assert.Equal(t, apm.TraceOptions(0xFE), opts)
}
