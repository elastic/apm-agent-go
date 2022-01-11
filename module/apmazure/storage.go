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
		"github.com/Azure/azure-storage-file-go/azfile",
		"github.com/Azure/azure-storage-queue-go/azqueue",
	)
}

// WrapPipeline wraps the provided pipeline.Pipeline, returning a new one that
// instruments requests and responses.
func WrapPipeline(next pipeline.Pipeline, options ...ServerOption) pipeline.Pipeline {
	p := &apmPipeline{next: next}
	for _, opt := range options {
		opt(p)
	}
	if p.tracer == nil {
		p.tracer = apm.DefaultTracer()
	}
	return p
}

// ServerOption sets options for tracing requests.
type ServerOption func(*apmPipeline)

// WithTracer returns a ServerOption which sets t as the tracer
// to use for tracing server requests.
func WithTracer(t *apm.Tracer) ServerOption {
	if t == nil {
		panic("t == nil")
	}

	return func(h *apmPipeline) {
		h.tracer = t
	}
}

type apmPipeline struct {
	next   pipeline.Pipeline
	tracer *apm.Tracer
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

	var span *apm.Span
	if rpc._type() == "messaging" && (req.Method == "GET" || req.Method == "") {
		// A new transaction is created when one or more messages are
		// received from a queue
		tx := p.tracer.StartTransaction(rpc.name(), rpc._type())
		ctx := apm.ContextWithTransaction(req.Context(), tx)
		r := req.Request.WithContext(ctx)
		req.Request = r
		defer tx.End()
		span = tx.StartSpan(rpc.name(), rpc._type(), apm.SpanFromContext(ctx))
	} else {
		span, ctx = apm.StartSpan(ctx, rpc.name(), rpc._type())
	}

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
	}
	// We may still have a response even if err != nil
	// eg., the client library considers 4XX as an error but still returns
	// the response to us.
	if resp.Response() != nil {
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
	switch storage {
	case "blob":
		rpc = &blobRPC{
			resourceName: strings.TrimPrefix(req.URL.Path, "/"),
			accountName:  accountName,
			req:          req,
		}
	case "queue":
		rpc = &queueRPC{
			resourceName: strings.TrimPrefix(req.URL.Path, "/"),
			accountName:  accountName,
			req:          req,
		}
	case "file":
		rpc = &fileRPC{
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
