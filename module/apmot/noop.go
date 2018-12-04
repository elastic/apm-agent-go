package apmot

import (
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type unsupportedSpanMethods struct{}

// BaggageItem returns the empty string; we do not support baggage.
func (unsupportedSpanMethods) BaggageItem(key string) string {
	return ""
}

func (unsupportedSpanMethods) LogKV(keyValues ...interface{}) {
	// No-op.
}

func (unsupportedSpanMethods) LogFields(fields ...log.Field) {
	// No-op.
}

func (unsupportedSpanMethods) LogEvent(event string) {
	// No-op.
}

func (unsupportedSpanMethods) LogEventWithPayload(event string, payload interface{}) {
	// No-op.
}

func (unsupportedSpanMethods) Log(ld opentracing.LogData) {
	// No-op.
}
