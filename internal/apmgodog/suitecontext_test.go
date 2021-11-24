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

//go:build go1.9
// +build go1.9

package apmgodog_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"go.elastic.co/apm"
	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/transport"
)

var (
	apiKey      string
	secretToken string
	env         []string // for subprocesses

	httpServer         *httptest.Server
	httpHandler        = &mockHTTPHandler{}
	httpRequestHeaders http.Header

	grpcServer  *grpc.Server
	grpcClient  *grpc.ClientConn
	grpcService *helloworldGRPCService

	tracer      = apmtest.NewRecordingTracer()
	span        *apm.Span
	transaction *apm.Transaction

	cloud *model.Cloud
)

// InitScenario initialises sc with step definitions.
func InitScenario(sc *godog.ScenarioContext) {
	httpServer = httptest.NewServer(
		apmhttp.Wrap(httpHandler, apmhttp.WithTracer(tracer.Tracer)),
	)

	grpcServer = grpc.NewServer(grpc.UnaryInterceptor(
		apmgrpc.NewUnaryServerInterceptor(
			apmgrpc.WithTracer(tracer.Tracer),
			apmgrpc.WithRecovery(),
		),
	))
	grpcService = &helloworldGRPCService{}
	pb.RegisterGreeterServer(grpcServer, grpcService)
	grpcListener := bufconn.Listen(1)

	var err error
	grpcClient, err = grpc.Dial("bufconn",
		grpc.WithInsecure(),
		grpc.WithDialer(func(string, time.Duration) (net.Conn, error) { return grpcListener.Dial() }),
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
	)
	if err != nil {
		panic(err)
	}
	go grpcServer.Serve(grpcListener)

	sc.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		apiKey = ""
		secretToken = ""
		span = nil
		transaction = nil
		httpRequestHeaders = nil
		httpHandler.panic = false
		httpHandler.statusCode = http.StatusOK
		grpcService.panic = false
		grpcService.err = nil
		tracer.ResetPayloads()
		for _, k := range os.Environ() {
			if strings.HasPrefix(k, "ELASTIC_APM") {
				os.Unsetenv(k)
			}
		}
		return ctx, nil
	})

	sc.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		env = nil
		cloud = nil
		return ctx, nil
	})

	sc.Step("^an agent$", anAgent)
	sc.Step("^an agent configured with$", anAgentConfiguredWith)
	sc.Step("^the agent sends a request to APM server$", sendRequest)
	sc.Step("^the Authorization header of the request is '(.*)'$", checkAuthorizationHeader)

	sc.Step("^an active span$", anActiveSpan)
	sc.Step("^an active transaction$", anActiveTransaction)

	// Outcome
	sc.Step("^a user sets the span outcome to '(.*)'$", userSetsSpanOutcome)
	sc.Step("^a user sets the transaction outcome to '(.*)'$", userSetsTransactionOutcome)
	sc.Step("^the agent sets the span outcome to '(.*)'$", agentSetsSpanOutcome)
	sc.Step("^the agent sets the transaction outcome to '(.*)'$", agentSetsTransactionOutcome)
	sc.Step("^an error is reported to the span$", anErrorIsReportedToTheSpan)
	sc.Step("^an error is reported to the transaction$", anErrorIsReportedToTheTransaction)
	sc.Step("^the span ends$", spanEnds)
	sc.Step("^the transaction ends$", transactionEnds)
	sc.Step("^the span outcome is '(.*)'$", spanOutcomeIs)
	sc.Step("^the transaction outcome is '(.*)'$", transactionOutcomeIs)

	// HTTP
	sc.Step("^a HTTP call is received that returns (.*)$", anHTTPTransactionWithStatusCode)
	sc.Step("^a HTTP call is made that returns (.*)$", anHTTPSpanWithStatusCode)

	// gRPC
	sc.Step("^a gRPC call is received that returns '(.*)'$", aGRPCTransactionWithStatusCode)
	sc.Step("^a gRPC call is made that returns '(.*)'$", aGRPCSpanWithStatusCode)

	// Cloud metadata
	sc.Step("an instrumented application is configured to collect cloud provider metadata for azure", func() error {
		return nil
	})
	sc.Step("the following environment variables are present", func(kv *godog.Table) error {
		for _, row := range kv.Rows[1:] {
			env = append(env, row.Cells[0].Value+"="+row.Cells[1].Value)
		}
		return nil
	})
	sc.Step("cloud metadata is collected", func() error {
		_, _, _, detectedCloud, _, err := getSubprocessMetadata(append([]string{
			"ELASTIC_APM_CLOUD_PROVIDER=auto", // Explicitly enable cloud metadata detection
			"http_proxy=http://proxy.invalid", // fail all HTTP requests
		}, env...)...)
		if err != nil {
			return err
		}
		if *detectedCloud != (model.Cloud{}) {
			cloud = detectedCloud
		}
		return nil
	})
	sc.Step("cloud metadata is not null", func() error {
		if cloud == nil {
			return errors.New("cloud metadata is empty")
		}
		return nil
	})
	sc.Step("cloud metadata is null", func() error {
		if cloud != nil {
			return fmt.Errorf("cloud metadata is non-empty: %+v", *cloud)
		}
		return nil
	})
	sc.Step("cloud metadata '(.*)' is '(.*)'", func(field, expected string) error {
		var actual string
		switch field {
		case "account.id":
			if cloud.Account == nil {
				return errors.New("cloud.account is nil")
			}
			actual = cloud.Account.ID
		case "provider":
			actual = cloud.Provider
		case "instance.id":
			if cloud.Instance == nil {
				return errors.New("cloud.instance is nil")
			}
			actual = cloud.Instance.ID
		case "instance.name":
			if cloud.Instance == nil {
				return errors.New("cloud.instance is nil")
			}
			actual = cloud.Instance.Name
		case "project.name":
			if cloud.Project == nil {
				return errors.New("cloud.project is nil")
			}
			actual = cloud.Project.Name
		case "region":
			actual = cloud.Region
		default:
			return fmt.Errorf("unexpected field %q", field)
		}
		if actual != expected {
			return fmt.Errorf("expected %q to be %q, got %q", field, expected, actual)
		}
		return nil
	})
}

func anAgent() error {
	// No-op; we create the tracer in the suite setup.
	return nil
}

func anAgentConfiguredWith(settings *godog.Table) error {
	for _, row := range settings.Rows[1:] {
		setting := row.Cells[0].Value
		value := row.Cells[1].Value
		switch setting {
		case "api_key":
			apiKey = value
		case "secret_token":
			secretToken = value
		case "cloud_provider":
			env = append(env, "ELASTIC_APM_CLOUD_PROVIDER="+value)
		default:
			return fmt.Errorf("unhandled setting %q", setting)
		}
	}
	return nil
}

func sendRequest() error {
	var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		httpRequestHeaders = r.Header
	}
	server := httptest.NewServer(h)
	defer server.Close()

	os.Setenv("ELASTIC_APM_SECRET_TOKEN", secretToken)
	os.Setenv("ELASTIC_APM_API_KEY", apiKey)
	os.Setenv("ELASTIC_APM_SERVER_URL", server.URL)
	if _, err := transport.InitDefault(); err != nil {
		return err
	}
	tracer, err := apm.NewTracer("godog", "")
	if err != nil {
		return err
	}
	defer tracer.Close()

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)
	return nil
}

func checkAuthorizationHeader(expected string) error {
	authHeader := httpRequestHeaders["Authorization"]
	if n := len(authHeader); n != 1 {
		return fmt.Errorf("got %d Authorization headers, expected 1", n)
	}
	if authHeader[0] != expected {
		return fmt.Errorf("got Authorization header value %q, expected %q", authHeader, expected)
	}
	return nil
}

func anActiveSpan() error {
	if err := anActiveTransaction(); err != nil {
		return err
	}
	span = transaction.StartSpan("name", "type", nil)
	return nil
}

func anActiveTransaction() error {
	transaction = tracer.StartTransaction("name", "type")
	return nil
}

func userSetsSpanOutcome(outcome string) error {
	span.Outcome = outcome
	return nil
}

func userSetsTransactionOutcome(outcome string) error {
	transaction.Outcome = outcome
	return nil
}

func agentSetsSpanOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		span.Context.SetHTTPStatusCode(200)
	case "failure":
		span.Context.SetHTTPStatusCode(400)
	}
	return nil
}

func agentSetsTransactionOutcome(outcome string) error {
	switch outcome {
	case "unknown":
	case "success":
		transaction.Context.SetHTTPStatusCode(200)
	case "failure":
		transaction.Context.SetHTTPStatusCode(500)
	}
	return nil
}

func anErrorIsReportedToTheSpan() error {
	e := tracer.NewError(errors.New("an error"))
	e.SetSpan(span)
	return nil
}

func anErrorIsReportedToTheTransaction() error {
	e := tracer.NewError(errors.New("an error"))
	e.SetTransaction(transaction)
	return nil
}

func spanEnds() error {
	span.End()
	tracer.Flush(nil)
	return nil
}

func transactionEnds() error {
	transaction.End()
	tracer.Flush(nil)
	return nil
}

func spanOutcomeIs(expected string) error {
	payloads := tracer.Payloads()
	if actual := payloads.Spans[0].Outcome; actual != expected {
		return fmt.Errorf("expected outcome %q, got %q", expected, actual)
	}
	return nil
}

func transactionOutcomeIs(expected string) error {
	tracer.Flush(nil)
	payloads := tracer.Payloads()
	if actual := payloads.Transactions[0].Outcome; actual != expected {
		return fmt.Errorf("expected outcome %q, got %q", expected, actual)
	}
	return nil
}

func anHTTPTransactionWithStatusCode(statusCode int) error {
	if statusCode < 0 {
		httpHandler.panic = true
	} else {
		httpHandler.statusCode = statusCode
	}
	return sendHTTPRequest(context.Background())
}

func anHTTPSpanWithStatusCode(statusCode int) error {
	if statusCode < 0 {
		httpHandler.panic = true
	}
	tx := tracer.StartTransaction("name", "type")
	defer tx.End()
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	return sendHTTPRequest(ctx)
}

func sendHTTPRequest(ctx context.Context) error {
	client := apmhttp.WrapClient(httpServer.Client())
	req, _ := http.NewRequest("GET", httpServer.URL, nil)
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func aGRPCTransactionWithStatusCode(statusCode string) error {
	if statusCode == "n/a" {
		grpcService.panic = true
	} else {
		code, err := parseGRPCStatusCode(statusCode)
		if err != nil {
			return err
		}
		if code != codes.OK {
			grpcService.err = status.Error(code, code.String())
		}
	}
	pb.NewGreeterClient(grpcClient).SayHello(context.Background(), &pb.HelloRequest{Name: "world"})
	return nil
}

func aGRPCSpanWithStatusCode(statusCode string) error {
	if statusCode == "n/a" {
		grpcService.panic = true
	} else {
		code, err := parseGRPCStatusCode(statusCode)
		if err != nil {
			return err
		}
		if code != codes.OK {
			grpcService.err = status.Error(code, code.String())
		}
	}
	tx := tracer.StartTransaction("name", "type")
	defer tx.End()
	ctx := apm.ContextWithTransaction(context.Background(), tx)
	pb.NewGreeterClient(grpcClient).SayHello(ctx, &pb.HelloRequest{Name: "world"})
	return nil
}

func parseGRPCStatusCode(s string) (codes.Code, error) {
	var code codes.Code
	err := code.UnmarshalJSON([]byte(strconv.Quote(s)))
	return code, err
}

type helloworldGRPCService struct {
	panic bool
	err   error
}

func (h *helloworldGRPCService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	if h.panic {
		panic("boom")
	}
	return &pb.HelloReply{Message: "hello, " + req.Name}, h.err
}

type mockHTTPHandler struct {
	panic      bool
	statusCode int
}

func (h *mockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.panic {
		panic("boom")
	}
	statusCode := h.statusCode
	if statusCode == 100 {
		// Cheat to avoid complexity of dealing with 100 Continue responses.
		statusCode = 101
	}
	w.WriteHeader(statusCode)
}
