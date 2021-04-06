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

package apms3session // import "go.elastic.co/apm/module/apms3session"

import (
	"fmt"
	"reflect"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/aws/aws-sdk-go/aws/request",
		"github.com/aws/aws-sdk-go/aws/session",
	)
}

type handlers struct {
	tracer *apm.Tracer
}

// Option sets options for tracing s3 requests.
type Option func(*handlers)

// WithTracer returns an Option which sets t as the tracer to use for tracing
// s3 requests.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(h *handlers) {
		h.tracer = t
	}
}

// WrapSession wraps the provided s3 session with handlers that hook into the
// aws sdk's request lifecycle.
func WrapSession(s *session.Session, opts ...Option) *session.Session {
	h := &handlers{
		tracer: apm.DefaultTracer,
	}
	for _, o := range opts {
		o(h)
	}

	s.Handlers.Send.PushFrontNamed(request.NamedHandler{
		Name: "go.elastic.co/apm/module/apms3session/send",
		Fn:   h.send,
	})
	s.Handlers.Complete.PushBackNamed(request.NamedHandler{
		Name: "go.elastic.co/apm/module/apms3session/complete",
		Fn:   h.complete,
	})

	return s
}

const (
	spanType    = "storage"
	spanSubtype = "s3"
)

func (h *handlers) send(req *request.Request) {
	if req.RetryCount > 0 {
		return
	}

	var (
		bucketName string

		ctx    = req.Context()
		region = *req.Config.Region
	)

	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return
	}

	// TODO: All Params structs are required to have a `Bucket` attr. If
	// there's a better way to get this, I would be overjoyed to avoid
	// using runtime reflection.
	params := reflect.ValueOf(req.Params).Elem()
	if n, ok := params.FieldByName("Bucket").Interface().(*string); ok {
		bucketName = *n
	} else {
		// TODO: How do we want to handle this?
		fmt.Printf("couldn't reflect Bucket into string %+v\n", req.Params)
	}

	spanName := spanSubtype + " " + req.Operation.Name + " " + bucketName
	span := tx.StartSpan(spanName, spanType, apm.SpanFromContext(ctx))
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.HTTPRequest = apmhttp.RequestWithContext(ctx, req.HTTPRequest)
		span.Context.SetHTTPRequest(req.HTTPRequest)
	} else {
		span.End()
		span = nil
	}

	span.Subtype = spanSubtype
	span.Action = req.Operation.Name

	destinationAddress := req.ClientInfo.Endpoint
	span.Context.SetDestinationAddress(destinationAddress, 0)
	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     spanSubtype,
		Resource: bucketName,
		Type:     spanType,
	})
	span.Context.SetDestinationCloud(apm.DestinationCloudSpanContext{
		Region: region,
	})

	req.SetContext(ctx)
}

func (h *handlers) complete(req *request.Request) {
	ctx := req.Context()
	span := apm.SpanFromContext(ctx)
	if span.Dropped() {
		return
	}
	defer span.End()

	span.Context.SetHTTPStatusCode(req.HTTPResponse.StatusCode)

	if err := req.Error; err != nil {
		apm.CaptureError(ctx, err).Send()
	}
}
