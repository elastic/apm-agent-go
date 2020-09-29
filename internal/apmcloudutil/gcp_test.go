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

func TestGCPCloudMetadata(t *testing.T) {
	t.Run("gce", func(t *testing.T) {
		srv, client := newGCEMetadataServer()
		defer srv.Close()

		for _, provider := range []Provider{Auto, GCP} {
			var out model.Cloud
			var logger testLogger
			assert.True(t, provider.getCloudMetadata(context.Background(), client, &logger, &out))
			assert.Zero(t, logger)
			assert.Equal(t, model.Cloud{
				Provider:         "gcp",
				Region:           "us-west3",
				AvailabilityZone: "us-west3-a",
				Instance: &model.CloudInstance{
					ID:   "4306570268266786072",
					Name: "basepi-test",
				},
				Machine: &model.CloudMachine{
					Type: "n1-standard-1",
				},
				Project: &model.CloudProject{
					ID:   "513326162531",
					Name: "elastic-apm",
				},
			}, out)
		}
	})

	t.Run("cloudrun", func(t *testing.T) {
		srv, client := newGoogleCloudRunMetadataServer()
		defer srv.Close()

		var out model.Cloud
		var logger testLogger
		assert.True(t, GCP.getCloudMetadata(context.Background(), client, &logger, &out))
		assert.Zero(t, logger)
		assert.Equal(t, model.Cloud{
			Provider:         "gcp",
			Region:           "australia-southeast1",
			AvailabilityZone: "australia-southeast1-1",
			Instance: &model.CloudInstance{
				ID: "00bf4bf02ddbda278fb9b4d70365018bd18a7d3ea42991e2cb03320b48a72b69b6d3765ff526347d7b8e0934dda4591cb1be3ead93086f0b390187fae88ee7cf8acdae7383",
			},
			Project: &model.CloudProject{
				ID:   "513326162531",
				Name: "elastic-apm",
			},
		}, out)
	})
}

func newGCEMetadataServer() (*httptest.Server, *http.Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/computeMetadata/v1/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{
    "instance": {
        "id": 4306570268266786072,
        "machineType": "projects/513326162531/machineTypes/n1-standard-1",
        "name": "basepi-test",
        "zone": "projects/513326162531/zones/us-west3-a"
    },
    "project": {"numericProjectId": 513326162531, "projectId": "elastic-apm"}
}`))
	}))

	client := &http.Client{Transport: newTargetedRoundTripper("metadata.google.internal", srv.Listener.Addr().String())}
	return srv, client
}

func newGoogleCloudRunMetadataServer() (*httptest.Server, *http.Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/computeMetadata/v1/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{
    "instance": {
        "id": "00bf4bf02ddbda278fb9b4d70365018bd18a7d3ea42991e2cb03320b48a72b69b6d3765ff526347d7b8e0934dda4591cb1be3ead93086f0b390187fae88ee7cf8acdae7383",
        "region":"projects/513326162531/regions/australia-southeast1",
        "zone":"projects/513326162531/zones/australia-southeast1-1"
    },
    "project": {
        "numericProjectId": 513326162531,
        "projectId": "elastic-apm"
    }
}`))
	}))

	client := &http.Client{Transport: newTargetedRoundTripper("metadata.google.internal", srv.Listener.Addr().String())}
	return srv, client
}
