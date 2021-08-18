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

package apmgoredis_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-redis/redis"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmgoredis"
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
			redisEmptyClient(),
		},
		{
			clientTypeBase,
			apmgoredis.Wrap(redisEmptyClient()),
		},
		{
			clientTypeBase,
			apmgoredis.Wrap(redisEmptyClient()).WithContext(context.Background()),
		},
		{
			clientTypeCluster,
			redisEmptyClusterClient(),
		},
		{
			clientTypeCluster,
			apmgoredis.Wrap(redisEmptyClusterClient()),
		},
		{
			clientTypeCluster,
			apmgoredis.Wrap(redisEmptyClusterClient()).WithContext(context.Background()),
		},
		{
			clientTypeRing,
			redisEmptyRing(),
		},
		{
			clientTypeRing,
			apmgoredis.Wrap(redisEmptyRing()),
		},
		{
			clientTypeRing,
			apmgoredis.Wrap(redisEmptyRing()).WithContext(context.Background()),
		},
	}
)

func TestWrap(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client).WithContext(ctx)
				client.Ping()
			})
			require.Len(t, spans, 1)
			assert.Equal(t, "PING", spans[0].Name)
			assert.Equal(t, "db", spans[0].Type)
			assert.Equal(t, "redis", spans[0].Subtype)
		})
	}
}

func TestWithContext(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			ping := func(ctx context.Context, client apmgoredis.Client) {
				span, ctx := apm.StartSpan(ctx, "ping", "custom")
				defer span.End()

				// bind client to the ctx containing the span above
				client = client.WithContext(ctx)
				client.Ping()
			}

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client)
				ping(ctx, client)
			})

			require.Len(t, spans, 2)
			assert.Equal(t, "PING", spans[0].Name)
			assert.Equal(t, "ping", spans[1].Name)
			assert.Equal(t, spans[1].ID, spans[0].ParentID)
		})
	}
}

func TestWrapPipeline(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client).WithContext(ctx)
				pipe := client.Pipeline()
				pipe.Do("")
				pipe.Do("")
				pipe.Exec()
			})

			require.Len(t, spans, 3)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "(empty command)", spans[1].Name)
			assert.Equal(t, "(empty command)", spans[2].Name)
		})
	}
}

func TestWrapPipelineTransaction(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			if testCase.clientType == clientTypeRing {
				t.Skipf("redis.Ring doesn't support Transaction")
			}

			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client).WithContext(ctx)
				pipe := client.TxPipeline()
				pipe.Do("")
				pipe.Do("")
				pipe.Exec()
			})

			require.Len(t, spans, 3)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "(empty command)", spans[1].Name)
			assert.Equal(t, "(empty command)", spans[2].Name)
		})
	}
}

func TestWrapPipelined(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client).WithContext(ctx)

				client.Pipelined(func(pipe redis.Pipeliner) error {
					pipe.Set("foo", "bar", 0)
					pipe.Get("foo")
					pipe.FlushDB()

					return nil
				})
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "SET", spans[1].Name)
			assert.Equal(t, "GET", spans[2].Name)
			assert.Equal(t, "FLUSHDB", spans[3].Name)
		})
	}
}

func TestWrapPipelinedTransaction(t *testing.T) {
	for i, testCase := range unitTestCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			if testCase.clientType == clientTypeRing {
				t.Skipf("redis.Ring doesn't support Transaction")
			}

			client := testCase.client

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				client := apmgoredis.Wrap(client).WithContext(ctx)

				client.TxPipelined(func(pipe redis.Pipeliner) error {
					pipe.Set("foo", "bar", 0)
					pipe.Get("foo")
					pipe.FlushDB()

					return nil
				})
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "SET", spans[1].Name)
			assert.Equal(t, "GET", spans[2].Name)
			assert.Equal(t, "FLUSHDB", spans[3].Name)
		})
	}
}

func redisEmptyClient() *redis.Client {
	return redis.NewClient(&redis.Options{})
}

func redisEmptyClusterClient() *redis.ClusterClient {
	return redis.NewClusterClient(&redis.ClusterOptions{})
}

func redisEmptyRing() *redis.Ring {
	return redis.NewRing(&redis.RingOptions{})
}
