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
	"container/list"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
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
// DefaultClusterName first looks for the X-Found-Handling-Cluster response
// header, using that if received; this is set by Elastic Cloud to the target
// cluster name. If the response header is not received, DefaultClusterName
// will make a query to "/_nodes/http" using the supplied RoundTripper, and
// cache the result by the request URL's Host.
//
// Each invocation of DefaultClusterName will return a distinct ClusterNameFunc
// with its own cached host-to-cluster-name mappings.
func DefaultClusterName(rt http.RoundTripper) ClusterNameFunc {
	client := &http.Client{Transport: rt}
	queryClusterName := func(
		ctx context.Context,
		requestURL *url.URL,
		header http.Header,
	) (string, error) {
		// Query /_nodes/http. This request is expected to fail for
		// clients that are not authorized.
		u := url.URL{
			Scheme: requestURL.Scheme,
			User:   requestURL.User,
			Host:   requestURL.Host,
			Path:   "/_nodes/http",
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", err
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var result struct {
			ClusterName string `json:"cluster_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}
		return result.ClusterName, nil
	}

	var mu sync.RWMutex
	lru := newStringsLRU(100)
	return func(resp *http.Response) string {
		// Elastic Cloud will add the cluster name in response headers.
		essHeader := resp.Header.Get("X-Found-Handling-Cluster")
		if essHeader != "" {
			return essHeader
		}

		host := resp.Request.Host
		if host == "" {
			host = resp.Request.URL.Host
		}
		mu.RLock()
		clusterName, ok := lru.get(host)
		mu.RUnlock()
		if ok {
			return clusterName
		}

		clusterName, err := queryClusterName(
			resp.Request.Context(),
			resp.Request.URL,
			resp.Request.Header,
		)
		if err != nil {
			// Sniffing the cluster name may fail when the
			// client is not authorized for cluster monitoring
			// or management. In this case we can't supply a
			// cluster name.
			clusterName = ""
		}

		// Cache the result by host. We cache the empty string on error
		// to avoid repeatedly querying for the cluster name.
		mu.Lock()
		lru.set(host, clusterName)
		mu.Unlock()
		return clusterName
	}
}

type stringsLRU struct {
	m    map[string]*list.Element
	size int
	list *list.List
}

type lruKeyValue struct {
	key   string
	value string
}

func newStringsLRU(size int) *stringsLRU {
	return &stringsLRU{
		m:    make(map[string]*list.Element, size),
		size: size,
		list: list.New(),
	}
}

func (lru *stringsLRU) get(k string) (string, bool) {
	kv, ok := lru.getKeyValue(k)
	if !ok {
		return "", false
	}
	return kv.value, true
}

func (lru *stringsLRU) set(k, v string) {
	kv, ok := lru.getKeyValue(k)
	if ok {
		kv.value = v
		return
	}

	if lru.list.Len() < lru.size {
		element := lru.list.PushFront(&lruKeyValue{
			key: k, value: v,
		})
		lru.m[k] = element
		return
	}

	// Replace least recently used element.
	element := lru.list.Back()
	kv = element.Value.(*lruKeyValue)
	delete(lru.m, kv.key)
	kv.key = k
	kv.value = v
	lru.m[k] = element
	lru.list.MoveToFront(element)
}

func (lru *stringsLRU) getKeyValue(k string) (*lruKeyValue, bool) {
	element, ok := lru.m[k]
	if !ok {
		return nil, false
	}
	lru.list.MoveToFront(element)
	return element.Value.(*lruKeyValue), true
}
