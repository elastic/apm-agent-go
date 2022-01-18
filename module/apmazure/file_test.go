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

package apmazure

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
)

func TestFile(t *testing.T) {
	retry := azfile.RetryOptions{
		MaxTries: 1,
	}
	po := azfile.PipelineOptions{
		Retry: retry,
	}
	p := azfile.NewPipeline(azfile.NewAnonymousCredential(), po)
	p = WrapPipeline(p)
	u, err := url.Parse("https://fakeaccnt.file.core.windows.net")
	require.NoError(t, err)
	serviceURL := azfile.NewServiceURL(*u, p)
	shareURL := serviceURL.NewShareURL("share")
	dirURL := shareURL.NewDirectoryURL("dir")
	fileURL := dirURL.NewFileURL("file")

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		fileURL.Download(ctx, 0, 0, false)
	})
	require.Len(t, errors, 1)
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "storage", span.Type)
	assert.Equal(t, "AzureFile Download share/dir/file", span.Name)
	assert.Equal(t, "azurefile", span.Subtype)
	assert.Equal(t, "Download", span.Action)
	destination := span.Context.Destination
	assert.Equal(t, "fakeaccnt.file.core.windows.net", destination.Address)
	assert.Equal(t, 443, destination.Port)
	assert.Equal(t, "azurefile/fakeaccnt", destination.Service.Resource)
}

func TestFileGetOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/main/specs/agents/tracing-instrumentation-azure.md#determining-operations-3
		{
			want:   "Download",
			values: url.Values{},
		},
		{
			want:   "GetProperties",
			values: url.Values{"restype": []string{"share"}},
		},
		{
			want:   "ListHandles",
			values: url.Values{"comp": []string{"listhandles"}},
		},
		{
			want:   "ListRanges",
			values: url.Values{"comp": []string{"rangelist"}},
		},
		{
			want:   "Stats",
			values: url.Values{"comp": []string{"stats"}},
		},
		{
			want:   "List",
			values: url.Values{"comp": []string{"list"}},
		},
		{
			want:   "GetMetadata",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "GetAcl",
			values: url.Values{"comp": []string{"acl"}},
		},
	}

	q := new(fileRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.getOperation(tc.values))
	}
}

func TestFilePostOperation(t *testing.T) {
	q := new(fileRPC)
	assert.Equal(t, "unknown operation", q.postOperation())
}

func TestFileHeadOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
	}{
		// https://github.com/elastic/apm/blob/main/specs/agents/tracing-instrumentation-azure.md#determining-operations-3
		{
			want:   "GetProperties",
			values: url.Values{},
		},
		{
			want:   "GetProperties",
			values: url.Values{"restype": []string{"share"}},
		},
		{
			want:   "GetMetadata",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "GetAcl",
			values: url.Values{"comp": []string{"acl"}},
		},
	}

	q := new(fileRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, q.headOperation(tc.values))
	}
}

func TestFilePutOperation(t *testing.T) {
	tcs := []struct {
		want   string
		values url.Values
		header http.Header
	}{
		// https://github.com/elastic/apm/blob/main/specs/agents/tracing-instrumentation-azure.md#determining-operations
		{
			want:   "Copy",
			header: http.Header{"x-ms-copy-source": []string{}},
		},
		{
			want:   "Abort",
			header: http.Header{"x-ms-copy-action:abort": []string{}},
		},
		{
			want:   "Create",
			values: url.Values{"restype": []string{"directory"}},
		},
		{
			want:   "Upload",
			values: url.Values{"comp": []string{"range"}},
		},
		{
			want:   "CloseHandles",
			values: url.Values{"comp": []string{"forceclosehandles"}},
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
			want:   "SetAcl",
			values: url.Values{"comp": []string{"acl"}},
		},
		{
			want:   "SetPermission",
			values: url.Values{"comp": []string{"filepermission"}},
		},
		{
			want:   "SetMetadata",
			values: url.Values{"comp": []string{"metadata"}},
		},
		{
			want:   "SetProperties",
			values: url.Values{"comp": []string{"properties"}},
		},
	}

	f := new(fileRPC)
	for _, tc := range tcs {
		assert.Equal(t, tc.want, f.putOperation(tc.values, tc.header))
	}
}

func TestFileOptionsOperation(t *testing.T) {
	f := new(fileRPC)
	assert.Equal(t, "Preflight", f.optionsOperation())
}

func TestFileDeleteOperation(t *testing.T) {
	f := new(fileRPC)
	assert.Equal(t, "Delete", f.deleteOperation())
}
