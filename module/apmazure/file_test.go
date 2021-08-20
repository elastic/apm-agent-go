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

package apmazure

import (
	"context"
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
