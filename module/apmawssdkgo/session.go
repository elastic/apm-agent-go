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

// +build go1.13

package apmawssdkgo // import "go.elastic.co/apm/module/apmawssdkgo"

import (
	"strconv"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/module/apmprometheus"
	"go.elastic.co/apm/stacktrace"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	stacktrace.RegisterLibraryPackage(
		"github.com/aws/aws-sdk-go",
	)
}

var (
	inflight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "apmawssdkgo",
		Name:      "in_flight_requests",
		Help:      "A gauge of in-flight AWS RPC calls, partitioned by service and rpc.",
	},
		[]string{"service", "rpc"},
	)
	receivedBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "apmawssdkgo",
			Name:      "http_received_bytes_total",
			Help:      "A counter for the total requests bytes sent to AWS, partitioned by service, rpc, and code.",
		},
		[]string{"service", "rpc", "code"},
	)
	sentBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "apmawssdkgo",
			Name:      "http_sent_bytes_total",
			Help:      "A counter for the total response bytes read from AWS, partitioned by service, rpc, and code.",
		},
		[]string{"service", "rpc", "code"},
	)
)

// WrapSession wraps the provided AWS session with handlers that hook into the
// AWS SDK's request lifecycle. Supported services are listed in serviceTypeMap
// variable below.
func WrapSession(s *session.Session, opts ...Option) *session.Session {
	cfg := &config{
		tracer:   apm.DefaultTracer,
		registry: prometheus.NewRegistry(),
	}

	for _, o := range opts {
		o(cfg)
	}

	// Record our metrics using client_golang, and have apmprometheus
	// convert them for us.
	cfg.registry.MustRegister(inflight, receivedBytes, sentBytes)
	cfg.tracer.RegisterMetricsGatherer(apmprometheus.Wrap(cfg.registry))

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

type config struct {
	tracer   *apm.Tracer
	registry *prometheus.Registry
}

// Option sets options for tracing server requests.
type Option func(*config)

// WithTracer returns a Option which sets t as the tracer to use for tracing
// requests made with the AWS SDK.
func WithTracer(t *apm.Tracer) Option {
	if t == nil {
		panic("t == nil")
	}
	return func(c *config) {
		c.tracer = t
	}
}

// WithRegistry returns an Option which sets r as the prometheus.Registry to
// use when recording AWS SDK metrics. If no registry is supplied, a new one
// will be instantiated.Metrics will be reported to the APM server regardless
// of the prometheus.Registry supplied.
func WithRegistry(r *prometheus.Registry) Option {
	if r == nil {
		panic("r == nil")
	}
	return func(c *config) {
		c.registry = r
	}
}

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
	setAdditional(*apm.Span)
}

func build(req *request.Request) {
	spanSubtype := req.ClientInfo.ServiceName
	spanType, ok := serviceTypeMap[spanSubtype]
	if !ok {
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
	span := tx.StartSpan("", spanType, apm.SpanFromContext(ctx))
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		defer req.SetContext(ctx)
	} else {
		span.End()
		span = nil
		return
	}

	switch spanSubtype {
	case serviceSQS:
		addMessageAttributesSQS(req, span, tx.ShouldPropagateLegacyHeader())
	case serviceSNS:
		addMessageAttributesSNS(req, span, tx.ShouldPropagateLegacyHeader())
	default:
		return
	}
}

func send(req *request.Request) {
	if req.RetryCount > 0 {
		return
	}

	spanSubtype := req.ClientInfo.ServiceName
	_, ok := serviceTypeMap[spanSubtype]
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
	case serviceSNS:
		if svc, err = newSNS(req); err != nil {
			// Unsupported method type or queue name.
			return
		}
	default:
		// Unsupported type
		return
	}

	span := apm.SpanFromContext(ctx)
	if !span.Dropped() {
		ctx = apm.ContextWithSpan(ctx, span)
		req.HTTPRequest = apmhttp.RequestWithContext(ctx, req.HTTPRequest)
		span.Context.SetHTTPRequest(req.HTTPRequest)
	} else {
		span.End()
		span = nil
		return
	}

	span.Name = svc.spanName()
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
	inflight.WithLabelValues(spanSubtype, req.Operation.Name).Inc()
}

func complete(req *request.Request) {
	inflight.WithLabelValues(req.ClientInfo.ServiceName, req.Operation.Name).Dec()

	ctx := req.Context()
	span := apm.SpanFromContext(ctx)
	if span.Dropped() {
		return
	}
	defer span.End()

	code := strconv.Itoa(req.HTTPResponse.StatusCode)
	service := req.ClientInfo.ServiceName
	operationName := req.Operation.Name

	sentBytes.WithLabelValues(
		service,
		operationName,
		code,
	).Add(float64(requestSize(req)))

	receivedBytes.WithLabelValues(
		service,
		operationName,
		code,
	).Add(float64(responseSize(req)))

	span.Context.SetHTTPStatusCode(req.HTTPResponse.StatusCode)

	if err := req.Error; err != nil {
		apm.CaptureError(ctx, err).Send()
	}
}

func requestSize(req *request.Request) int {
	var (
		r = req.HTTPRequest
		s = 0
	)
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

func responseSize(req *request.Request) int {
	var (
		r = req.HTTPResponse
		s = 0
	)

	s += len(r.Status)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
