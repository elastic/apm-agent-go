package trace_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/apm-agent-go/trace"
	"github.com/elastic/apm-agent-go/transport/transporttest"
)

func TestTracerFlushIntervalEnv(t *testing.T) {
	t.Run("suffix", func(t *testing.T) {
		testTracerFlushIntervalEnv(t, "1s", time.Second)
	})
	t.Run("no_suffix", func(t *testing.T) {
		testTracerFlushIntervalEnv(t, "1", time.Second)
	})
}

func TestTracerFlushIntervalEnvInvalid(t *testing.T) {
	os.Setenv("ELASTIC_APM_FLUSH_INTERVAL", "aeon")
	defer os.Unsetenv("ELASTIC_APM_FLUSH_INTERVAL")

	_, err := trace.NewTracer("tracer.testing", "")
	assert.Error(t, err, "failed to parse ELASTIC_APM_FLUSH_INTERVAL: time: invalid duration aeon")
}

func testTracerFlushIntervalEnv(t *testing.T, envValue string, expectedInterval time.Duration) {
	os.Setenv("ELASTIC_APM_FLUSH_INTERVAL", envValue)
	defer os.Unsetenv("ELASTIC_APM_FLUSH_INTERVAL")

	tracer, err := trace.NewTracer("tracer.testing", "")
	require.NoError(t, err)
	defer tracer.Close()
	tracer.Transport = transporttest.Discard

	before := time.Now()
	tracer.StartTransaction("name", "type").Done(-1)
	assert.Equal(t, trace.TracerStats{TransactionsSent: 0}, tracer.Stats())
	for tracer.Stats().TransactionsSent == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.WithinDuration(t, before.Add(expectedInterval), time.Now(), 100*time.Millisecond)
}
