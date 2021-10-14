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

//go:build go1.11
// +build go1.11

package apmgoredisv8_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-redis/redis/v8"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	apmgoredis "go.elastic.co/apm/module/apmgoredisv8"
)

const (
	clientTypeBase = iota
	clientTypeCluster
	clientTypeRing
)

var (
	unitTestCases = []struct {
		clientType int
		client     redis.UniversalClient
	}{
		{
			clientTypeBase,
			redisHookedClient(),
		},
		{
			clientTypeCluster,
			redisHookedClusterClient(),
		},
		{
			clientTypeRing,
			redisHookedRing(),
		},
	}
)

func TestHook(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client.Ping(ctx)
				client.Get(ctx, "key")
				client.Do(ctx, "")
			})
			require.Len(t, spans, 3)
			assert.Equal(t, "PING", spans[0].Name)
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "redis", spans[0].Subtype)
			assert.Equal(t, "redis", spans[0].Context.Destination.Service.Resource)
			assert.Equal(t, "GET", spans[1].Name)
			assert.Equal(t, "db", spans[1].Type)
			assert.Equal(t, "redis", spans[1].Subtype)
			assert.Equal(t, "redis", spans[1].Context.Destination.Service.Resource)
			assert.Equal(t, "(empty command)", spans[2].Name)
			assert.Equal(t, "db", spans[2].Type)
			assert.Equal(t, "redis", spans[2].Subtype)
			assert.Equal(t, "redis", spans[2].Context.Destination.Service.Resource)
		})
	}
}

func TestHookPipeline(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				pipe := client.Pipeline()
				pipe.Get(ctx, "key")
				pipe.Set(ctx, "key", "value", 0)
				pipe.Get(ctx, "key")
				pipe.Do(ctx, "")
				_, _ = pipe.Exec(ctx)
			})

			require.Len(t, spans, 1)
			assert.Equal(t, "GET, SET, GET, (empty command)", spans[0].Name)
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "redis", spans[0].Subtype)
			assert.Equal(t, "redis", spans[0].Context.Destination.Service.Resource)
		})
	}
}

func TestHookTxPipeline(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				pipe := client.TxPipeline()
				pipe.Get(ctx, "key")
				pipe.Set(ctx, "key", "value", 0)
				pipe.Get(ctx, "key")
				pipe.Do(ctx, "")
				_, _ = pipe.Exec(ctx)
			})

			require.Len(t, spans, 1)
			if _, ok := client.(*redis.Client); ok {
				// *redis.Client wraps queued commands with MULTI/EXEC
				// in (*redis.Client) processTxPipeline
				assert.Equal(t, "MULTI, GET, SET, GET, (empty command), EXEC", spans[0].Name)
			} else {
				// *redis.ClusterClient and *redis.Ring don't wrap queued commands
				// with MULTI/EXEC in (*redis.Client) processTxPipeline
				assert.Equal(t, "GET, SET, GET, (empty command)", spans[0].Name)
			}
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "redis", spans[0].Subtype)
			assert.Equal(t, "redis", spans[0].Context.Destination.Service.Resource)
		})
	}
}

func redisEmptyClient() *redis.Client {
	return redis.NewClient(&redis.Options{})
}

func redisHookedClient() *redis.Client {
	client := redisEmptyClient()
	client.AddHook(&durationHook{
		duration: 2 * time.Millisecond,
		wrapped:  apmgoredis.NewHook(),
	})
	return client
}

func redisEmptyClusterClient() *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{})
}

func redisHookedClusterClient() *redis.ClusterClient {
	client := redisEmptyClusterClient()
	client.AddHook(&durationHook{
		duration: 2 * time.Millisecond,
		wrapped:  apmgoredis.NewHook(),
	})
	return client
}

func redisEmptyRing() *redis.Ring {
	return redis.NewRing(&redis.RingOptions{})
}

func redisHookedRing() *redis.Ring {
	client := redisEmptyRing()
	client.AddHook(&durationHook{
		duration: 2 * time.Millisecond,
		wrapped:  apmgoredis.NewHook(),
	})
	return client
}

// durationHook decorates the existing hook to avoid the exit spans from being
// dropped.
type durationHook struct {
	wrapped  redis.Hook
	duration time.Duration
}

func (h *durationHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return h.wrapped.BeforeProcess(ctx, cmd)
}
func (h *durationHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	span := apm.SpanFromContext(ctx)
	span.Duration = h.duration
	return h.wrapped.AfterProcess(ctx, cmd)
}

func (h *durationHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return h.wrapped.BeforeProcessPipeline(ctx, cmds)
}
func (h *durationHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	span := apm.SpanFromContext(ctx)
	span.Duration = h.duration
	return h.wrapped.AfterProcessPipeline(ctx, cmds)
}
