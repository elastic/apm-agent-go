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

package apmgoredis

import (
	"context"
	"strings"

	"github.com/go-redis/redis"

	"go.elastic.co/apm"
)

// Client is the interface returned by ContextClient.
//
// Client implements redis.UniversalClient
type Client interface {
	redis.UniversalClient

	// ClusterClient returns the eventually embedded redis.ClusterClient or nil
	Cluster() *redis.ClusterClient

	// WithContext returns a shallow copy of the client with
	// its context changed to ctx and will add instrumentation
	// with client.WrapProcess and client.WrapProcessPipeline
	//
	// To report commands as spans, ctx must contain a transaction or span.
	WithContext(ctx context.Context) Client
}

// Wrap according to client type. The context can be changed using Client.WithContext.
// That will add instrumentation with client.WrapProcess and client.WrapProcessPipeline
func Wrap(client redis.UniversalClient) Client {
	switch client.(type) {
	case contextClient:
		return client.(contextClient)
	case contextClusterClient:
		return client.(contextClusterClient)
	case *redis.Client:
		return contextClient{Client: client.(*redis.Client)}
	case *redis.ClusterClient:
		return contextClusterClient{ClusterClient: client.(*redis.ClusterClient)}
	}

	return nil
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

func (c contextClient) Cluster() *redis.ClusterClient {
	return nil
}

type contextClusterClient struct {
	*redis.ClusterClient
}

func (c contextClusterClient) Cluster() *redis.ClusterClient {
	return c.ClusterClient
}

func (c contextClusterClient) WithContext(ctx context.Context) Client {
	c.ClusterClient = c.ClusterClient.WithContext(ctx)

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
			cmdNames := make([]string, len(cmds))
			for i, cmd := range cmds {
				cmdName := strings.ToUpper(cmd.Name())
				if cmdName == "" {
					cmdName = "(flush pipeline)"
				}

				cmdNames[i] = cmdName
			}

			spanName := "(pipeline) " + strings.Join(cmdNames, " ")
			span, _ := apm.StartSpan(ctx, spanName, "db.redis")
			defer span.End()

			return oldProcess(cmds)
		}
	}
}
