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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

type blobRPC struct {
	accountName  string
	resourceName string
	req          pipeline.Request
}

func (b *blobRPC) name() string {
	return fmt.Sprintf("AzureBlob %s %s", b.operation(), b.resourceName)
}

func (b *blobRPC) _type() string {
	return "storage"
}

func (b *blobRPC) subtype() string {
	return "azureblob"
}

func (b *blobRPC) storageAccountName() string {
	return b.accountName
}

func (b *blobRPC) resource() string {
	return b.resourceName
}

func (b *blobRPC) operation() string {
	if b.req.Method == http.MethodDelete {
		return "Delete"
	}
	q := b.req.URL.Query()
	switch b.req.Method {
	// From net/http documentation:
	// For client requests, an empty string means GET.
	case http.MethodGet, "":
		return getOperation(q)
	case http.MethodPost:
		return postOperation(q)
	case http.MethodHead:
		return headOperation(q)
	case http.MethodPut:
		return putOperation(q, b.req.Header)
	default:
		return b.req.Method
	}
}

func getOperation(v url.Values) string {
	restype := v.Get("restype")
	comp := v.Get("comp")
	if (restype == "" && comp == "") || comp == "blocklist" {
		return "Download"
	}
	if restype == "container" && comp == "" {
		return "GetProperties"
	}

	switch comp {
	case "metadata":
		return "GetMetadata"
	case "acl":
		return "GetAcl"
	case "list":
		if restype == "container" {
			return "ListBlobs"
		}
		return "ListContainers"
	case "tags":
		if v.Get("where") != "" {
			return "FindTags"
		}
		return "GetTags"
	default:
		return "unknown operation"
	}
}

func postOperation(v url.Values) string {
	comp := v.Get("comp")
	switch comp {
	case "batch":
		return "Batch"
	case "query":
		return "Query"
	case "userdelegationkey":
		return "GetUserDelegationKey"
	default:
		return "unknown operation"
	}

}
func headOperation(v url.Values) string {
	restype := v.Get("restype")
	comp := v.Get("comp")
	if restype == "" && comp == "" {
		return "GetProperties"
	}
	if restype != "container" {
		return "unknown operation"
	}
	switch comp {
	case "metadata":
		return "GetMetadata"
	case "acl":
		return "GetAcl"
	default:
		return "unknown operation"
	}
}
func putOperation(v url.Values, h http.Header) string {
	// header.Get canonicalizes the key, ie. x-ms-copy-source->X-Ms-Copy-Source.
	// The headers used are all lowercase, so we access the map directly.
	_, copySource := h["x-ms-copy-source"]
	_, blobType := h["x-ms-blob-type"]
	_, pageWrite := h["x-ms-page-write"]
	restype := v.Get("restype")
	comp := v.Get("comp")
	if restype == "container" && comp == "acl" {
		return "SetAcl"
	}

	if comp == "" && !(copySource || blobType || pageWrite) {
		return "Create"
	}
	if copySource {
		return "Copy"
	}
	if blobType {
		return "Upload"
	}
	if comp == "page" && pageWrite {
		return "Clear"
	}

	switch comp {
	case "block", "blocklist", "page", "appendblock":
		return "Upload"
	case "copy":
		return "Abort"
	case "metadata":
		return "SetMetadata"
	case "lease", "snapshot", "undelete", "seal", "rename":
		return strings.Title(comp)
	case "properties", "tags", "tier", "expiry":
		return "Set" + strings.Title(comp)
	default:
		return "unknown operation"
	}
}
