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

package integration_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmelasticsearch"
)

func TestElastic(t *testing.T) {
	if elasticsearchURL == "" {
		t.Skipf("ELASTICSEARCH_URL not specified")
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		// Addresses set from ELASTICSEARCH_URL
		Transport: apmelasticsearch.WrapRoundTripper(http.DefaultTransport),
	})
	require.NoError(t, err)

	_, spans, errs := apmtest.WithTransaction(func(ctx context.Context) {
		res, err := es.Search(
			es.Search.WithIndex("no_index"),
			es.Search.WithContext(ctx),
			es.Search.WithBody(strings.NewReader(`{"query":{"match_all":{}}}`)),
		)
		require.NoError(t, err)
		res.Body.Close()
	})
	assert.Empty(t, errs)
	require.Len(t, spans, 1)

	esurl, err := url.Parse(elasticsearchURL)
	require.NoError(t, err)
	esurl.Path = "/no_index/_search"

	// We test the value of destination in unit tests.
	require.NotNil(t, spans[0].Context.Destination)
	spans[0].Context.Destination = nil

	assert.Equal(t, "Elasticsearch: GET no_index/_search", spans[0].Name)
	assert.Equal(t, "db", spans[0].Type)
	assert.Equal(t, "elasticsearch", spans[0].Subtype)
	assert.Equal(t, "", spans[0].Action)
	assert.Equal(t, &model.SpanContext{
		Database: &model.DatabaseSpanContext{
			Type:      "elasticsearch",
			Statement: `{"query":{"match_all":{}}}`,
		},
		HTTP: &model.HTTPSpanContext{
			URL:        esurl,
			StatusCode: 404,
		},
	}, spans[0].Context)
}
