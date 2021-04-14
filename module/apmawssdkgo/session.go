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
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/stacktrace"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/aws/aws-sdk-go",
	)
}

// WrapSession wraps the provided AWS session with handlers that hook into the
// AWS SDK's request lifecycle. Supported services are listed in serviceTypeMap
// variable below.
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
	serviceS3       = "s3"
	serviceDynamoDB = "dynamodb"
	serviceSQS      = "sqs"
)

var (
	serviceTypeMap = map[string]string{
		serviceS3:       "storage",
		serviceDynamoDB: "db",
		serviceSQS:      "messaging",
	}
)

type service interface {
	spanName() string
	resource() string
	setAdditional(*apm.Span)
}

func send(req *request.Request) {
	if req.RetryCount > 0 {
		return
	}

	spanSubtype := req.ClientInfo.ServiceName
	spanType, ok := serviceTypeMap[spanSubtype]
	if !ok {
		return
	}

	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return
	}

	var (
		svc service
		err error
	)
	switch spanSubtype {
	case serviceS3:
		svc = newS3(req)
	case serviceDynamoDB:
		svc = newDynamoDB(req)
	case serviceSQS:
		if svc, err = newSQS(req); err != nil {
			// Unsupported method type or queue name.
			return
		}
	default:
		// Unsupported type
		return
	}

	span := tx.StartSpan(svc.spanName(), spanType, apm.SpanFromContext(ctx))
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.HTTPRequest = apmhttp.RequestWithContext(ctx, req.HTTPRequest)
		span.Context.SetHTTPRequest(req.HTTPRequest)
	} else {
		span.End()
		span = nil
		return
	}

	span.Subtype = spanSubtype
	span.Action = req.Operation.Name

	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     spanSubtype,
		Resource: svc.resource(),
	})

	if region := req.Config.Region; region != nil {
		span.Context.SetDestinationCloud(apm.DestinationCloudSpanContext{
			Region: *region,
		})
	}

	svc.setAdditional(span)

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
