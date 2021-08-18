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

//go:build go1.10
// +build go1.10

package integration_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/olivere/elastic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmelasticsearch"
)

var elasticsearchURL = os.Getenv("ELASTICSEARCH_URL")

func TestOlivereElastic(t *testing.T) {
	if elasticsearchURL == "" {
		t.Skipf("ELASTICSEARCH_URL not specified")
	}

	client, err := elastic.NewClient(
		elastic.SetURL(elasticsearchURL),
		elastic.SetHttpClient(&http.Client{
			Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
		}),
	)
	require.NoError(t, err)

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		client.Search("no_index").Query(elastic.NewMatchAllQuery()).Do(ctx)
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	esurl, err := url.Parse(elasticsearchURL)
	require.NoError(t, err)
	esurl.Path = "/no_index/_search"
	nodesInfo, err := client.NodesInfo().Do(context.Background())
	require.NoError(t, err)
	require.Len(t, nodesInfo.Nodes, 1)
	for _, node := range nodesInfo.Nodes {
		esurl.Host = node.HTTP.PublishAddress
	}

	addr, portstr, err := net.SplitHostPort(esurl.Host)
	require.NoError(t, err)
	port, err := strconv.Atoi(portstr)
	require.NoError(t, err)

	assert.Equal(t, "Elasticsearch: POST no_index/_search", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "elasticsearch", spans[0].Subtype)
	assert.Equal(t, "", spans[0].Action)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "elasticsearch",
			Statement: `{"query":{"match_all":{}}}`,
		},
		Destination: &model.DestinationSpanContext{
			Address: addr,
			Port:    port,
			Service: &model.DestinationServiceSpanContext{
				Type:     "db",
				Name:     "elasticsearch",
				Resource: "elasticsearch",
			},
		},
		HTTP: &model.HTTPSpanContext{
			URL:        esurl,
			StatusCode: 404,
		},
	}, spans[0].Context)
}
