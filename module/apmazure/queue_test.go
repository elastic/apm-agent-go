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

//go:build go1.14
// +build go1.14

package apmazure // import "go.elastic.co/apm/module/apmazure"

import (
	"context"
	"net/url"
	"testing"

	"github.com/Azure/azure-storage-queue-go/azqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
)

func TestQueueSend(t *testing.T) {
	retry := azqueue.RetryOptions{
		MaxTries: 1,
	}
	po := azqueue.PipelineOptions{
		Retry: retry,
	}
	p := azqueue.NewPipeline(azqueue.NewAnonymousCredential(), po)
	p = WrapPipeline(p)
	u, err := url.Parse("https://fakeaccnt.queue.core.windows.net")
	require.NoError(t, err)
	queueURL := azqueue.NewQueueURL(*u, p)
	msgURL := queueURL.NewMessagesURL()

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		msgURL.Enqueue(ctx, "new message", 0, 0)
	})
	require.Len(t, errors, 1)
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "messaging", span.Type)
	assert.Equal(t, "AzureQueue SEND to fakeaccnt", span.Name)
	assert.Equal(t, "azurequeue", span.Subtype)
	assert.Equal(t, "SEND", span.Action)
	destination := span.Context.Destination
	assert.Equal(t, "fakeaccnt.queue.core.windows.net", destination.Address)
	assert.Equal(t, 443, destination.Port)
	assert.Equal(t, "azurequeue/fakeaccnt", destination.Service.Resource)
}

func TestQueueReceive(t *testing.T) {
	tracer, transport := transporttest.NewRecorderTracer()
	defer tracer.Close()
	retry := azqueue.RetryOptions{
		MaxTries: 1,
	}
	po := azqueue.PipelineOptions{
		Retry: retry,
	}
	p := azqueue.NewPipeline(azqueue.NewAnonymousCredential(), po)
	p = WrapPipeline(p, WithTracer(tracer))
	u, err := url.Parse("https://fakeaccnt.queue.core.windows.net")
	require.NoError(t, err)
	queueURL := azqueue.NewQueueURL(*u, p)
	msgURL := queueURL.NewMessagesURL()

	msgURL.Peek(context.Background(), 32)
	tracer.Flush(nil)

	payloads := transport.Payloads()
	transaction := payloads.Transactions[0]
	// ParentID is empty, a new transaction was created
	assert.Equal(t, model.SpanID{}, transaction.ParentID)
	assert.Equal(t, "AzureQueue PEEK from fakeaccnt", transaction.Name)
	assert.Equal(t, "messaging", transaction.Type)

	span := payloads.Spans[0]
	assert.Equal(t, "messaging", span.Type)
	assert.Equal(t, "AzureQueue PEEK from fakeaccnt", span.Name)
	assert.Equal(t, "azurequeue", span.Subtype)
	assert.Equal(t, "PEEK", span.Action)
	destination := span.Context.Destination
	assert.Equal(t, "fakeaccnt.queue.core.windows.net", destination.Address)
	assert.Equal(t, 443, destination.Port)
	assert.Equal(t, "azurequeue/fakeaccnt", destination.Service.Resource)
}

func TestQueueGetOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations-1
		{
			want:   "RECEIVE",
			values: url.Values{},
		},
		{
			want:   "PEEK",
			values: url.Values{"peekonly": []string{"true"}},
		},
		{
			want:   "LISTQUEUES",
			values: url.Values{"comp": []string{"list"}},
		},
		{
			want:   "GETPROPERTIES",
			values: url.Values{"comp": []string{"properties"}},
		},
		{
			want:   "STATS",
			values: url.Values{"comp": []string{"stats"}},
		},
		{
			want:   "GETMETADATA",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "GETACL",
			values: url.Values{"comp": []string{"acl"}},
		},
	}

	q := new(queueRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.getOperation(tc.values))
	}
}

func TestQueueHeadOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations-1
		{
			want:   "GETMETADATA",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "GETACL",
			values: url.Values{"comp": []string{"acl"}},
		},
	}

	q := new(queueRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.headOperation(tc.values))
	}
}

func TestQueuePostOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations-1
		{
			want:   "SEND",
			values: url.Values{},
		},
	}

	q := new(queueRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.postOperation(tc.values))
	}
}

func TestQueuePutOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations-1
		{
			want:   "SETMETADATA",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "SETACL",
			values: url.Values{"comp": []string{"acl"}},
		},
		{
			want:   "SETPROPERTIES",
			values: url.Values{"comp": []string{"properties"}},
		},
		{
			want:   "UPDATE",
			values: url.Values{"popreceipt": []string{"value"}},
		},
		{
			want:   "CREATE",
			values: url.Values{},
		},
	}

	q := new(queueRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.putOperation(tc.values))
	}
}
