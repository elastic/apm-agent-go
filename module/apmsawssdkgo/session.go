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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"bytes"
	"io"
	"net/http/httputil"
	"os"
	"strings"

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

// WrapSession wraps the provided s3 session with handlers that hook into the
// aws sdk's request lifecycle.
func WrapSession(s *session.Session) *session.Session {
	s.Handlers.Send.PushFrontNamed(request.NamedHandler{
		Name: "go.elastic.co/apm/module/apmawssdkgo/send",
		Fn:   send,
	})
	s.Handlers.Complete.PushBackNamed(request.NamedHandler{
		Name: "go.elastic.co/apm/module/apmawssdkgo/complete",
		Fn:   complete,
	})

	return s
}

const (
	spanType    = "storage"
	spanSubtype = "s3"
)

func send(req *request.Request) {
	if req.RetryCount > 0 {
		return
	}

	var (
		ctx    = req.Context()
		region = *req.Config.Region
	)

	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return
	}

	bucketName := getBucketName(req)
	byts, err := httputil.DumpRequestOut(req.HTTPRequest, true)
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, bytes.NewBuffer(byts))

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

func complete(req *request.Request) {
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

func getBucketName(req *request.Request) string {
	strings.Split(req.HTTPRequest.URL.Path[1:], "/")[0]
}
