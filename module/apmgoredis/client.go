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

package apmgoredis // import "go.elastic.co/apm/module/apmgoredis"

import (
	"context"
	"strings"

	"github.com/go-redis/redis"

	"go.elastic.co/apm"
)

// Client is the interface returned by Wrap.
//
// Client implements redis.UniversalClient
type Client interface {
	redis.UniversalClient

	// Client() returns the wrapped *redis.Client,
	// or nil if a non-normal client is wrapped
	RedisClient() *redis.Client

	// ClusterClient returns the wrapped *redis.ClusterClient,
	// or nil if a non-cluster client is wrapped.
	Cluster() *redis.ClusterClient

	// Ring returns the wrapped *redis.Ring,
	// or nil if a non-ring client is wrapped.
	RingClient() *redis.Ring

	// WithContext returns a shallow copy of the client with
	// its context changed to ctx and will add instrumentation
	// with client.WrapProcess and client.WrapProcessPipeline
	//
	// To report commands as spans, ctx must contain a transaction or span.
	WithContext(ctx context.Context) Client
}

// Wrap wraps client such that executed commands are reported as spans to Elastic APM,
// using the client's associated context.
// A context-specific client may be obtained by using Client.WithContext.
func Wrap(client redis.UniversalClient) Client {
	switch client.(type) {
	case *redis.Client:
		return contextClient{Client: client.(*redis.Client)}
	case *redis.ClusterClient:
		return contextClusterClient{ClusterClient: client.(*redis.ClusterClient)}
	case *redis.Ring:
		return contextRingClient{Ring: client.(*redis.Ring)}
	}

	return client.(Client)

}

type contextClient struct {
	*redis.Client
}

func (c contextClient) WithContext(ctx context.Context) Client {
	c.Client = c.Client.WithContext(ctx)

	c.WrapProcess(process(ctx))
	c.WrapProcessPipeline(processPipeline(ctx))

	return c
}

func (c contextClient) RedisClient() *redis.Client {
	return c.Client
}

func (c contextClient) Cluster() *redis.ClusterClient {
	return nil
}

func (c contextClient) RingClient() *redis.Ring {
	return nil
}

type contextClusterClient struct {
	*redis.ClusterClient
}

func (c contextClusterClient) RedisClient() *redis.Client {
	return nil
}

func (c contextClusterClient) Cluster() *redis.ClusterClient {
	return c.ClusterClient
}

func (c contextClusterClient) RingClient() *redis.Ring {
	return nil
}

func (c contextClusterClient) WithContext(ctx context.Context) Client {
	c.ClusterClient = c.ClusterClient.WithContext(ctx)

	c.WrapProcess(process(ctx))
	c.WrapProcessPipeline(processPipeline(ctx))

	return c
}

type contextRingClient struct {
	*redis.Ring
}

func (c contextRingClient) RedisClient() *redis.Client {
	return nil
}

func (c contextRingClient) Cluster() *redis.ClusterClient {
	return nil
}

func (c contextRingClient) RingClient() *redis.Ring {
	return c.Ring
}

func (c contextRingClient) WithContext(ctx context.Context) Client {
	c.Ring = c.Ring.WithContext(ctx)

	c.WrapProcess(process(ctx))
	c.WrapProcessPipeline(processPipeline(ctx))

	return c
}

func process(ctx context.Context) func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
	return func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			spanName := strings.ToUpper(cmd.Name())
			span, _ := apm.StartSpan(ctx, spanName, "db.redis")
			defer span.End()

			return oldProcess(cmd)
		}
	}
}

func processPipeline(ctx context.Context) func(oldProcess func(cmds []redis.Cmder) error) func(cmds []redis.Cmder) error {
	return func(oldProcess func(cmds []redis.Cmder) error) func(cmds []redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			pipelineSpan, ctx := apm.StartSpan(ctx, "(pipeline)", "db.redis")

			for i := len(cmds); i > 0; i-- {
				cmdName := strings.ToUpper(cmds[i-1].Name())
				if cmdName == "" {
					cmdName = "(empty command)"
				}

				span, _ := apm.StartSpan(ctx, cmdName, "db.redis")
				defer span.End()
			}

			defer pipelineSpan.End()

			return oldProcess(cmds)
		}
	}
}
