package apmconfig_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/apm-agent-go/internal/apmconfig"
)

func TestParseDurationEnv(t *testing.T) {
	const envKey = "ELASTIC_APM_TEST_DURATION"
	os.Setenv(envKey, "")

	d, err := apmconfig.ParseDurationEnv(envKey, "s", 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 42*time.Second, d)

	os.Setenv(envKey, "5")
	d, err = apmconfig.ParseDurationEnv(envKey, "s", 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, d)

	os.Setenv(envKey, "5ms")
	d, err = apmconfig.ParseDurationEnv(envKey, "s", 42*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Millisecond, d)

	os.Setenv(envKey, "5")
	_, err = apmconfig.ParseDurationEnv(envKey, "", 42*time.Second)
	assert.EqualError(t, err, "failed to parse ELASTIC_APM_TEST_DURATION: time: missing unit in duration 5")
}
