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

type fileRPC struct {
	accountName  string
	resourceName string
	req          pipeline.Request
}

func (f *fileRPC) name() string {
	return fmt.Sprintf("AzureFile %s %s", f.operation(), f.resourceName)
}

func (f *fileRPC) _type() string {
	return "storage"
}

func (f *fileRPC) subtype() string {
	return "azurefile"
}

func (f *fileRPC) storageAccountName() string {
	return f.accountName
}

func (f *fileRPC) resource() string {
	return f.resourceName
}

func (f *fileRPC) operation() string {
	q := f.req.URL.Query()
	switch f.req.Method {
	case http.MethodOptions:
		return f.optionsOperation()
	case http.MethodDelete:
		return f.deleteOperation()
	// From net/http documentation:
	// For client requests, an empty string means GET.
	case http.MethodGet, "":
		return f.getOperation(q)
	case http.MethodPost:
		return f.postOperation()
	case http.MethodHead:
		return f.headOperation(q)
	case http.MethodPut:
		return f.putOperation(q, f.req.Header)
	default:
		return f.req.Method
	}
}

func (f *fileRPC) deleteOperation() string {
	return "Delete"
}

func (f *fileRPC) optionsOperation() string {
	return "Preflight"
}

func (f *fileRPC) getOperation(v url.Values) string {
	if v.Get("restype") == "share" {
		return "GetProperties"
	}

	switch comp := v.Get("comp"); comp {
	case "":
		return "Download"
	case "listhandles":
		return "ListHandles"
	case "rangelist":
		return "ListRanges"
	case "metadata", "acl":
		return "Get" + strings.Title(comp)
	case "list", "stats":
		return strings.Title(comp)
	default:
		return "unknown operation"
	}
}

func (f *fileRPC) postOperation() string {
	return "unknown operation"

}

func (f *fileRPC) headOperation(v url.Values) string {
	comp := v.Get("comp")
	if v.Get("restype") == "share" || comp == "" {
		return "GetProperties"
	}

	switch comp {
	case "metadata", "acl":
		return "Get" + strings.Title(comp)
	default:
		return "unknown operation"
	}
}

func (f *fileRPC) putOperation(v url.Values, h http.Header) string {
	if _, copySource := h["x-ms-copy-source"]; copySource {
		return "Copy"
	}

	if _, copyAction := h["x-ms-copy-action:abort"]; copyAction {
		return "Abort"
	}
	restype := v.Get("restype")
	if restype == "directory" {
		return "Create"
	}

	switch comp := v.Get("comp"); comp {
	case "range":
		return "Upload"
	case "forceclosehandles":
		return "CloseHandles"
	case "lease", "snapshot", "undelete":
		return strings.Title(comp)
	case "acl", "metadata", "properties":
		return "Set" + strings.Title(comp)
	case "filepermission":
		return "SetPermission"
	default:
		return "unknown operation"
	}
}
