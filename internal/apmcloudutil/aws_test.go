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

func TestAWSCloudMetadata(t *testing.T) {
	srv, client := newAWSMetadataServer()
	defer srv.Close()

	for _, provider := range []Provider{Auto, AWS} {
		var out model.Cloud
		var logger testLogger
		assert.True(t, provider.getCloudMetadata(context.Background(), client, &logger, &out))
		assert.Zero(t, logger)
		assert.Equal(t, model.Cloud{
			Provider:         "aws",
			Region:           "us-east-2",
			AvailabilityZone: "us-east-2a",
			Instance: &model.CloudInstance{
				ID: "i-0ae894a7c1c4f2a75",
			},
			Machine: &model.CloudMachine{
				Type: "t2.medium",
			},
			Account: &model.CloudAccount{
				ID: "946960629917",
			},
		}, out)
	}
}

func newAWSMetadataServer() (*httptest.Server, *http.Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			w.Write([]byte("topsecret"))
			return
		case "/latest/dynamic/instance-identity/document":
			token := r.Header.Get("X-Aws-Ec2-Metadata-Token")
			if token != "topsecret" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("invalid token"))
				return
			}
			break
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Write([]byte(`{
    "accountId": "946960629917",
    "architecture": "x86_64",
    "availabilityZone": "us-east-2a",
    "billingProducts": null,
    "devpayProductCodes": null,
    "marketplaceProductCodes": null,
    "imageId": "ami-07c1207a9d40bc3bd",
    "instanceId": "i-0ae894a7c1c4f2a75",
    "instanceType": "t2.medium",
    "kernelId": null,
    "pendingTime": "2020-06-12T17:46:09Z",
    "privateIp": "172.31.0.212",
    "ramdiskId": null,
    "region": "us-east-2",
    "version": "2017-09-30"
}`))
	}))

	client := &http.Client{Transport: newTargetedRoundTripper("169.254.169.254", srv.Listener.Addr().String())}
	return srv, client
}
