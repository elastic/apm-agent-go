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

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo/v2"

import (
	"context"

	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/stacktrace"

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
	s.Handlers.Build.PushFrontNamed(request.NamedHandler{
		Name: "go.elastic.co/apm/module/apmawssdkgo/build",
		Fn:   build,
	})
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

// We add AWS spans to context using a separate context key, to avoid
// modifying spans not created by the "build" handler.
type awsSpanKey struct{}

const (
	serviceS3       = "s3"
	serviceDynamoDB = "dynamodb"
	serviceSQS      = "sqs"
	serviceSNS      = "sns"
)

var (
	serviceTypeMap = map[string]string{
		serviceS3:       "storage",
		serviceDynamoDB: "db",
		serviceSQS:      "messaging",
		serviceSNS:      "messaging",
	}
)

type service interface {
	spanName() string
	resource() string
	targetName() string
	setAdditional(*apm.Span)
}

func build(req *request.Request) {
	spanSubtype := req.ClientInfo.ServiceName
	spanType, ok := serviceTypeMap[spanSubtype]
	if !ok {
		// Not a supported service.
		return
	}
	if spanSubtype == serviceSNS && !supportedSNSMethod(req) {
		return
	}
	if spanSubtype == serviceSQS && !supportedSQSMethod(req) {
		return
	}

	ctx := req.Context()
	tx := apm.TransactionFromContext(ctx)
	if tx == nil {
		return
	}

	// The span name is added in the `send()` function, after other
	// handlers have generated the necessary information on the request.
	span := tx.StartExitSpan("", spanType, apm.SpanFromContext(ctx))
	if span.Dropped() {
		span.End()
		return
	}

	switch spanSubtype {
	case serviceSQS:
		addMessageAttributesSQS(req, span, tx.ShouldPropagateLegacyHeader())
	case serviceSNS:
		addMessageAttributesSNS(req, span, tx.ShouldPropagateLegacyHeader())
	}

	ctx = apm.ContextWithSpan(ctx, span)
	ctx = context.WithValue(ctx, awsSpanKey{}, span)
	req.SetContext(ctx)
}

func send(req *request.Request) {
	if req.RetryCount > 0 {
		return
	}

	span, ok := req.Context().Value(awsSpanKey{}).(*apm.Span)
	if !ok {
		return
	}

	var (
		svc service
		err error
	)
	spanSubtype := req.ClientInfo.ServiceName
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
	case serviceSNS:
		if svc, err = newSNS(req); err != nil {
			// Unsupported method type or queue name.
			return
		}
	default:
		// Unsupported type
		return
	}

	span.Name = svc.spanName()
	span.Subtype = spanSubtype
	span.Action = req.Operation.Name

	span.Context.SetHTTPRequest(req.HTTPRequest)

	span.Context.SetDestinationService(apm.DestinationServiceSpanContext{
		Name:     spanSubtype,
		Resource: svc.resource(),
	})
	span.Context.SetServiceTarget(apm.ServiceTargetSpanContext{
		Type: spanSubtype,
		Name: svc.targetName(),
	})

	if region := req.Config.Region; region != nil {
		span.Context.SetDestinationCloud(apm.DestinationCloudSpanContext{
			Region: *region,
		})
	}

	svc.setAdditional(span)
}

func complete(req *request.Request) {
	ctx := req.Context()
	span, ok := ctx.Value(awsSpanKey{}).(*apm.Span)
	if !ok {
		return
	}
	defer span.End()

	span.Context.SetHTTPStatusCode(req.HTTPResponse.StatusCode)

	if err := req.Error; err != nil {
		apm.CaptureError(ctx, err).Send()
	}
}
