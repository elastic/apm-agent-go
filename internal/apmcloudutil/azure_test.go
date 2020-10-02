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

package apmcloudutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.elastic.co/apm/model"
)

func TestAzureCloudMetadata(t *testing.T) {
	srv, client := newAzureMetadataServer()
	defer srv.Close()

	for _, provider := range []Provider{Auto, Azure} {
		var out model.Cloud
		var logger testLogger
		assert.True(t, provider.getCloudMetadata(context.Background(), client, &logger, &out))
		assert.Zero(t, logger)
		assert.Equal(t, model.Cloud{
			Provider: "azure",
			Region:   "westus2",
			Instance: &model.CloudInstance{
				ID:   "e11ebedc-019d-427f-84dd-56cd4388d3a8",
				Name: "basepi-test",
			},
			Machine: &model.CloudMachine{
				Type: "Standard_D2s_v3",
			},
			Project: &model.CloudProject{
				Name: "basepi-testing",
			},
			Account: &model.CloudAccount{
				ID: "7657426d-c4c3-44ac-88a2-3b2cd59e6dba",
			},
		}, out)
	}
}

func newAzureMetadataServer() (*httptest.Server, *http.Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata/instance/compute" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{
    "location": "westus2",
    "name": "basepi-test",
    "resourceGroupName": "basepi-testing",
    "subscriptionId": "7657426d-c4c3-44ac-88a2-3b2cd59e6dba",
    "vmId": "e11ebedc-019d-427f-84dd-56cd4388d3a8",
    "vmScaleSetName": "",
    "vmSize": "Standard_D2s_v3",
    "zone": ""
}`))
	}))

	client := &http.Client{Transport: newTargetedRoundTripper("169.254.169.254", srv.Listener.Addr().String())}
	return srv, client
}
