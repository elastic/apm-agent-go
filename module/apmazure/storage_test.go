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

// +build go1.14

package apmazure

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/armcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/armstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
)

func TestBlob(t *testing.T) {
	cred := new(tokenCredential)

	opts := &armcore.ConnectionOptions{
		HTTPClient: new(fakeTransport),
	}
	conn := NewConnection("https://storage-account-name.blob.core.windows.net", cred, opts)
	client := armstorage.NewBlobContainersClient(conn, "subscription-id")

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		client.Create(
			ctx,
			"resource-group-name",
			"storage-account-name",
			"container-name",
			armstorage.BlobContainer{},
			new(armstorage.BlobContainersCreateOptions),
		)
	})
	require.Len(t, errors, 0)
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "storage", span.Type)
	assert.Equal(t, "AzureBlob Create container-name", span.Name)
	assert.Equal(t, 400, span.Context.HTTP.StatusCode)
	assert.Equal(t, "azureblob", span.Subtype)
	assert.Equal(t, "Create", span.Action)
	destination := span.Context.Destination
	assert.Equal(t, "storage-account-name.blob.core.windows.net", destination.Address)
	assert.Equal(t, 443, destination.Port)
	assert.Equal(t, "azureblob/storage-account-name", destination.Service.Resource)
	// Aren't these deprecated???
	assert.Equal(t, "azureblob", destination.Service.Name)
	assert.Equal(t, "storage", destination.Service.Type)
}

func TestGetOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations
		{
			want:   "Download",
			values: url.Values{},
		},
		{
			want:   "Download",
			values: url.Values{"comp": []string{"blocklist"}},
		},
		{
			want:   "GetProperties",
			values: url.Values{"restype": []string{"container"}},
		},
		{
			want:   "GetMetadata",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "GetAcl",
			values: url.Values{"restype": []string{"container"}, "comp": []string{"acl"}},
		},
		{
			want:   "ListBlobs",
			values: url.Values{"restype": []string{"container"}, "comp": []string{"list"}},
		},
		{
			want:   "ListContainers",
			values: url.Values{"comp": []string{"list"}},
		},
		{
			want:   "GetTags",
			values: url.Values{"comp": []string{"tags"}},
		},
		{
			want:   "FindTags",
			values: url.Values{"comp": []string{"tags"}, "where": []string{"value"}},
		},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.want, getOperation(tc.values))
	}
}

func TestHeadOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations
		{
			want:   "GetProperties",
			values: url.Values{},
		},
		{
			want:   "GetMetadata",
			values: url.Values{"restype": []string{"container"}, "comp": []string{"metadata"}},
		},
		{
			want:   "GetAcl",
			values: url.Values{"restype": []string{"container"}, "comp": []string{"acl"}},
		},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.want, headOperation(tc.values))
	}
}

func TestPostOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations
		{
			want:   "unknown operation",
			values: url.Values{},
		},
		{
			want:   "Batch",
			values: url.Values{"comp": []string{"batch"}},
		},
		{
			want:   "Query",
			values: url.Values{"comp": []string{"query"}},
		},
		{
			want:   "GetUserDelegationKey",
			values: url.Values{"comp": []string{"userdelegationkey"}},
		},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.want, postOperation(tc.values))
	}
}

func TestPutOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
		header http.Header
	}{
		// https://github.com/elastic/apm/blob/master/specs/agents/tracing-instrumentation-azure.md#determining-operations
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
		},
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
			values: url.Values{"comp": []string{"block"}},
		},
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
			values: url.Values{"comp": []string{"page"}},
		},
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
			values: url.Values{"comp": []string{"incrementalcopy"}},
		},
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
			values: url.Values{"comp": []string{"appendblock"}},
		},
		{
			want:   "Abort",
			values: url.Values{"comp": []string{"copy"}},
		},
		{
			want:   "Upload",
			header: http.Header{"x-ms-blob-type": []string{"BlockBlob"}},
		},
		{
			want:   "Upload",
			values: url.Values{"comp": []string{"block"}},
		},
		{
			want:   "Upload",
			values: url.Values{"comp": []string{"blocklist"}},
		},
		{
			want:   "Upload",
			values: url.Values{"comp": []string{"page"}},
		},
		{
			want:   "Upload",
			values: url.Values{"comp": []string{"appendblock"}},
		},
		{
			want:   "Create",
			header: http.Header{},
			values: url.Values{},
		},
		{
			want:   "SetMetadata",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "SetAcl",
			values: url.Values{"restype": []string{"container"}, "comp": []string{"acl"}},
		},
		{
			want:   "Lease",
			values: url.Values{"comp": []string{"lease"}},
		},
		{
			want:   "Snapshot",
			values: url.Values{"comp": []string{"snapshot"}},
		},
		{
			want:   "Undelete",
			values: url.Values{"comp": []string{"undelete"}},
		},
		{
			want:   "Seal",
			values: url.Values{"comp": []string{"seal"}},
		},
		{
			want:   "Rename",
			values: url.Values{"comp": []string{"rename"}},
		},
		{
			want:   "SetProperties",
			values: url.Values{"comp": []string{"properties"}},
		},
		{
			want:   "SetTags",
			values: url.Values{"comp": []string{"tags"}},
		},
		{
			want:   "SetTier",
			values: url.Values{"comp": []string{"tier"}},
		},
		{
			want:   "SetExpiry",
			values: url.Values{"comp": []string{"expiry"}},
		},
		{
			want:   "Clear",
			header: http.Header{"x-ms-page-write": []string{}},
			values: url.Values{"comp": []string{"page"}},
		},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.want, putOperation(tc.values, tc.header))
	}
}

type tokenCredential struct{}

func (t *tokenCredential) GetToken(_ context.Context, _ azcore.TokenRequestOptions) (*azcore.AccessToken, error) {
	return nil, nil
}

func (t *tokenCredential) AuthenticationPolicy(options azcore.AuthenticationPolicyOptions) azcore.Policy {
	return new(fakePolicy)
}

type fakePolicy struct{}

func (p *fakePolicy) Do(req *azcore.Request) (*azcore.Response, error) {
	resp, err := req.Next()
	return resp, err
}

type fakeTransport struct{}

func (t *fakeTransport) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       &fakeBuffer{new(bytes.Buffer)},
	}, nil
}

type fakeBuffer struct{ *bytes.Buffer }

func (b *fakeBuffer) Close() error { return nil }
