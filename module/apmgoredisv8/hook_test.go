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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-redis/redis/v8"

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
			assert.Equal(t, "GET", spans[1].Name)
			assert.Equal(t, "db", spans[1].Type)
			assert.Equal(t, "redis", spans[1].Subtype)
			assert.Equal(t, "(empty command)", spans[2].Name)
			assert.Equal(t, "db", spans[2].Type)
			assert.Equal(t, "redis", spans[2].Subtype)
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
		})
	}
}

func redisEmptyClient() *redis.Client {
	return redis.NewClient(&redis.Options{})
}

func redisHookedClient() *redis.Client {
	client := redisEmptyClient()
	client.AddHook(apmgoredis.NewHook())
	return client
}

func redisEmptyClusterClient() *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{})
}

func redisHookedClusterClient() *redis.ClusterClient {
	client := redisEmptyClusterClient()
	client.AddHook(apmgoredis.NewHook())
	return client
}

func redisEmptyRing() *redis.Ring {
	return redis.NewRing(&redis.RingOptions{})
}

func redisHookedRing() *redis.Ring {
	client := redisEmptyRing()
	client.AddHook(apmgoredis.NewHook())
	return client
}
