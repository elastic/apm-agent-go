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
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/armcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/armstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/apmtest"
)

func TestQueueSend(t *testing.T) {
	cred := new(tokenCredential)

	opts := &armcore.ConnectionOptions{
		HTTPClient: new(fakeTransport),
	}
	conn := NewConnection("https://storage-account-name.queue.core.windows.net", cred, opts)
	client := armstorage.NewQueueClient(conn, "subscription-id")

	_, spans, errors := apmtest.WithTransaction(func(ctx context.Context) {
		client.Create(
			ctx,
			"resource-group-name",
			"storage-account-name",
			"queue-name",
			armstorage.StorageQueue{},
			new(armstorage.QueueCreateOptions),
		)
	})
	require.Len(t, errors, 0)
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "messaging", span.Type)
	assert.Equal(t, "AzureQueue SEND to queue-name", span.Name)
	assert.Equal(t, 400, span.Context.HTTP.StatusCode)
	assert.Equal(t, "azurequeue", span.Subtype)
	assert.Equal(t, "SEND", span.Action)
	destination := span.Context.Destination
	assert.Equal(t, "storage-account-name.queue.core.windows.net", destination.Address)
	assert.Equal(t, 443, destination.Port)
	assert.Equal(t, "azurequeue/storage-account-name", destination.Service.Resource)
	// Aren't these deprecated???
	assert.Equal(t, "azurequeue", destination.Service.Name)
	assert.Equal(t, "messaging", destination.Service.Type)
}

// TODO
func TestQueueReceive(t *testing.T) {
	t.Skip()
}
