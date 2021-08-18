// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build go1.9
// +build go1.9

package apmzap_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmzap"
)

func TestTraceContext(t *testing.T) {
	var buf zaptest.Buffer
	logger := newLogger(&buf, zap.DebugLevel)

	tx, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		span, ctx := apm.StartSpan(ctx, "name", "type")
		defer span.End()
		logger.With(apmzap.TraceContext(ctx)...).Debug("beep")
		logger.Debug("beep", apmzap.TraceContext(ctx)...)
	})
	require.Len(t, spans, 1)
	lines := buf.Lines()
	require.Len(t, lines, 2)

	for _, line := range lines {
		assert.Equal(t, fmt.Sprintf(
			`{"level":"debug","message":"beep","trace.id":"%x","transaction.id":"%x","span.id":"%x"}`,
			tx.TraceID[:], tx.ID[:], spans[0].ID[:],
		), line)
	}
}

func TestTraceContextNoSpan(t *testing.T) {
	var buf zaptest.Buffer
	logger := newLogger(&buf, zap.DebugLevel)

	tx, _, _ := apmtest.WithTransaction(func(ctx context.Context) {
		logger.Debug("beep", apmzap.TraceContext(ctx)...)
	})
	lines := buf.Lines()
	require.Len(t, lines, 1)

	assert.Equal(t, fmt.Sprintf(
		`{"level":"debug","message":"beep","trace.id":"%x","transaction.id":"%x"}`,
		tx.TraceID[:], tx.ID[:],
	), lines[0])
}

func TestTraceContextEmpty(t *testing.T) {
	var buf zaptest.Buffer
	logger := newLogger(&buf, zap.DebugLevel)

	// apmzap.TraceContext will return nil if the context does not contain a transaction.
	ctx := context.Background()
	logger.Debug("beep", apmzap.TraceContext(ctx)...)

	lines := buf.Lines()
	require.Len(t, lines, 1)
	assert.Equal(t, `{"level":"debug","message":"beep"}`, lines[0])
}

func newLogger(ws zapcore.WriteSyncer, level zapcore.LevelEnabler) *zap.Logger {
	config := zapcore.EncoderConfig{
		LevelKey:       "level",
		NameKey:        "name",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stack",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(config), ws, level))
}
