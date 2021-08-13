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

package apmazure // import "go.elastic.co/apm/module/apmazure"

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"

	"github.com/Azure/azure-sdk-for-go/sdk/armcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/Azure/azure-sdk-for-go",
	)
}

// NewConnection returns a new *armcore.Connection whose Pipeline is using an
// http.Client that has been wrapped by apmhttp.WrapClient.
func NewConnection(endpoint string, cred azcore.TokenCredential, options *armcore.ConnectionOptions) *armcore.Connection {
	if options == nil {
		options = &armcore.ConnectionOptions{}
	} else {
		// create a copy so we don't modify the original
		cp := *options
		options = &cp
	}
	options.PerCallPolicies = append(options.PerCallPolicies, new(policy))
	return armcore.NewConnection(endpoint, cred, options)
}

type policy struct{}

func (p *policy) Do(req *azcore.Request) (*azcore.Response, error) {
	rpc, err := newAzureRPC(req)
	if err != nil || !rpc.supported() {
		return req.Next()
	}

	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return req.Next()
	}

	span := tx.StartSpan(rpc.name(), rpc._type, apm.SpanFromContext(ctx))
	defer span.End()
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.Request = apmhttp.RequestWithContext(ctx, req.Request)
		span.Context.SetHTTPRequest(req.Request)
	} else {
		return req.Next()
	}
	span.Action = rpc.operation
	span.Subtype = rpc.subtype
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     rpc.subtype,
		Resource: rpc.resource(),
	})

	if host, port, err := net.SplitHostPort(req.URL.Host); err == nil {
		// strconv.Atoi returns 0 if err != nil
		// TODO: How do we want to handle an error?
		p, _ := strconv.Atoi(port)
		span.Context.SetDestinationAddress(host, p)
	}

	resp, err := req.Next()
	if err != nil {
		apm.CaptureError(ctx, err).Send()
	}

	span.Context.SetHTTPStatusCode(resp.StatusCode)

	return resp, err
}

type azureRPC struct {
	_type              string
	subtype            string
	storageAccountName string
	container          string
	operation          string
}

func newAzureRPC(req *azcore.Request) (*azureRPC, error) {
	u := req.URL

	m := make(map[string]string)
	// Remove initial /
	split := strings.Split(u.Path[1:], "/")
	if len(split)%2 != 0 {
		return nil, fmt.Errorf("unexpected path: %s", u.Path)
	}
	for i := 0; i < len(split); i += 2 {
		m[split[i]] = split[i+1]
	}
	rpc := new(azureRPC)
	if p, ok := m["providers"]; ok {
		// TODO(stn): Check that providers all follow this pattern
		// p == "Microsoft.Storage"
		rpc._type = strings.ToLower(p[10:])
	}
	if _, ok := m["blobServices"]; ok {
		rpc.subtype = "azureblob"
	}
	if p, ok := m["storageAccounts"]; ok {
		rpc.storageAccountName = p
	}
	if p, ok := m["containers"]; ok {
		rpc.container = p
	}
	rpc.operation = operation(req)

	return rpc, nil
}

func (a *azureRPC) supported() bool {
	return a._type == "storage" && a.subtype == "azureblob"
}

func (a *azureRPC) resource() string {
	return a.subtype + "/" + a.storageAccountName
}

func (a *azureRPC) name() string {
	var n string
	switch a.subtype {
	case "azureblob":
		n = "AzureBlob"
	}
	return fmt.Sprintf("%s %s %s", n, a.operation, a.container)
}

func operation(req *azcore.Request) string {
	if req.Method == http.MethodDelete {
		return "Delete"
	}
	q := req.URL.Query()
	switch req.Method {
	// From net/http documentation:
	// For client requests, an empty string means GET.
	case http.MethodGet, "":
		return getOperation(q)
	case http.MethodPost:
		return postOperation(q)
	case http.MethodHead:
		return headOperation(q)
	case http.MethodPut:
		return putOperation(q, req.Header)
	default:
		return req.Method
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

	// TODO: Figure out Create
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
