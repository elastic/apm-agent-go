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
	"os"
	"strings"
	"testing"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmgoredis"
)

func TestMain(m *testing.M) {
	ret := m.Run()

	clusterReset()

	os.Exit(ret)
}

func TestRequestContext(t *testing.T) {
	testCases := getTestCases(t)

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client
			if client == nil {
				t.Errorf("cannot create client")
			}

			defer client.Close()
			cleanRedis(t, client, testCase.clientType)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				for i := 0; i < 2; i++ {
					wrappedClient := apmgoredis.Wrap(client).WithContext(ctx)

					cmd := wrappedClient.Get("content")
					if cmd.Err() == nil {
						return
					}

					value := []byte("Lorem ipsum dolor sit amet")
					if cmd := wrappedClient.Set("content", value, 0); cmd.Err() != nil {
						require.NoError(t, cmd.Err())
					}
				}
			})

			require.Len(t, spans, 3)

			assert.Equal(t, "GET", spans[0].Name)
			assert.Equal(t, "SET", spans[1].Name)
			assert.Equal(t, "GET", spans[2].Name)
		})
	}
}

func TestPipelined(t *testing.T) {
	testCases := getTestCases(t)

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client
			if client == nil {
				t.Errorf("cannot create client")
			}

			defer client.Close()
			cleanRedis(t, client, testCase.clientType)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				wrappedClient := apmgoredis.Wrap(client).WithContext(ctx)

				_, err := wrappedClient.Pipelined(func(pipe redis.Pipeliner) error {
					err := pipe.Set("foo", "bar", 0).Err()
					require.NoError(t, err)

					err = pipe.Get("foo").Err()
					require.NoError(t, err)

					err = pipe.FlushDB().Err()
					require.NoError(t, err)

					return err
				})

				require.NoError(t, err)
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "SET", spans[1].Name)
			assert.Equal(t, "GET", spans[2].Name)
			assert.Equal(t, "FLUSHDB", spans[3].Name)
		})
	}
}

func TestPipeline(t *testing.T) {
	testCases := getTestCases(t)

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			client := testCase.client
			if client == nil {
				t.Errorf("cannot create client")
			}

			defer client.Close()
			cleanRedis(t, client, testCase.clientType)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				wrappedClient := apmgoredis.Wrap(client).WithContext(ctx)

				pipe := wrappedClient.Pipeline()

				err := pipe.Set("foo", "bar", 0).Err()
				require.NoError(t, err)

				err = pipe.Get("foo").Err()
				require.NoError(t, err)

				err = pipe.FlushDB().Err()
				require.NoError(t, err)

				_, err = pipe.Exec()
				require.NoError(t, err)
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "SET", spans[1].Name)
			assert.Equal(t, "GET", spans[2].Name)
			assert.Equal(t, "FLUSHDB", spans[3].Name)
		})
	}
}

func TestPipelinedTransaction(t *testing.T) {
	testCases := getTestCases(t)

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			if testCase.clientType == clientTypeRing {
				t.Skipf("redis.Ring doesn't support Transaction")
			}

			client := testCase.client
			if client == nil {
				t.Errorf("cannot create client")
			}

			defer client.Close()
			cleanRedis(t, client, testCase.clientType)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				wrappedClient := apmgoredis.Wrap(client).WithContext(ctx)

				var incr1 *redis.IntCmd
				var incr2 *redis.IntCmd
				var incr3 *redis.IntCmd
				_, err := wrappedClient.TxPipelined(func(pipe redis.Pipeliner) error {
					incr1 = pipe.Incr("foo")
					assert.NoError(t, incr1.Err())

					incr2 = pipe.Incr("bar")
					assert.NoError(t, incr2.Err())

					incr3 = pipe.Incr("bar")
					assert.NoError(t, incr3.Err())

					return nil
				})

				assert.Equal(t, int64(1), incr1.Val())
				assert.Equal(t, int64(1), incr2.Val())
				assert.Equal(t, int64(2), incr3.Val())

				assert.NoError(t, err)
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "INCR", spans[1].Name)
			assert.Equal(t, "INCR", spans[2].Name)
			assert.Equal(t, "INCR", spans[3].Name)
		})
	}
}

func TestPipelineTransaction(t *testing.T) {
	testCases := getTestCases(t)

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			if testCase.clientType == clientTypeRing {
				t.Skipf("redis.Ring doesn't support Transaction")
			}

			client := testCase.client
			if client == nil {
				t.Errorf("cannot create client")
			}

			defer client.Close()
			cleanRedis(t, client, testCase.clientType)

			_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
				wrappedClient := apmgoredis.Wrap(client).WithContext(ctx)

				pipe := wrappedClient.TxPipeline()

				incr1 := pipe.Incr("foo")
				assert.NoError(t, incr1.Err())

				incr2 := pipe.Incr("bar")
				assert.NoError(t, incr2.Err())

				incr3 := pipe.Incr("bar")
				assert.NoError(t, incr3.Err())

				_, err := pipe.Exec()
				require.NoError(t, err)

				assert.Equal(t, int64(1), incr1.Val())
				assert.Equal(t, int64(1), incr2.Val())
				assert.Equal(t, int64(2), incr3.Val())
			})

			require.Len(t, spans, 4)
			assert.Equal(t, "(pipeline)", spans[0].Name)
			assert.Equal(t, "INCR", spans[1].Name)
			assert.Equal(t, "INCR", spans[2].Name)
			assert.Equal(t, "INCR", spans[3].Name)
		})
	}
}

func redisClient(t *testing.T) *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		t.Skipf("REDIS_URL not specified")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr: opt.Addr,
	})

	return client
}

func redisRingClient(t *testing.T) *redis.Ring {
	redisURLs := strings.Split(os.Getenv("REDIS_RING_URLS"), " ")
	if len(redisURLs) == 0 {
		if t != nil {
			t.Skipf("REDIS_RING_URLS not specified")
		}
	}

	optRedisURLs := make(map[string]string, len(redisURLs))
	for _, redisURL := range redisURLs {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil
		}

		optRedisURLs[opt.Addr] = opt.Addr
	}

	client := redis.NewRing(&redis.RingOptions{
		Addrs: optRedisURLs,
	})

	return client
}

func redisClusterClient(t *testing.T) *redis.ClusterClient {
	redisURLs := strings.Split(os.Getenv("REDIS_CLUSTER_URLS"), " ")
	if len(redisURLs) == 0 {
		if t != nil {
			t.Skipf("REDIS_CLUSTER_URLS not specified")
		}
	}

	for i, redisURL := range redisURLs {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil
		}

		redisURLs[i] = opt.Addr
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: redisURLs,
	})

	return client
}

func clusterReset() {
	client := redisClusterClient(nil)

	if client == nil {
		return
	}

	client.ForEachMaster(func(master *redis.Client) error {
		master.FlushDB().Err()
		master.ClusterResetSoft().Err()

		return nil
	})
}

func cleanRedis(t *testing.T, client redis.UniversalClient, clientType int) {
	switch clientType {
	case clientTypeBase:
		st := client.FlushDB()
		require.NoError(t, st.Err())
	case clientTypeCluster:
		var err error
		switch client.(type) {
		case *redis.ClusterClient:
			err = client.(*redis.ClusterClient).ForEachMaster(func(master *redis.Client) error {
				return master.FlushDB().Err()
			})
		case apmgoredis.Client:
			err = client.(apmgoredis.Client).Cluster().ForEachMaster(func(master *redis.Client) error {
				return master.FlushDB().Err()
			})
		}

		require.NoError(t, err)
	case clientTypeRing:
		var err error
		switch client.(type) {
		case *redis.Ring:
			err = client.(*redis.Ring).ForEachShard(func(shard *redis.Client) error {
				return shard.FlushDB().Err()
			})
		case apmgoredis.Client:
			err = client.(apmgoredis.Client).RingClient().ForEachShard(func(shard *redis.Client) error {
				return shard.FlushDB().Err()
			})
		}

		require.NoError(t, err)
	}
}

func getTestCases(t *testing.T) []struct {
	clientType int
	client     redis.UniversalClient
} {
	return []struct {
		clientType int
		client     redis.UniversalClient
	}{
		{
			clientTypeBase,
			redisClient(t),
		},
		{
			clientTypeBase,
			apmgoredis.Wrap(redisClient(t)),
		},
		{
			clientTypeBase,
			apmgoredis.Wrap(redisClient(t)).WithContext(context.Background()),
		},
		{
			clientTypeCluster,
			redisClusterClient(t),
		},
		{
			clientTypeCluster,
			apmgoredis.Wrap(redisClusterClient(t)),
		},
		{
			clientTypeCluster,
			apmgoredis.Wrap(redisClusterClient(t)).WithContext(context.Background()),
		},
		{
			clientTypeRing,
			redisRingClient(t),
		},
		{
			clientTypeRing,
			apmgoredis.Wrap(redisRingClient(t)),
		},
		{
			clientTypeRing,
			apmgoredis.Wrap(redisRingClient(t)).WithContext(context.Background()),
		},
	}
}
