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

package apmelasticsearch // import "go.elastic.co/apm/module/apmelasticsearch/v2"

import (
	"net/http"
)

// ClusterNameFunc is a function for fetching the Elasticsearch
// cluster name for an HTTP request.
type ClusterNameFunc func(*http.Response) string

// WithClusterName returns a ClientOption which sets f as the
// function to use to obtain the Elasticsearch cluster name for
// recording on spans.
func WithClusterName(f ClusterNameFunc) ClientOption {
	if f == nil {
		panic("f == nil")
	}
	return func(rt *roundTripper) {
		rt.clusterNameFunc = f
	}
}

// DefaultClusterName returns the default ClusterNameFunc implementation
// used by WrapRoundTripper, if WithClusterName is not specified.
//
// DefaultClusterName looks for the X-Found-Handling-Cluster response
// header, using that if received; this is set by Elastic Cloud to the target
// cluster name.
// with its own cached host-to-cluster-name mappings.
func DefaultClusterName(rt http.RoundTripper) ClusterNameFunc {
	return func(resp *http.Response) string {
		// Elastic Cloud will add the cluster name in response headers.
		return resp.Header.Get("X-Found-Handling-Cluster")
	}
}
