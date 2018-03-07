package trace

import (
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

const (
	envFlushInterval = "ELASTIC_APM_FLUSH_INTERVAL"
	envMaxQueueSize  = "ELASTIC_APM_MAX_QUEUE_SIZE"
	envMaxSpans      = "ELASTIC_APM_TRANSACTION_MAX_SPANS"

	defaultFlushInterval = 10 * time.Second
	defaultMaxQueueSize  = 500
	defaultMaxSpans      = 500
)

func initialFlushInterval() (time.Duration, error) {
	value := os.Getenv(envFlushInterval)
	if value == "" {
		return defaultFlushInterval, nil
	}
	// TODO(axw) check what format other agents accept
	// in ELASTIC_APM_FLUSH_INTERVAL. Ideally we should
	// accept a common format.
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envFlushInterval)
	}
	return d, nil
}

func initialMaxTransactionQueueSize() (int, error) {
	value := os.Getenv(envMaxQueueSize)
	if value == "" {
		return defaultMaxQueueSize, nil
	}
	size, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envMaxQueueSize)
	}
	return size, nil
}

func initialMaxSpans() (int, error) {
	value := os.Getenv(envMaxSpans)
	if value == "" {
		return defaultMaxSpans, nil
	}
	max, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse %s", envMaxSpans)
	}
	return max, nil
}
