package apmgoredisv9_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apmgoredis "go.elastic.co/apm/module/apmgoredisv9/v2"
	"go.elastic.co/apm/v2/apmtest"
)

const (
	clientTypeBase = iota
	clientTypeCluster
	clientTypeRing
)

var unitTestCases = []struct {
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
			assert.Equal(t, "MULTI, GET, SET, GET, (empty command), EXEC", spans[0].Name)
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
