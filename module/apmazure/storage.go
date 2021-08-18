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
	"errors"
	"fmt"
	"net"
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
	if err != nil {
		return req.Next()
	}

	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return req.Next()
	}

	span := tx.StartSpan(rpc.name(), rpc._type(), apm.SpanFromContext(ctx))
	defer span.End()
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.Request = apmhttp.RequestWithContext(ctx, req.Request)
		span.Context.SetHTTPRequest(req.Request)
	} else {
		return req.Next()
	}
	span.Action = rpc.operation()
	span.Subtype = rpc.subtype()
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     rpc.subtype(),
		Resource: rpc.subtype() + "/" + rpc.storageAccountName(),
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

type azureRPC interface {
	name() string
	_type() string
	subtype() string
	storageAccountName() string
	storage() string
	operation() string
}

func newAzureRPC(req *azcore.Request) (azureRPC, error) {
	u := req.URL

	m := make(map[string]string)
	// Remove initial /
	split := strings.Split(u.Path[1:], "/")
	if len(split)%2 != 0 {
		return nil, fmt.Errorf("unexpected path: %s", u.Path)
	}
	for i := 0; i < len(split); i += 2 {
		fmt.Printf("%s: %s\n", split[i], split[i+1])
		m[split[i]] = split[i+1]
	}
	var rpc azureRPC
	if _, ok := m["blobServices"]; ok {
		rpc = &blobRPC{
			storageName: m["containers"],
			accountName: m["storageAccounts"],
			req:         req,
		}
	}
	if _, ok := m["queueServices"]; ok {
		rpc = &queueRPC{
			storageName: m["queues"],
			accountName: m["storageAccounts"],
			req:         req,
		}
	}
	if rpc == nil {
		return nil, errors.New("unsupported service")
	}

	return rpc, nil
}
