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
	"errors"
	"strings"

	"github.com/Azure/azure-pipeline-go/pipeline"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/Azure/azure-pipeline-go",
		"github.com/Azure/azure-storage-blob-go/azblob",
	)
}

// WrapPipeline wraps the provided pipeline.Pipeline, returning a new one that
// instruments requests and responses.
func WrapPipeline(p pipeline.Pipeline) pipeline.Pipeline {
	return &apmPipeline{p}
}

type apmPipeline struct {
	next pipeline.Pipeline
}

func (p *apmPipeline) Do(
	ctx context.Context,
	methodFactory pipeline.Factory,
	req pipeline.Request,
) (pipeline.Response, error) {
	rpc, err := newAzureRPC(req)
	if err != nil {
		return p.next.Do(ctx, methodFactory, req)
	}

	span, ctx := apm.StartSpan(ctx, rpc.name(), rpc._type())
	defer span.End()
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.Request = apmhttp.RequestWithContext(ctx, req.Request)
		span.Context.SetHTTPRequest(req.Request)
	} else {
		return p.next.Do(ctx, methodFactory, req)
	}
	span.Action = rpc.operation()
	span.Subtype = rpc.subtype()
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Resource: rpc.subtype() + "/" + rpc.storageAccountName(),
	})

	resp, err := p.next.Do(ctx, methodFactory, req)
	if err != nil {
		apm.CaptureError(ctx, err).Send()
	} else {
		span.Context.SetHTTPStatusCode(resp.Response().StatusCode)
	}

	return resp, err
}

type azureRPC interface {
	name() string
	_type() string
	subtype() string
	storageAccountName() string
	resource() string
	operation() string
}

func newAzureRPC(req pipeline.Request) (azureRPC, error) {
	split := strings.Split(req.Host, ".")
	accountName, storage := split[0], split[1]
	var rpc azureRPC
	if storage == "blob" {
		rpc = &blobRPC{
			resourceName: strings.TrimPrefix(req.URL.Path, "/"),
			accountName:  accountName,
			req:          req,
		}
	}
	if rpc == nil {
		return nil, errors.New("unsupported service")
	}

	return rpc, nil
}
